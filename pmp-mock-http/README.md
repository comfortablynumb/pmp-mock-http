# PMP Mock HTTP - Mock Configurations

This directory contains mock configurations for this repository. It follows the standard plugin structure used by PMP Mock HTTP.

## Directory Structure

```
pmp-mock-http/
└── openai/          # OpenAI API mocks
    ├── chat-completions.yaml
    ├── completions.yaml
    ├── embeddings.yaml
    └── models.yaml
```

## OpenAI Mocks

The `openai/` directory contains mocks for the OpenAI API:

- **chat-completions.yaml**: Chat completions endpoint (streaming and non-streaming)
- **completions.yaml**: Text completions endpoint with rate limiting examples
- **embeddings.yaml**: Embeddings endpoint
- **models.yaml**: Model listing and retrieval endpoints

### Usage

To use these mocks with pmp-mock-http:

```bash
# Run the server (will automatically load mocks from pmp-mock-http directory)
./pmp-mock-http

# Test OpenAI chat completions
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Features

- ✅ Chat completions (streaming and non-streaming)
- ✅ Text completions
- ✅ Model listing and retrieval
- ✅ Embeddings
- ✅ Error responses (invalid API key, rate limiting)
- ✅ Regex-based header matching for authorization

## Using as a Plugin

This repository can also be used as a plugin in other PMP Mock HTTP instances:

```bash
# Clone this repo as a plugin
./pmp-mock-http --plugins "https://github.com/your-org/pmp-mock-http.git"

# Only load OpenAI mocks
./pmp-mock-http \
  --plugins "https://github.com/your-org/pmp-mock-http.git" \
  --plugin-include-only "openai"
```
