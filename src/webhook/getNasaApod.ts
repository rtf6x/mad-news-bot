import fetch from 'node-fetch';
import * as redis from 'redis';
import { promisify } from 'util';
import settings from '../settings';

const redisClient = redis.createClient(settings.redis);
const redisGet = promisify(redisClient.get).bind(redisClient);

const apodUrl = 'https://api.nasa.gov/planetary/apod?api_key=DEMO_KEY';

export default async function getNasaApod() {
  let res = await redisGet('nasa-apod');
  if (!res) {
    let data: any = await fetch(apodUrl);
    data = await data.json();
    res = {
      copyright: data.copyright,
      date: data.date,
      explanation: data.explanation,
      hdurl: data.hdurl,
      media_type: data.media_type,
      service_version: data.service_version,
      title: data.title,
      url: data.url,
    };
    redisClient.setex('nasa-apod', 24 * 3600, JSON.stringify(res));
  } else {
    res = JSON.parse(res);
  }

  // tslint:disable-next-line:max-line-length
  return {
    photo: res.hdurl,
    message: `${res.title} (${res.date})

${res.explanation}

(c) ${res.copyright}
`,
  };
}
