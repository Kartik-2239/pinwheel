
# Pinwheel

A reverse proxy written in GO for LLM APIs. It allows you to create local proxy API keys and allow users to access certain models at optional limits such as max cost and expiration date.

## How it works

- It looks for provider API keys in `.env` for OpenAI, OpenRouter, or Anthropic.
- The cli is used to setup api keys with limits and allowed models. 
- The proxy on each requests looks up the database to figure out models, providers and if the request is valid.
- Then the request is modified and sent to the provider according to the router.
- The response is then sent back to the user for both streaming and non streaming responses.


## Supported providers

- OpenRouter via `OPENROUTER_API_KEY`
- OpenAI via `OPENAI_API_KEY`
- Anthropic via `ANTHROPIC_API_KEY`
- Groq via `GROQ_API_KEY`
- Gemini via `GEMINI_API_KEY`

## Requirements
- API key for one of the supported providers
- go >= 1.25.0
- Docker and Docker Compose for the included Postgres database

## Setup

Copy a `.env.example` to `.env` and fill in your API keys:

```env
OPENAI_API_KEY=your-openai-key
OPENROUTER_API_KEY=your-openrouter-key
ANTHROPIC_API_KEY=your-anthropic-key
GROQ_API_KEY=your-groq-key
GEMINI_API_KEY=your-gemini-key

DATABASE_URL=postgres://user:pass@localhost:5432/pinwheel-db?sslmode=disable
```

## Run everything with Docker Compose

This starts both Postgres and the proxy container. Use this mode when you do not want to run the proxy with `go run`:

```sh
docker compose up -d --build
```

The containerized proxy listens on `http://localhost:8081`.

The Postgres container is exposed on `localhost:5432` for local tools such as the CLI, `psql`, or `pgweb`.

Run the CLI to create API keys with

```sh
go run cmd/cli/main.go
```

To stop containers without deleting database data:

```sh
docker compose down
```

To reset the local database completely:

```sh
docker compose down -v
docker compose up -d --build
```

## Local development with Go

The CLI and proxy both require Postgres. If you want to run the proxy with `go run`, do not run the proxy container at the same time because both try to bind `localhost:8081`.

For local Go development, start only the database container:

```sh
docker compose up -d pinwheel-db
```

Then use the host connection string because the Go process is running outside Docker:

```sh
# this is alread in .env.example
export DATABASE_URL='postgres://user:pass@localhost:5432/pinwheel-db?sslmode=disable'
```

Run the CLI or the proxy locally:

```sh
# CLI
go run cmd/cli/main.go
# Proxy
go run cmd/proxy/main.go
```

Pick one proxy runtime at a time since both of them try to bind `localhost:8081` by default.

- Docker proxy: `docker compose up -d --build`
- Local Go proxy: `docker compose up -d pinwheel-db`, then `go run cmd/proxy/main.go`


## Example request

```sh
curl http://localhost:8081/chat/completions \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer sk-your-proxy-key" \
	-d '{
		"model": "model-name",
		"messages": [{"role": "user", "content": "Hello"}]
	}'
```