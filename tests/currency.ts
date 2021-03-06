import fetch from 'node-fetch';
import dayjs from 'dayjs';
import 'dayjs/locale/ru';

const currenciesUrl = 'https://www.cbr-xml-daily.ru/daily_json.js';
dayjs.locale('ru');

const randomCurrencies = [
  'AUD', 'AZN', 'AMD', 'BYN', 'BGN', 'BRL', 'HUF', 'HKD', 'DKK', 'INR', 'KZT', 'CAD', 'KGS', 'CNY', 'MDL', 'NOK',
  'PLN', 'RON', 'XDR', 'SGD', 'TJS', 'TRY', 'TMT', 'UZS', 'UAH', 'CZK', 'SEK', 'CHF', 'ZAR', 'KRW',
];

const getCurrencyData = (data, currencyKey) => {
  if (!data.Valute[currencyKey]) {
    return '';
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

async function run() {
  let data = await fetch(currenciesUrl);
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
  return `[Обновлено: ${dates.updated} | Предыдущие данные: ${dates.prevDate}]

${getCurrencyData(data, 'USD')}
${getCurrencyData(data, 'EUR')}
${getCurrencyData(data, 'GBP')}
${getCurrencyData(data, 'JPY')}
${getCurrencyData(data, randomCurrencies[Math.floor(Math.random() * randomCurrencies.length)])}

(Источник: cbr-xml-daily.ru)
`;
}

(async () => {
  const message = await run();
  console.log(message);
  process.exit(1);
})();
