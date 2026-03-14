# Self-hosting quickstart

Self-hosting untils lets you run the full application stack on your own hardware.
Some knowledge of Linux server admin and Docker is required.

untils is still in development - rough edges and backwards-incompatible changes should be expected.

If you'd prefer a hosted version, one's on its way! Join the waitlist at https://untils.com
and I'll let you know when it arrives.

## Prerequisites

### System
 
- Docker with `docker compose`
- Enough RAM to comfortably run Google Chrome headless

### External APIs

- Brave Search API ([quickstart](https://api-dashboard.search.brave.com/documentation/quickstart))
- LLM provider ([OpenAI](https://developers.openai.com/api/docs/quickstart/) or [x.ai](https://docs.x.ai/developers/quickstart) API keys)
- (optional) Pushover app key for delivering push notifications ([quickstart](https://pushover.net/api))
- (optional) SMTP server for delivering emails

## 1. Download the compose and env files

Create a working directory, then download the self-hosting files from GitHub:

```sh
mkdir untils-selfhost
cd untils-selfhost
curl -O https://raw.githubusercontent.com/alexpls/untils/main/docker-compose.selfhosted.yml
curl -o .env.selfhosted https://raw.githubusercontent.com/alexpls/untils/main/.env.selfhosted.example
```

## 2. Edit `.env.selfhosted`

```sh
nano .env.selfhosted
```

Set the values for your deployment. The example file includes comments for each setting.

## 3. Start the stack

```sh
docker compose -f docker-compose.selfhosted.yml up -d
```

This starts the app, PostgreSQL, and the bundled browser service used for checks.

## 4. Sign in

Open:

```txt
http://localhost:3322/app
```

Sign in with:

- email: the `ADMIN_EMAIL` value
- password: `abc123`

If this is the first boot, **change that password immediately**.
