# mad-news-bot

Telegram bot: mad news, horoscopes, currency rates, NASA APOD, and bad advice from [bad-advice-oracle](https://github.com/rtf6x/bad-advice-oracle).

## Stack

- Go 1.24 (`cmd/server`, `cmd/scheduler`)
- Redis db `0` — NASA APOD cache, COVID data, chat_id mapping for `/badadvice`
- Redis db `3` — bad-advice-oracle queue (`advice:queue`, pub/sub `advice:events`)

## Layout

```
cmd/server      — webhook API + advice listener
cmd/scheduler   — NASA APOD prefetch
internal/       — commands, telegram, handlers
```

## Bot commands

| Command | Description |
|---------|-------------|
| `/madnews` | Generate a mad news story |
| `/nasaapod` | NASA Astronomy Picture of the Day (from cache, no API wait) |
| `/covid19` | COVID-19 statistics |
| `/prograscope` | Programmer horoscope |
| `/alcoscope` | Drinking horoscope |
| `/currency` | CBR exchange rates |
| `/currency USD` | Single currency rate |
| `/carAdvice` | Car buying advice |
| `/badadvice <question>` | Bad advice from an LLM (async) |

## Local development

```bash
cp .env.example .env
# fill in BOT_TOKEN, REDIS_*, NASA_APOD_KEY

make tidy

make run       # HTTP API + advice listener on :8346
make scheduler # one-off NASA APOD prefetch
```

## /badadvice

The server enqueues a job in the bad-advice-oracle Redis queue (`advice:queue`, db `3`)
and listens to `advice:events` in the same process, sending the reply to Telegram.

## NASA APOD

`/nasaapod` reads from Redis only. The scheduler (`cmd/scheduler`) fetches NASA API once a day and stores the result under the `nasa-apod` key.

In production, cron is configured by `scripts/jenkins.sh` (06:00). You can also trigger an update via webhook:

```json
POST /api/webhooks/mad-news
{"service":"updateApod"}
```

## Deployment

In production, the project is built and deployed on the server by **Jenkins**. Job Execute shell:

```bash
#!/bin/bash
./scripts/jenkins.sh
```

`scripts/jenkins.sh` builds binaries, restarts the server via pm2, and sets up a cron job for the scheduler (06:00).

Locally (without Jenkins):

```bash
scripts/jenkins.sh
```

Or with Docker:

```bash
docker compose up -d --build
```

## Environment variables

See `.env.example`.
