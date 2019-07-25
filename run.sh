#!/usr/bin/env bash
BOT_TOKEN=Your_BOT_Token
until ./go-discord-ctftime "$BOT_TOKEN" > go-discord-ctftime.log 2>&1; do
  echo "Process crashed with exit code $?. Respawning.." >&2
  sleep 1
done
