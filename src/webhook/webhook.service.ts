const twilio = require('twilio');
import { Injectable } from '@nestjs/common';
import fetch from 'node-fetch';
import * as MadNews from 'mad-news';
import * as cheerio from 'cheerio';
import * as redis from 'redis';
import { promisify } from 'util';
import settings from '../settings';

// const client = new twilio(settings.twilio.sid, settings.twilio.token);
// const covid19 = 'https://coronavirus-tracker-api.herokuapp.com/v2/locations';
const covid19 = 'https://www.worldometers.info/coronavirus/';
const MessagingResponse = twilio.twiml.MessagingResponse;

const redisClient = redis.createClient(settings.redis);
const redisGet = promisify(redisClient.get).bind(redisClient);

const carAdviceProtoResults = [
  'Не бери жука, там 1.2 движок на 1.4 веса, и экологический класс D',
  'YOLO, что хочется - то и бери!',
  'Любишь кататься - люби и катайся!',
  'Какой ответ ожидаешь ты, юный падаван? Выбрать сам способен ибо сила ведёт тебя. Но остерегайся стороны тёмной влияния',
  'PSA - зло. Французы умеют делать только дизель',
  'Вам шашечки, или ехать?',
  'Машина бывает Тойота и стиральная',
  'Сухая DSG - не течёт',
];
let carAdviceResults = [];

@Injectable()
export class WebhookService {
  static async sendWAReply(): Promise<any> {
    const Madness = new MadNews().fullString.trim().replace(/\s\s/g, ' ');
    console.log(`New madness: [${Madness}]`);
    const twiml = new MessagingResponse();
    return twiml.message(Madness);
  }

  static getCovidDaysLeft(pop, cases, daily) {
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

  static async getCovid2() {
    let perDay: any = 0;
    let perDayR: any = 0;
    let perDayD: any = 0;
    let perDayDR: any = 0;
    let data = await fetch(covid19);
    data = await data.text();
    var $ = cheerio.load(data);
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

    res.a = res.c - res.d - res.r;
    res.ra = res.rc - res.rd - res.rr;

    const population = 7800000000;
    const casesPercent = Math.round(100 / (population / res.c) * 100) / 100;
    const deathsPercent = Math.round(100 / (res.c / res.d) * 100) / 100;
    const recoveryPercent = Math.round(100 / (res.c / res.r) * 100) / 100;

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
          const daysLeft = Math.floor((Date.now() - nextPrevResults.date) / (24 * 3600000));
          // если прошло больше 2х дней, ставим prev и обновляем nextPrev
          if (daysLeft > 2) {
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

    const daysLeft = WebhookService.getCovidDaysLeft(population, res.c, perDay);

    if (perDay) {
      perDay = parseInt(perDay, 10).toLocaleString('ru');
    }
    if (perDayR) {
      perDayR = parseInt(perDayR, 10).toLocaleString('ru');
    }
    if (perDayD) {
      perDayD = parseInt(perDayD, 10).toLocaleString('ru');
    }
    if (perDayDR) {
      perDayDR = parseInt(perDayDR, 10).toLocaleString('ru');
    }

    return `[Мир] Случаев: ${(res.c).toLocaleString('ru')} | Смертей: ${(res.d).toLocaleString('ru')} | Выздоровели: ${(res.r).toLocaleString('ru')} | Болеют: ${(res.a).toLocaleString('ru')}
[Россия] Случаев: ${(res.rc).toLocaleString('ru')} | Смертей: ${(res.rd).toLocaleString('ru')} | Выздоровели: ${(res.rr).toLocaleString('ru')} | Болеют: ${(res.ra).toLocaleString('ru')}

[Процент заболевших (от ${(population).toLocaleString('ru')})] ${casesPercent}%
[Процент смертей] ${deathsPercent}%
[Выздоровели] ${recoveryPercent}%

[Новых случаев (Мир)] ${perDay || 'N/A'}
[Новых случаев (Россия)] ${perDayR || 'N/A'}

[Умерли за день] ${perDayD || 'N/A'}
[Умерли за день (Россия)] ${perDayDR || 'N/A'}

[Дней осталось (с текущей заболеваемостью)] ${(daysLeft).toLocaleString('ru')}
[Умрут] ${(Math.floor(population / 100 * deathsPercent)).toLocaleString('ru')}

(Источник: worldometers)
`;
  }

  static async sendReply(req): Promise<any> {
    if (req.service && req.service === 'updateCovid') {
      const message = await WebhookService.getCovid2();
      return { status: 'success', code: 0, message };
    }

    if (!req.message || !req.message.text || !req.message.chat || !req.message.chat.id) {
      return { status: 'success', code: 0 };
    }
    console.log(`[chat message][${req.message.chat.id}]`, req.message.text);

    if (req.message.text === '/covid19' || req.message.text === '/covid19@madnews_rtf6x_bot') {
      const message = await WebhookService.getCovid2();
      if (req.message.chat && req.message.chat.id) {
        await fetch(`https://api.telegram.org/bot${settings.botId}/sendMessage?chat_id=${req.message.chat.id}&text=${encodeURIComponent(message)}`);
      }
      return { status: 'success', code: 0 };
    }

    if (req.message.text === '/madnews' || req.message.text === '/madnews@madnews_rtf6x_bot') {
      const Madness = new MadNews().fullString.trim().replace(/\s\s/g, ' ');
      console.log(`New madness: [${Madness}]`);
      await fetch(`https://api.telegram.org/bot${settings.botId}/sendMessage?chat_id=${req.message.chat.id}&text=${encodeURIComponent(Madness)}`);
    }

    if (req.message.text === '/carAdvice' || req.message.text === '/carAdvice@madnews_rtf6x_bot') {
      if (!carAdviceResults.length) {
        carAdviceResults = JSON.parse(JSON.stringify(carAdviceProtoResults));
      }
      const result = carAdviceResults.splice(Math.floor(Math.random() * carAdviceResults.length), 1)[0];
      await fetch(`https://api.telegram.org/bot${settings.botId}/sendMessage?chat_id=${req.message.chat.id}&text=${encodeURIComponent(result)}`);
    }

    if (req.message.text.toLowerCase().indexOf('спасибо')) {
      const result = 'Да не за что!';
      await fetch(`https://api.telegram.org/bot${settings.botId}/sendMessage?chat_id=${req.message.chat.id}&text=${encodeURIComponent(result)}`);
    }

    return { status: 'success', code: 0 };
  }
}
