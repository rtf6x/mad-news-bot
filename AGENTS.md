# Agent instructions

## Infrastructure

Deploy inventory for this project lives in the sibling repo `rootfox.cc-infra` at `state/dplo/projects/mad-news-rtf6x-bot/`.

When deploy paths, ports, env, secrets refs, nginx, or dplo scripts change, update `rootfox.cc-infra` in the same change set and keep it current.

## Deploy

dplo on `ci.rootfox.cc` runs `scripts/build.sh` (build + pm2 + scheduler cron).
