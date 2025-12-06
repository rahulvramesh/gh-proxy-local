# Factory AI Droid Setup Guide

This guide explains how to configure and use gh-proxy-local with Factory AI's Droid.

## Overview

Factory AI's Droid can use custom LLM models through custom model configurations. This guide shows how to set up gh-proxy-local as a custom model provider for Droid, allowing you to use GitHub Copilot models through the proxy.

## Prerequisites

- gh-proxy-local running locally (see [HOSTING.md](./HOSTING.md))
- Factory AI installed and configured
- GitHub Copilot authenticated (via `~/.copilot_credentials.json`)

## Quick Setup

### Step 1: Start gh-proxy-local

First, ensure the proxy server is running:

```bash
# Using binary
./bin/gh-proxy-local

# Or using Docker
docker-compose up -d

# Or using Docker directly
docker run -d -p 8080:8080 gh-proxy-local:latest
```

Verify it's running:
```bash
curl http://localhost:8080/health
# Response: {"status": "ok"}
```

### Step 2: Get Available Models

List all available models from the proxy:

```bash
curl http://localhost:8080/v1/models | jq '.data[].id'
```

Common models available:
- `claude-opus-4.5` - Claude Opus 4.5 (Anthropic)
- `claude-sonnet-4.5` - Claude Sonnet 4.5 (Anthropic)
- `gpt-4o` - GPT-4o (OpenAI)
- `gpt-5` - GPT-5 (OpenAI)
- And many more...

### Step 3: Configure Factory AI Droid

Add the custom model configuration to your Factory AI settings. The configuration file is typically located at:

**Location:**
- **macOS/Linux:** `~/.factory/config.json` or via Factory AI Settings UI
- **Windows:** `%APPDATA%\.factory\config.json`

**Configuration Template:**

```json
{
  "custom_models": [
    {
      "model_display_name": "GH: claude-opus-4.5",
      "supports_images": true,
      "model": "claude-opus-4.5",
      "base_url": "http://localhost:8080",
      "api_key": "<master-key-if-any>",
      "provider": "anthropic",
      "extra_headers": {}
    }
  ]
}
```

**Configuration Fields:**

| Field | Description | Example |
|-------|-------------|---------|
| `model_display_name` | Display name in Factory AI UI | `"GH: claude-opus-4.5"` |
| `supports_images` | Enable vision capabilities | `true` or `false` |
| `model` | Model ID from `/v1/models` | `"claude-opus-4.5"` |
| `base_url` | Proxy server URL | `"http://localhost:8080"` |
| `api_key` | Optional master key (if using COPILOT_API_KEY) | `"your-key"` or `""` |
| `provider` | API provider format | `"anthropic"` or `"openai"` |
| `extra_headers` | Additional HTTP headers | `{}` |

### Step 4: Add Multiple Models (Optional)

You can configure multiple models for different use cases:

```json
{
  "custom_models": [
    {
      "model_display_name": "GH: Claude Opus 4.5",
      "supports_images": true,
      "model": "claude-opus-4.5",
      "base_url": "http://localhost:8080",
      "api_key": "",
      "provider": "anthropic",
      "extra_headers": {}
    },
    {
      "model_display_name": "GH: Claude Sonnet 4.5",
      "supports_images": true,
      "model": "claude-sonnet-4.5",
      "base_url": "http://localhost:8080",
      "api_key": "",
      "provider": "anthropic",
      "extra_headers": {}
    },
    {
      "model_display_name": "GH: GPT-4o",
      "supports_images": true,
      "model": "gpt-4o",
      "base_url": "http://localhost:8080",
      "api_key": "",
      "provider": "openai",
      "extra_headers": {}
    },
    {
      "model_display_name": "GH: GPT-5",
      "supports_images": true,
      "model": "gpt-5",
      "base_url": "http://localhost:8080",
      "api_key": "",
      "provider": "openai",
      "extra_headers": {}
    }
  ]
}
```

## With Master Key Authentication

If you've configured a master key (`COPILOT_API_KEY`) on the proxy server:

### Step 1: Set Master Key on Proxy

```bash
# Run proxy with master key
export COPILOT_API_KEY=your_secret_master_key
./bin/gh-proxy-local

# Or with Docker
docker run -e COPILOT_API_KEY=your_secret_master_key \
  -p 8080:8080 \
  gh-proxy-local:latest
```

### Step 2: Update Factory AI Configuration

Include the master key in the configuration:

```json
{
  "custom_models": [
    {
      "model_display_name": "GH: claude-opus-4.5",
      "supports_images": true,
      "model": "claude-opus-4.5",
      "base_url": "http://localhost:8080",
      "api_key": "your_secret_master_key",
      "provider": "anthropic",
      "extra_headers": {
        "Authorization": "Bearer your_secret_master_key"
      }
    }
  ]
}
```

## Verifying the Setup

### Test Connection in Factory AI

1. Open Factory AI settings
2. Navigate to Models or Custom Models section
3. Look for your configured model (e.g., "GH: claude-opus-4.5")
4. Try making a request with the model
5. Verify it receives a response

### Manual Testing

Test the proxy directly:

```bash
# Test basic connectivity
curl http://localhost:8080/health

# List available models
curl http://localhost:8080/v1/models

# Test with specific model (Anthropic format)
curl http://localhost:8080/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-opus-4.5",
    "max_tokens": 100,
    "messages": [{"role": "user", "content": "Hello!"}]
  }'

# Test with specific model (OpenAI format)
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Hello!"}],
    "max_tokens": 100
  }'
```

## Common Issues & Troubleshooting

### Connection Refused

**Problem:** Factory AI can't connect to `http://localhost:8080`

**Solutions:**
- Ensure gh-proxy-local is running: `curl http://localhost:8080/health`
- Check proxy is listening on correct port: `lsof -i :8080`
- Restart the proxy server
- Verify localhost resolves: `ping localhost`

### Invalid API Key

**Problem:** 401/403 error with master key

**Solutions:**
- Verify `COPILOT_API_KEY` matches in proxy and config
- Ensure `api_key` field matches exactly in Factory AI config
- Check proxy logs for auth errors
- Re-authenticate GitHub Copilot: `rm ~/.copilot_credentials.json && ./bin/gh-proxy-local`

### Model Not Found

**Problem:** "Model not found" or 404 error

**Solutions:**
- List available models: `curl http://localhost:8080/v1/models | jq '.data[].id'`
- Verify model ID spelling exactly matches
- Check if model is preview: `curl http://localhost:8080/v1/models | jq '.data[] | select(.id=="model-name")'`

### GitHub Authentication Expired

**Problem:** Proxy returns 401 "Re-authenticate" message

**Solutions:**
```bash
# Delete cached credentials
rm ~/.copilot_credentials.json

# Run proxy to re-authenticate
./bin/gh-proxy-local
# Follow OAuth flow at https://github.com/login/device

# Restart your service
```

### Wrong Provider Format

**Problem:** Model returns errors or unexpected format

**Solutions:**
- For Claude/Anthropic models, use `"provider": "anthropic"`
- For GPT/OpenAI models, use `"provider": "openai"`
- Check model owner: `curl http://localhost:8080/v1/models | jq '.data[] | select(.id=="model").owned_by'`

## Advanced Configuration

### Custom Headers

Add custom headers if needed (e.g., for proxies or authentication layers):

```json
{
  "custom_models": [
    {
      "model_display_name": "GH: claude-opus-4.5",
      "supports_images": true,
      "model": "claude-opus-4.5",
      "base_url": "http://localhost:8080",
      "api_key": "",
      "provider": "anthropic",
      "extra_headers": {
        "X-Custom-Header": "value",
        "Authorization": "Bearer token"
      }
    }
  ]
}
```

### Remote Proxy Server

If gh-proxy-local runs on a remote machine:

```json
{
  "custom_models": [
    {
      "model_display_name": "Remote GH: claude-opus-4.5",
      "supports_images": true,
      "model": "claude-opus-4.5",
      "base_url": "http://proxy-server.example.com:8080",
      "api_key": "master_key_if_configured",
      "provider": "anthropic",
      "extra_headers": {}
    }
  ]
}
```

## Performance Tips

1. **Keep proxy running:** Run gh-proxy-local as a background service or in Docker for always-on access
2. **Use Docker:** Container setup ensures consistent performance and easy management
3. **Enable caching:** Copilot tokens are automatically cached in `~/.copilot_credentials.json`
4. **Monitor resources:** Check CPU/memory usage, especially with multiple concurrent requests

## Security Considerations

1. **Local Use Only:** Currently designed for local development
2. **Master Key:** If using `COPILOT_API_KEY`, treat it as a secret
3. **Network Access:** Keep proxy on localhost for development
4. **GitHub Terms:** Ensure usage complies with GitHub Copilot terms
5. **Credentials:** Never commit credentials to version control

## Getting Help

- Check gh-proxy-local logs: `docker logs -f gh-proxy-local`
- Enable debug logging: `COPILOT_DEBUG=1 ./bin/gh-proxy-local`
- Review [HOSTING.md](./HOSTING.md) for detailed proxy documentation
- Check GitHub issues: https://github.com/rahulvramesh/gh-proxy-local/issues

## Related Documentation

- [Main README](../README.md) - Project overview and features
- [HOSTING.md](./HOSTING.md) - Detailed hosting and deployment guide
- [GitHub Copilot](https://github.com/features/copilot)
- [Factory AI Documentation](https://docs.factory.ai)
