import fetch from 'node-fetch';
import * as cheerio from 'cheerio';
import * as redis from 'redis';
import { promisify } from 'util';
import settings from '../settings';

const redisClient = redis.createClient(settings.redis);
const redisGet = promisify(redisClient.get).bind(redisClient);

// const client = new twilio(settings.twilio.sid, settings.twilio.token);
// const covid19 = 'https://coronavirus-tracker-api.herokuapp.com/v2/locations';
const covid19 = 'https://www.worldometers.info/coronavirus/';

function getCovidDaysLeft(pop, cases, daily) {
  console.log(`[getCovidDaysLeft] ${pop} ${cases} ${daily}`);
  if (!daily || !cases) {
    return 'N/A';
  }
  let cs = cases;
  let dailyMultiplier = Math.floor(cases / daily);
  // console.log('dailyMultiplier (initial):', dailyMultiplier);
  let days = 0;
  while (cs < pop) {
    cs = Math.floor(cs + (cs / dailyMultiplier));
    dailyMultiplier = Math.floor(cs / daily);
    // console.log('cs:', cs);
    // console.log('dailyMultiplier:', dailyMultiplier);
    days++;
  }
  return days;
}

export default async function covid() {
  let data: any = await fetch(covid19);
  data = await data.text();
  const $ = cheerio.load(data);
  const res: any = {
    a: 0,
    c: 0,
    d: 0,
    r: 0,
    ra: 0,
    rc: 0,
    rd: 0,
    rr: 0,
    nc: 0,
  };
  $('.maincounter-number span').each((i, item) => {
    if (i > 2) {
      return;
    }
    res[['c', 'd', 'r'][i]] = parseInt($(item).text().trim().replace(/,/g, ''), 10);
  });
  $('#main_table_countries_today tbody tr').each((i, item) => {
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

  res.a = res.c - res.d - res.r;
  res.ra = res.rc - res.rd - res.rr;

  const population = 7800000000;
  const casesPercent = Math.round(100 / (population / res.c) * 100) / 100;
  const deathsPercent = Math.round(100 / (res.c / res.d) * 100) / 100;

  let prevResults = await redisGet('prev');
  if (!prevResults) {
    redisClient.setex('prev', 14 * 24 * 3600, JSON.stringify({
      date: Date.now(),
      a: res.a,
      c: res.c,
      d: res.d,
      r: res.r,
      ra: res.ra,
      rc: res.rc,
      rd: res.rd,
      rr: res.rr,
      nc: res.nc,
    }));
  } else {
    prevResults = JSON.parse(prevResults);
    const daysLeft1 = Math.floor((Date.now() - prevResults.date) / (24 * 3600000));
    if (daysLeft1 > 2) {
      let nextPrevResults = await redisGet('nextPrev');
      console.log('nextPrevResults', nextPrevResults);
      if (!nextPrevResults) {
        redisClient.setex('nextPrev', 14 * 24 * 3600, JSON.stringify({
          date: Date.now(),
          a: res.a,
          c: res.c,
          d: res.d,
          r: res.r,
          ra: res.ra,
          rc: res.rc,
          rd: res.rd,
          rr: res.rr,
          nc: res.nc,
        }));
      } else {
        nextPrevResults = JSON.parse(nextPrevResults);
        const daysLeft2 = Math.floor((Date.now() - nextPrevResults.date) / (24 * 3600000));
        // если прошло больше 2х дней, ставим prev и обновляем nextPrev
        if (daysLeft2 > 2) {
          redisClient.setex('prev', 14 * 24 * 3600, JSON.stringify(nextPrevResults));
          redisClient.setex('nextPrev', 14 * 24 * 3600, JSON.stringify({
            date: Date.now(),
            a: res.a,
            c: res.c,
            d: res.d,
            r: res.r,
            ra: res.ra,
            rc: res.rc,
            rd: res.rd,
            rr: res.rr,
            nc: res.nc,
          }));
        }
      }
    }
  }

  // tslint:disable-next-line:max-line-length
  return `[Данные по SARS-Cov19 на 13 Апреля 2024]

[Мир] Случаев: ${(res.c).toLocaleString('ru')}
[Мир] Смертей: ${(res.d).toLocaleString('ru')}

[Россия] Случаев: ${(res.rc).toLocaleString('ru')}
[Россия] Смертей: ${(res.rd).toLocaleString('ru')}

[Процент заболевших (от ${(population).toLocaleString('ru')})] ${casesPercent}%
[Смертность] ${deathsPercent}%

(Источник: worldometers)
`;
}
