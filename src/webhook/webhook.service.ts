const twilio = require('twilio');
import { Injectable } from '@nestjs/common';
import fetch from 'node-fetch';
import MadNews from 'mad-news';
import getCovid19 from './getCovid19';
import currency from './currency';
import carAdvice from './carAdvice';
import settings from '../settings';
import getPrograscope from './getPrograScope';

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

  static async sendHireMe(body): Promise<any> {
    console.log('body', body);
    if (!body.history?.length) {
      return { status: 'error', code: 1 };
    }
    let message = `Points: ${body.points}\n`;
    body.history.forEach(historyItem => {
      message += `${historyItem.question}: ${historyItem.answer}\n`;
    });
    await fetch(`https://api.telegram.org/bot${settings.botId}/sendMessage?chat_id=324702279&text=${encodeURIComponent(message)}`);
    return { status: 'success', code: 0 };
  }

  static async sendReply(body): Promise<any> {
    if (body.service && body.service === 'updateCovid') {
      const message = await getCovid19();
      return { status: 'success', code: 0, message };
    }

    if (!body.message || !body.message.text || !body.message.chat || !body.message.chat.id) {
      return { status: 'success', code: 0 };
    }
    const senderId = body.message.from?.id || 0;
    console.log(`[chat message][${body.message.chat.id}][${body.message.from?.id || 0}]`, body.message.text);

    if (body.message.text === '/prograscope' || body.message.text === '/prograscope@madnews_rtf6x_bot') {
      const message = getPrograscope(senderId);
      if (body.message.chat && body.message.chat.id) {
        await fetch(`https://api.telegram.org/bot${settings.botId}/sendMessage?chat_id=${body.message.chat.id}&text=${encodeURIComponent(message)}`);
      }
      return { status: 'success', code: 0 };
    }

    if (body.message.text === '/covid19' || body.message.text === '/covid19@madnews_rtf6x_bot') {
      const message = await getCovid19();
      if (body.message.chat && body.message.chat.id) {
        await fetch(`https://api.telegram.org/bot${settings.botId}/sendMessage?chat_id=${body.message.chat.id}&text=${encodeURIComponent(message)}`);
      }
      return { status: 'success', code: 0 };
    }

    if (body.message.text === '/currency' || body.message.text === '/currency@madnews_rtf6x_bot') {
      const message = await currency();
      if (body.message.chat && body.message.chat.id) {
        await fetch(`https://api.telegram.org/bot${settings.botId}/sendMessage?chat_id=${body.message.chat.id}&text=${encodeURIComponent(message)}`);
      }
      return { status: 'success', code: 0 };
    }

    if (body.message.text.indexOf('/currency ') === 0 || body.message.text.indexOf('/currency@madnews_rtf6x_bot ') === 0) {
      const currencyKey = body.message.text.split(' ')[1].toUpperCase();
      const message = await currency(currencyKey);
      if (body.message.chat && body.message.chat.id) {
        await fetch(`https://api.telegram.org/bot${settings.botId}/sendMessage?chat_id=${body.message.chat.id}&text=${encodeURIComponent(message)}`);
      }
      return { status: 'success', code: 0 };
    }

    if (body.message.text === '/madnews' || body.message.text === '/madnews@madnews_rtf6x_bot') {
      Madness.generate();
      const mad = Madness.fullString;
      console.log(`New madness: [${mad}]`);
      await fetch(`https://api.telegram.org/bot${settings.botId}/sendMessage?chat_id=${body.message.chat.id}&text=${encodeURIComponent(mad)}`);
    }

    if (body.message.text === '/carAdvice' || body.message.text === '/carAdvice@madnews_rtf6x_bot') {
      const message = await carAdvice();
      if (body.message.chat && body.message.chat.id) {
        await fetch(`https://api.telegram.org/bot${settings.botId}/sendMessage?chat_id=${body.message.chat.id}&text=${encodeURIComponent(message)}`);
      }
      return { status: 'success', code: 0 };
    }

    return { status: 'success', code: 0 };
  }
}
