import fetch from 'node-fetch';
import * as cheerio from 'cheerio';
import * as redis from 'redis';
import { promisify } from 'util';
import { WebhookService } from '../src/webhook/webhook.service'
import settings from "../src/settings";

const covid19 = 'https://www.worldometers.info/coronavirus/';

const redisClient = redis.createClient(settings.redis);
const redisGet = promisify(redisClient.get).bind(redisClient);

async function run() {
  let perDay = 0;
  let perDayR = 0;
  let perDayD = 0;
  let perDayDR = 0;
  let data = await fetch(covid19);
  data = await data.text();
  var $ = cheerio.load(data);
  const res: any = {
    c: 0,
    d: 0,
    r: 0,
    rc: 0,
    rd: 0,
    rr: 0,
    nc: 0,
  };
  $('.maincounter-number span').each(function (i, item) {
    if (i > 2) {
      return;
    }
    res[['c', 'd', 'r'][i]] = parseInt($(item).text().trim().replace(/,/g, ''), 10);
  });
  $('#main_table_countries_today tbody tr').each(function (i, item) {
    if ($('td:nth-child(2)', item).text() === 'World') {
      res.nc = parseInt($('td:nth-child(4)', item).text().trim().replace(/,/g, '') || '0', 10);
      return;
    }
    if ($('td:nth-child(2) a', item).text() !== 'Russia') {
      return;
    }
    res.rc = parseInt($('td:nth-child(3)', item).text().trim().replace(/,/g, ''), 10);
    res.rd = parseInt($('td:nth-child(5)', item).text().trim().replace(/,/g, ''), 10);
    res.rr = parseInt($('td:nth-child(7)', item).text().trim().replace(/,/g, ''), 10);
  });

  const population = 7800000000;
  const casesPercent = Math.round(100 / (population / res.c) * 100) / 100;
  const deathsPercent = Math.round(100 / (res.c / res.d) * 100) / 100;
  const recoveryPercent = Math.round(100 / (res.c / res.r) * 100) / 100;

  let prevResults = await redisGet('prev');
  console.log('prevResults', prevResults);
  if (!prevResults) {
    redisClient.setex('prev', 14 * 24 * 3600, JSON.stringify({
      date: Date.now(),
      c: res.c,
      d: res.d,
      r: res.r,
      rc: res.rc,
      rd: res.rd,
      rr: res.rr,
      nc: res.nc,
    }));
  } else {
    prevResults = JSON.parse(prevResults);
    const daysLeft = Math.floor((Date.now() - prevResults.date) / (24 * 3600000));
    perDay = res.c - prevResults.c;
    perDayR = res.rc - prevResults.rc;
    perDayD = res.d - prevResults.d;
    perDayDR = res.rd - prevResults.rd;
    if (daysLeft) {
      perDay = Math.floor(perDay / daysLeft);
      perDayR = Math.floor(perDayR / daysLeft);
      perDayD = Math.floor(perDayD / daysLeft);
      perDayDR = Math.floor(perDayDR / daysLeft);
    }
    if (daysLeft > 2) {
      let nextPrevResults = await redisGet('nextPrev');
      console.log('nextPrevResults', nextPrevResults);
      if (!nextPrevResults) {
        redisClient.setex('nextPrev', 14 * 24 * 3600, JSON.stringify({
          date: Date.now(),
          c: res.c,
          d: res.d,
          r: res.r,
          rc: res.rc,
          rd: res.rd,
          rr: res.rr,
          nc: res.nc,
        }));
      } else {
        nextPrevResults = JSON.parse(nextPrevResults);
        const daysLeft = Math.floor((Date.now() - nextPrevResults.date) / (24 * 3600000));
        // если прошло больше 2х дней, ставим prev и обновляем nextPrev
        if (daysLeft > 2) {
          redisClient.setex('prev', 14 * 24 * 3600, JSON.stringify(nextPrevResults));
          redisClient.setex('nextPrev', 14 * 24 * 3600, JSON.stringify({
            date: Date.now(),
            c: res.c,
            d: res.d,
            r: res.r,
            rc: res.rc,
            rd: res.rd,
            rr: res.rr,
            nc: res.nc,
          }));
        }
      }
    }
  }

  return `[Мир] Случаев: ${(res.c).toLocaleString('ru')} | Смертей: ${(res.d).toLocaleString('ru')} | Выздоровели: ${(res.r).toLocaleString('ru')}
[Россия] Случаев: ${(res.rc).toLocaleString('ru')} | Смертей: ${(res.rd).toLocaleString('ru')} | Выздоровели: ${(res.rr).toLocaleString('ru')}

[Процент заболевших от общего населения (${(population).toLocaleString('ru')})] ${casesPercent}%
[Процент смертей] ${deathsPercent}%
[Выздоровели] ${recoveryPercent}%

[Умрут (приблизительно, от общего населения)] ${(Math.floor(population / 100 * deathsPercent)).toLocaleString('ru')}

[В день] ${perDay || 'N/A'}
[В день (Россия)] ${perDayR || 'N/A'}

[Умерли за день] ${perDayD || 'N/A'}
[Умерли за день (Россия)] ${perDayDR || 'N/A'}

[Дней осталось (с такой заболеваемостью в день)] ${(WebhookService.getCovidDaysLeft(population, res.c, perDay)).toLocaleString('ru')}

(Источник: worldometers.info/coronavirus/)
`;
}

(async () => {
  const message = await run();
  console.log(message);
  process.exit(1);
})();
