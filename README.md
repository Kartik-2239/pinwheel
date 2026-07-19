
# OpenAI Proxy

A reverse proxy written in GO for LLM APIs. It allows you to create local proxy API keys and allow users to access certain models at optional limits such as max cost and expiration date.

## How it works

- It looks for provider API keys in `.env` for OpenAI, OpenRouter, or Anthropic.
- The cli is used to setup api keys with limits and allowed models. 
- The proxy on each requests looks up the database to figure out models, providers and if the request is valid. Then the request is modified and sent to the provider.
- The response is then sent back to the user for streaming and non streaming responses.


## Supported providers

- OpenRouter via `OPENROUTER_API_KEY`
- OpenAI via `OPENAI_API_KEY`
- Anthropic via `ANTHROPIC_API_KEY`
- Groq via `GROQ_API_KEY`
- Gemini via `GEMINI_API_KEY`

## Requirements
- API key for one of the supported providers
- go >= 1.25.0

## Setup

Create a `.env` file with at least one provider key:

```env
OPENAI_API_KEY=your-openai-key
OPENROUTER_API_KEY=your-openrouter-key
ANTHROPIC_API_KEY=your-anthropic-key
PROXY_DB_PATH=proxy.db
```

`PROXY_DB_PATH` is optional and defaults to `proxy.db`.

## Create a proxy API key

```sh
go run ./cmd/cli
```

Use the TUI to:

1. Enter a name for the key.
2. Select allowed models.
3. Choose optional limits.
4. Copy the generated key when it is printed.

## Run the proxy

```sh
go run ./cmd/proxy
```

The proxy listens on `http://localhost:8081`.

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