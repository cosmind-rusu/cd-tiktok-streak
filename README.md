# cd-tiktok-streak

`cd-tiktok-streak` is a small Go + Playwright utility that keeps a TikTok message streak alive by sending a short message to one or more configured conversations.

It uses exported TikTok cookies instead of a username/password login flow, opens the TikTok inbox, finds the target conversation, and sends a message such as `"."`.

## What the project does

- Loads TikTok session cookies from `cookies.json`
- Opens the TikTok messages page with Playwright
- Searches for the configured conversation
- Matches the visible TikTok display nickname, not just the raw username
- Sends a configurable message
- Supports `run once` mode and optional daily scheduling

## Current stack

- Go
- Playwright for Go
- Chromium / Chrome via Playwright

## Project layout

- [main.go](C:/Users/cosmi/Desktop/cd-tiktok-streak/main.go:1): main application logic
- [config.example.json](C:/Users/cosmi/Desktop/cd-tiktok-streak/config.example.json:1): example configuration
- [Dockerfile](C:/Users/cosmi/Desktop/cd-tiktok-streak/Dockerfile:1): container image for running the bot
- [docker-compose.yml](C:/Users/cosmi/Desktop/cd-tiktok-streak/docker-compose.yml:1): recommended Docker deployment

## Requirements

- Go `1.25+`
- Playwright runtime installed for the version pinned in `go.mod`
- A valid `cookies.json` exported from your own TikTok session

## Setup

1. Install Go dependencies:

```bash
go mod tidy
```

2. Install Playwright for the version used by this project:

```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install
```

3. Create your config file:

```bash
copy config.example.json config.json
```

4. Export your TikTok cookies to `cookies.json`.

5. Edit `config.json` and set your target users and message.

## Configuration

Example:

```json
{
  "run_once": true,
  "headless": false,
  "target_users": ["dilan_samayoa"],
  "message": ".",
  "schedule": {
    "enabled": false,
    "time": "00:02"
  },
  "cookies_file": "cookies.json",
  "log_file": "cd-tiktok-streak.log",
  "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
  "messages_url": "https://www.tiktok.com/messages",
  "browser_channel": "",
  "locale": "en-US",
  "timezone_id": "Europe/Madrid"
}
```

Important fields:

- `run_once`: run immediately and exit
- `headless`: show or hide the browser window
- `target_users`: list of usernames or conversation targets
- `message`: the text to send
- `schedule.enabled`: keep the process running and send daily at `schedule.time`
- `schedule.time`: daily execution time in `HH:MM` 24-hour format
- `cookies_file`: exported TikTok cookies file
- `log_file`: log output file
- `browser_channel`: optional Playwright browser channel; leave it empty in Docker, Railway, or Dokploy so the bundled Chromium is used
- `timezone_id`: IANA timezone used by the browser and the scheduler, for example `Europe/Madrid`, `America/Mexico_City`, or `America/New_York`

Relative paths such as `cookies.json` and `cd-tiktok-streak.log` are resolved from the folder where your `config.json` lives.

Daily scheduler example:

```json
{
  "run_once": false,
  "schedule": {
    "enabled": true,
    "time": "21:30"
  },
  "timezone_id": "Europe/Madrid"
}
```

With that config, the bot stays running and sends the message every day at `21:30` in the configured timezone, even if the Docker host is using a different timezone.

## Usage

Run once with Go:

```bash
go run . -config config.json -run-once
```

Run once with the compiled binary:

```bash
.\cd-tiktok-streak.exe -config config.json -run-once
```

Build the binary:

```bash
go build -o cd-tiktok-streak .
```

## Docker

Recommended layout:

```text
docker-data/
  config.json
  cookies.json
  cd-tiktok-streak.log
```

Prepare the data folder:

```bash
mkdir docker-data
copy config.example.json docker-data\config.json
```

Put your exported TikTok cookies into `docker-data/cookies.json`, then edit `docker-data/config.json`.

If you keep `"cookies_file": "cookies.json"` and `"log_file": "cd-tiktok-streak.log"` in the config, Docker will resolve both files inside `docker-data/`.

Build and run with Docker Compose:

```bash
docker compose up -d --build
```

Stop it:

```bash
docker compose down
```

If you prefer plain `docker run`, use the same mounted folder approach:

```bash
docker build -t cd-tiktok-streak .
```

```bash
docker run --rm ^
  -v ${PWD}\docker-data:/data ^
  cd-tiktok-streak
```

The container entrypoint now reads `/data/config.json` automatically. It can also create `config.json` and `cookies.json` from environment variables, which is useful on PaaS platforms.

## Railway and Dokploy

Recommended deployment path:

- Railway: use the root `Dockerfile`. Railway auto-detects a root `Dockerfile` and uses it during deployment.
- Dokploy: use `Dockerfile` in production, or `nixpacks.toml` if you choose the Nixpacks build type in the panel.

The project includes [deploy/start.sh](C:/Users/cosmi/Desktop/cd-tiktok-streak/deploy/start.sh:1), which supports both mounted files and environment variables:

- `CDRUSU_CONFIG_JSON`: raw JSON for `config.json`
- `CDRUSU_CONFIG_JSON_B64`: base64-encoded `config.json`
- `CDRUSU_COOKIES_JSON`: raw JSON for `cookies.json`
- `CDRUSU_COOKIES_JSON_B64`: base64-encoded `cookies.json`
- `CDRUSU_DATA_DIR`: optional data directory, defaults to `/data`

Recommended PaaS setup:

1. Set `CDRUSU_CONFIG_JSON_B64` and `CDRUSU_COOKIES_JSON_B64` as secret environment variables.
2. In `config.json`, keep `"cookies_file": "cookies.json"` and `"log_file": "cd-tiktok-streak.log"`.
3. Set `"run_once": false`, enable `schedule.enabled`, and choose `schedule.time` plus `timezone_id`.

Example worker config for Railway or Dokploy:

```json
{
  "run_once": false,
  "headless": true,
  "target_users": ["username1"],
  "message": ".",
  "schedule": {
    "enabled": true,
    "time": "21:30"
  },
  "cookies_file": "cookies.json",
  "log_file": "cd-tiktok-streak.log",
  "browser_channel": "",
  "timezone_id": "Europe/Madrid"
}
```

PowerShell helper to generate the base64 secrets:

```powershell
[Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes((Get-Content .\docker-data\config.json -Raw)))
[Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes((Get-Content .\docker-data\cookies.json -Raw)))
```

Nixpacks support is included in [nixpacks.toml](C:/Users/cosmi/Desktop/cd-tiktok-streak/nixpacks.toml:1). It installs Node, downloads Playwright Chromium, and starts the app through the same `deploy/start.sh` script.

## Notes

- This project depends on TikTok's current inbox UI. If TikTok changes selectors or layout, parts of the automation may need to be updated.
- `headless: false` is useful while tuning selectors or verifying login state.
- `cookies.json` should never be shared or committed.
- The current Playwright Go package works, but its ecosystem is weaker than official Playwright for Node.

## Safety

Use only your own TikTok account and your own exported cookies. A valid cookies file gives session-level access to the account while it remains valid.
