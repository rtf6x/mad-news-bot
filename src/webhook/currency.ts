import fetch from 'node-fetch';
import * as dayjs from 'dayjs';
import 'dayjs/locale/ru';

const currenciesUrl = 'https://www.cbr-xml-daily.ru/daily_json.js';
dayjs.locale('ru');

const randomCurrencies = [
  'AUD', 'AZN', 'AMD', 'BYN', 'BGN', 'BRL', 'HUF', 'HKD', 'DKK', 'INR', 'KZT', 'CAD', 'KGS', 'CNY', 'MDL', 'NOK',
  'PLN', 'RON', 'XDR', 'SGD', 'TJS', 'TRY', 'TMT', 'UZS', 'UAH', 'CZK', 'SEK', 'CHF', 'ZAR', 'KRW',
];

const getCurrencyData = (data, currencyKey) => {
  if (!data.Valute[currencyKey]) {
    return `[${currencyKey}] Нет данных по валюте!`;
  }
  const result = {
    cur: data.Valute[currencyKey].Value,
    isRise: data.Valute[currencyKey].Value > data.Valute[currencyKey].Previous ? 'опухло' : 'сдулось',
    change: Number((data.Valute[currencyKey].Value - data.Valute[currencyKey].Previous).toFixed(2)),
  };
  if (result.change < 0) {
    result.change = result.change * -1;
  }
  let changed = 'не изменилось';
  if (result.change !== 0) {
    changed = `${result.isRise} на ${result.change}`;
  }
  const name = data.Valute[currencyKey].Nominal > 1 ?
    `${data.Valute[currencyKey].Nominal} ${data.Valute[currencyKey].Name}` :
    data.Valute[currencyKey].Name;
  return `[${currencyKey}][${name}] ${result.cur} (${changed})`;
};

export default async function currency(currencyKey = null) {
  let data: any = await fetch(currenciesUrl);
  data = await data.json();
  if (
    !data.Valute ||
    !data.Valute.USD ||
    !data.Valute.EUR ||
    !data.Valute.USD.Value ||
    !data.Valute.USD.Previous ||
    !data.Valute.EUR.Value ||
    !data.Valute.EUR.Previous
  ) {
    return 'Не могу получить данные с cbr-xml-daily ;(';
  }
  const dates = {
    updated: dayjs(data.Date || 0).format('DD.MM.YYYY (HH:mm)'),
    prevDate: dayjs(data.PreviousDate || 0).format('DD.MM.YYYY (HH:mm)'),
  };
  let currencies;
  if (currencyKey) {
    currencies = `${getCurrencyData(data, currencyKey)}`;
  } else {
    currencies = `${getCurrencyData(data, 'USD')}
${getCurrencyData(data, 'EUR')}
${getCurrencyData(data, 'GBP')}
${getCurrencyData(data, 'JPY')}
${getCurrencyData(data, randomCurrencies[Math.floor(Math.random() * randomCurrencies.length)])}`;
  }
  return `[Обновлено: ${dates.updated} | Предыдущие данные: ${dates.prevDate}]

${currencies}

(Источник: cbr-xml-daily.ru)
`;
}
