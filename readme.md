# mad-news-bot

Telegram-бот: безумные новости, гороскопы, валюты, NASA APOD и плохие советы от [bad-advice-oracle](https://github.com/rtf6x/bad-advice-oracle).

## Стек

- Go 1.24 (`cmd/server`, `cmd/worker`, `cmd/scheduler`)
- Redis — кэш NASA APOD, COVID-снапшоты, очередь `/badadvice`

## Структура

```
cmd/server      — webhook API
cmd/worker      — async /badadvice
cmd/scheduler   — prefetch NASA APOD
internal/       — команды, telegram, handlers
```

## Команды бота

| Команда | Описание |
|---------|----------|
| `/madnews` | Сгенерировать безумную новость |
| `/nasaapod` | Фото дня NASA (из кэша, без ожидания API) |
| `/covid19` | Статистика COVID-19 |
| `/prograscope` | Гороскоп программиста |
| `/alcoscope` | Алкогороскоп |
| `/currency` | Курсы валют ЦБ РФ |
| `/currency USD` | Одна валюта |
| `/carAdvice` | Совет по покупке авто |
| `/badadvice <вопрос>` | Плохой совет от LLM (async) |

## Локальный запуск

```bash
cp .env.example .env
# заполнить BOT_TOKEN, REDIS_*, NASA_APOD_KEY, BAD_ADVICE_URL

make tidy

make run       # HTTP API на :8346
make worker    # обработка /badadvice
make scheduler # разовый prefetch NASA APOD
```

## NASA APOD

`/nasaapod` читает только Redis. Scheduler (`cmd/scheduler`) раз в сутки тянет NASA API и кладёт результат в ключ `nasa-apod`.

На проде cron ставится из `scripts/jenkins.sh` (06:00). Можно дернуть webhook:

```json
POST /api/webhooks/mad-news
{"service":"updateApod"}
```

## Деплой

```bash
scripts/jenkins.sh   # pm2: server + worker + cron scheduler
```

Или Docker:

```bash
docker compose up -d --build
```

## Переменные окружения

См. `.env.example`.
