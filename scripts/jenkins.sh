#!/bin/bash
# Jenkins → Execute shell
set -euo pipefail
export PATH=/usr/local/go/bin:$PATH

: "${BOT_TOKEN:?BOT_TOKEN must be set in the Jenkins job environment}"
: "${REDIS_PASSWORD:?REDIS_PASSWORD must be set in the Jenkins job environment}"
: "${NASA_APOD_KEY:?NASA_APOD_KEY must be set in the Jenkins job environment}"

APP_NAME="mad-news-bot"
LEGACY_WORKER_NAME="mad-news-bot-worker"

export APP_ADDR="127.0.0.1:8346"
export BOT_TOKEN
export BOT_USERNAME="madnews_rtf6x_bot"
export HIRE_CHAT_ID="324702279"
export REDIS_ADDR="159.69.113.123:63719"
export REDIS_PASSWORD
export REDIS_DB="0"
export ORACLE_REDIS_DB="3"
export NASA_APOD_KEY
export ADVICE_JOB_TTL_SEC="120"
export CHAT_LOG_DIR="$HOME/mad-news-bot-chat-logs"

go build -ldflags="-s -w" -o mad-news-bot-server ./cmd/server
go build -ldflags="-s -w" -o mad-news-bot-scheduler ./cmd/scheduler

pm2 stop --silent "$LEGACY_WORKER_NAME" || true
pm2 delete --silent "$LEGACY_WORKER_NAME" || true

if pm2 describe "$APP_NAME" >/dev/null 2>&1; then
  pm2 restart "$APP_NAME" --update-env
else
  pm2 start ./mad-news-bot-server --name "$APP_NAME"
fi

pm2 reset "$APP_NAME"
pm2 flush "$APP_NAME"
pm2 save

./mad-news-bot-scheduler

SCHEDULER_CMD="cd $(pwd) && REDIS_ADDR=$REDIS_ADDR REDIS_PASSWORD=$REDIS_PASSWORD REDIS_DB=$REDIS_DB NASA_APOD_KEY=$NASA_APOD_KEY ./mad-news-bot-scheduler >> $HOME/mad-news-bot-scheduler.log 2>&1"
(crontab -l 2>/dev/null | grep -v mad-news-bot-scheduler; echo "0 6 * * * $SCHEDULER_CMD") | crontab -
