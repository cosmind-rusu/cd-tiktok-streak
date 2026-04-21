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

- [main.go](C:/Users/cosmi/Desktop/TikTok-Streak-Bot/main.go:1): main application logic
- [config.example.json](C:/Users/cosmi/Desktop/TikTok-Streak-Bot/config.example.json:1): example configuration
- [Dockerfile](C:/Users/cosmi/Desktop/TikTok-Streak-Bot/Dockerfile:1): container image for running the bot

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
  "browser_channel": "chrome",
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
- `cookies_file`: exported TikTok cookies file
- `log_file`: log output file

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

Build the image:

```bash
docker build -t cd-tiktok-streak .
```

Run it:

```bash
docker run --rm ^
  -v ${PWD}\config.json:/app/config.json ^
  -v ${PWD}\cookies.json:/app/cookies.json ^
  -v ${PWD}\cd-tiktok-streak.log:/app/cd-tiktok-streak.log ^
  cd-tiktok-streak
```

## Notes

- This project depends on TikTok's current inbox UI. If TikTok changes selectors or layout, parts of the automation may need to be updated.
- `headless: false` is useful while tuning selectors or verifying login state.
- `cookies.json` should never be shared or committed.
- The current Playwright Go package works, but its ecosystem is weaker than official Playwright for Node.

## Safety

Use only your own TikTok account and your own exported cookies. A valid cookies file gives session-level access to the account while it remains valid.
