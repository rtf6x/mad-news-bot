# mad-news-bot

Telegram-бот: безумные новости, гороскопы, валюты, NASA APOD и плохие советы от [bad-advice-oracle](https://github.com/rtf6x/bad-advice-oracle).

## Стек

- Go 1.24 (`cmd/server`, `cmd/scheduler`)
- Redis db `0` — кэш NASA APOD, COVID, привязка chat_id для `/badadvice`
- Redis db `3` — очередь bad-advice-oracle (`advice:queue`, pub/sub `advice:events`)

## Структура

```
cmd/server      — webhook API + advice listener
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
# заполнить BOT_TOKEN, REDIS_*, NASA_APOD_KEY

make tidy

make run       # HTTP API + advice listener на :8346
make scheduler # разовый prefetch NASA APOD
```

## /badadvice

Server кладёт задачу в Redis-очередь bad-advice-oracle (`advice:queue`, db `3`)
и в том же процессе слушает `advice:events`, отправляя ответ в Telegram.

## NASA APOD

`/nasaapod` читает только Redis. Scheduler (`cmd/scheduler`) раз в сутки тянет NASA API и кладёт результат в ключ `nasa-apod`.

На проде cron ставится из `scripts/jenkins.sh` (06:00). Можно дернуть webhook:

```json
POST /api/webhooks/mad-news
{"service":"updateApod"}
```

## Деплой

```bash
scripts/jenkins.sh   # pm2: server + cron scheduler
```

Или Docker:

```bash
docker compose up -d --build
```

## Переменные окружения

См. `.env.example`.
