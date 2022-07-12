const twilio = require('twilio');
import { Injectable } from '@nestjs/common';
import fetch from 'node-fetch';
import * as MadNews from 'mad-news';
import getCovid19 from './getCovid19';
import currency from './currency';
import carAdvice from './carAdvice';
import settings from '../settings';

// @ts-ignore
const Madness = new MadNews('ru');

const MessagingResponse = twilio.twiml.MessagingResponse;

@Injectable()
export class WebhookService {
  static async sendWAReply(): Promise<any> {
    Madness.generate();
    const mad = Madness.fullString.trim().replace(/\s\s/g, ' ');
    console.log(`New madness: [${mad}]`);
    const twiml = new MessagingResponse();
    return twiml.message(mad);
  }

  static async sendReply(req): Promise<any> {
    if (req.service && req.service === 'updateCovid') {
      const message = await getCovid19();
      return { status: 'success', code: 0, message };
    }

    if (!req.message || !req.message.text || !req.message.chat || !req.message.chat.id) {
      return { status: 'success', code: 0 };
    }
    console.log(`[chat message][${req.message.chat.id}]`, req.message.text);

    if (req.message.text === '/covid19' || req.message.text === '/covid19@madnews_rtf6x_bot') {
      const message = await getCovid19();
      if (req.message.chat && req.message.chat.id) {
        await fetch(`https://api.telegram.org/bot${settings.botId}/sendMessage?chat_id=${req.message.chat.id}&text=${encodeURIComponent(message)}`);
      }
      return { status: 'success', code: 0 };
    }

    if (req.message.text === '/currency' || req.message.text === '/currency@madnews_rtf6x_bot') {
      const message = await currency();
      if (req.message.chat && req.message.chat.id) {
        await fetch(`https://api.telegram.org/bot${settings.botId}/sendMessage?chat_id=${req.message.chat.id}&text=${encodeURIComponent(message)}`);
      }
      return { status: 'success', code: 0 };
    }

    if (req.message.text.indexOf('/currency ') === 0 || req.message.text.indexOf('/currency@madnews_rtf6x_bot ') === 0) {
      const currencyKey = req.message.text.split(' ')[1].toUpperCase();
      const message = await currency(currencyKey);
      if (req.message.chat && req.message.chat.id) {
        await fetch(`https://api.telegram.org/bot${settings.botId}/sendMessage?chat_id=${req.message.chat.id}&text=${encodeURIComponent(message)}`);
      }
      return { status: 'success', code: 0 };
    }

    if (req.message.text === '/madnews' || req.message.text === '/madnews@madnews_rtf6x_bot') {
      Madness.generate();
      const mad = Madness.fullString;
      console.log(`New madness: [${mad}]`);
      await fetch(`https://api.telegram.org/bot${settings.botId}/sendMessage?chat_id=${req.message.chat.id}&text=${encodeURIComponent(mad)}`);
    }

    if (req.message.text === '/carAdvice' || req.message.text === '/carAdvice@madnews_rtf6x_bot') {
      const message = await carAdvice();
      if (req.message.chat && req.message.chat.id) {
        await fetch(`https://api.telegram.org/bot${settings.botId}/sendMessage?chat_id=${req.message.chat.id}&text=${encodeURIComponent(message)}`);
      }
      return { status: 'success', code: 0 };
    }

    return { status: 'success', code: 0 };
  }
}
