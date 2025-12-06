# Raycast AI Setup Guide

This guide explains how to configure and use gh-proxy-local with Raycast AI.

## Overview

Raycast AI allows you to add custom AI providers through configuration. This guide shows how to set up gh-proxy-local as a custom provider for Raycast, enabling you to use GitHub Copilot models directly in Raycast.

## Prerequisites

- gh-proxy-local running locally (see [HOSTING.md](./HOSTING.md))
- Raycast installed and configured
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
curl http://localhost:8080/v1/models | jq '.data[] | {id, name, owned_by, capabilities}'
```

Common models available:
- `gpt-5-mini` - GPT-5 Mini (OpenAI)
- `gpt-4o` - GPT-4o (OpenAI)
- `gpt-5` - GPT-5 (OpenAI)
- `claude-opus-4.5` - Claude Opus 4.5 (Anthropic)
- `claude-sonnet-4.5` - Claude Sonnet 4.5 (Anthropic)
- And many more...

### Step 3: Configure Raycast AI

Raycast stores custom providers in a configuration file. The configuration location depends on your OS:

**File Location:**
- **macOS:** `~/.config/raycast/settings.json` or `~/Library/Application Support/Raycast/settings.json`
- **Linux:** `~/.config/raycast/settings.json`

**Configuration Template:**

```json
{
  "providers": [
    {
      "id": "GH",
      "name": "Github Copilot",
      "base_url": "http://localhost:8080",
      "models": [
        {
          "id": "gpt-5-mini",
          "name": "GPT-5 Mini",
          "context": 264000,
          "provider": "openai",
          "abilities": {
            "temperature": {
              "supported": false
            },
            "vision": {
              "supported": true
            },
            "system_message": {
              "supported": true
            },
            "tools": {
              "supported": true
            },
            "reasoning_effort": {
              "supported": false
            }
          }
        }
      ]
    }
  ]
}
```

**Configuration Fields:**

| Field | Description | Example |
|-------|-------------|---------|
| `id` | Unique provider identifier | `"GH"` |
| `name` | Display name in Raycast | `"Github Copilot"` |
| `base_url` | Proxy server URL | `"http://localhost:8080"` |
| `models` | Array of available models | See below |

**Model Configuration Fields:**

| Field | Description | Example |
|-------|-------------|---------|
| `id` | Model ID from `/v1/models` | `"gpt-5-mini"` |
| `name` | Display name in Raycast | `"GPT-5 Mini"` |
| `context` | Maximum context window tokens | `264000` |
| `provider` | API provider format | `"openai"` or `"anthropic"` |
| `abilities` | Feature support matrix | See below |

**Ability Fields:**

Each ability can support the following properties:

```json
{
  "supported": true/false,           // Whether ability is supported
  "required": false,                 // (Optional) If ability must be used
  "default": value,                  // (Optional) Default value
  "max": value,                       // (Optional) Maximum value for numeric abilities
  "min": value                        // (Optional) Minimum value for numeric abilities
}
```

Common abilities:
- `temperature` - Controls randomness (0-2)
- `vision` - Image/vision capabilities
- `system_message` - Custom system prompts
- `tools` - Function/tool calling
- `reasoning_effort` - Chain-of-thought reasoning
- `max_tokens` - Output length control

### Step 4: Add Multiple Models

Configure multiple models from different providers:

```json
{
  "providers": [
    {
      "id": "GH",
      "name": "Github Copilot",
      "base_url": "http://localhost:8080",
      "models": [
        {
          "id": "gpt-5-mini",
          "name": "GPT-5 Mini",
          "context": 264000,
          "provider": "openai",
          "abilities": {
            "temperature": { "supported": false },
            "vision": { "supported": true },
            "system_message": { "supported": true },
            "tools": { "supported": true },
            "reasoning_effort": { "supported": false }
          }
        },
        {
          "id": "gpt-4o",
          "name": "GPT-4o",
          "context": 128000,
          "provider": "openai",
          "abilities": {
            "temperature": { "supported": true, "max": 2, "min": 0 },
            "vision": { "supported": true },
            "system_message": { "supported": true },
            "tools": { "supported": true },
            "reasoning_effort": { "supported": false }
          }
        },
        {
          "id": "gpt-5",
          "name": "GPT-5",
          "context": 400000,
          "provider": "openai",
          "abilities": {
            "temperature": { "supported": true, "max": 2, "min": 0 },
            "vision": { "supported": true },
            "system_message": { "supported": true },
            "tools": { "supported": true },
            "reasoning_effort": { "supported": true }
          }
        },
        {
          "id": "claude-opus-4.5",
          "name": "Claude Opus 4.5",
          "context": 144000,
          "provider": "anthropic",
          "abilities": {
            "temperature": { "supported": true, "max": 2, "min": 0 },
            "vision": { "supported": true },
            "system_message": { "supported": true },
            "tools": { "supported": true },
            "reasoning_effort": { "supported": false }
          }
        },
        {
          "id": "claude-sonnet-4.5",
          "name": "Claude Sonnet 4.5",
          "context": 144000,
          "provider": "anthropic",
          "abilities": {
            "temperature": { "supported": true, "max": 2, "min": 0 },
            "vision": { "supported": true },
            "system_message": { "supported": true },
            "tools": { "supported": true },
            "reasoning_effort": { "supported": false }
          }
        }
      ]
    }
  ]
}
```

## Getting Model Details

Get detailed information about a specific model:

```bash
# Get all models
curl http://localhost:8080/v1/models | jq '.data'

# Get a specific model's capabilities
curl http://localhost:8080/v1/models | jq '.data[] | select(.id=="gpt-5-mini")'

# Format for Raycast configuration
curl http://localhost:8080/v1/models | jq '.data[] | {
  id,
  name,
  context: .limits.max_context_window_tokens,
  capabilities
}'
```

## Using in Raycast

Once configured:

1. **Access AI Features:** Use Raycast AI commands
2. **Select Provider:** Choose "Github Copilot" from provider dropdown
3. **Select Model:** Choose model (e.g., "GPT-5 Mini")
4. **Start Using:** Ask questions or use AI features

### Keyboard Shortcuts

- **⌘ + K**: Ask AI question
- **⌘ + ⇧ + A**: Use AI for selected text
- Check Raycast settings for more shortcuts

## Advanced Configuration

### Custom Abilities

Customize ability support based on your needs:

```json
{
  "id": "gpt-5-mini",
  "name": "GPT-5 Mini",
  "context": 264000,
  "provider": "openai",
  "abilities": {
    "temperature": {
      "supported": true,
      "default": 0.7,
      "min": 0,
      "max": 2
    },
    "vision": {
      "supported": true
    },
    "system_message": {
      "supported": true,
      "required": false
    },
    "tools": {
      "supported": true
    },
    "max_tokens": {
      "supported": true,
      "default": 1024,
      "min": 1,
      "max": 64000
    }
  }
}
```

### Multiple Providers

You can add multiple provider configurations:

```json
{
  "providers": [
    {
      "id": "GH",
      "name": "Github Copilot",
      "base_url": "http://localhost:8080",
      "models": [...]
    },
    {
      "id": "CUSTOM",
      "name": "Custom Provider",
      "base_url": "http://other-service:8080",
      "models": [...]
    }
  ]
}
```

### Remote Proxy Server

If gh-proxy-local runs on a remote machine:

```json
{
  "id": "GH",
  "name": "Github Copilot (Remote)",
  "base_url": "http://proxy-server.example.com:8080",
  "models": [...]
}
```

## Verifying the Setup

### Check Configuration

Verify your settings.json is valid JSON:

```bash
# macOS
jq empty ~/.config/raycast/settings.json && echo "Valid JSON"

# Or validate online at jsonlint.com
```

### Test Connection

Test the proxy directly:

```bash
# Health check
curl http://localhost:8080/health

# List models
curl http://localhost:8080/v1/models | jq '.data[].id'

# Test with a model
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5-mini",
    "messages": [{"role": "user", "content": "Hello!"}],
    "max_tokens": 100
  }'
```

### In Raycast

1. Open Raycast preferences (⌘ + ,)
2. Go to "AI"
3. Select "Github Copilot" provider
4. Try the AI features
5. Check if models appear in dropdown

## Common Issues & Troubleshooting

### Connection Refused

**Problem:** Raycast can't connect to gh-proxy-local

**Solutions:**
- Ensure gh-proxy-local is running: `curl http://localhost:8080/health`
- Verify base_url is correct: `http://localhost:8080`
- Check proxy is listening on port 8080: `lsof -i :8080`
- For remote setup, ensure network connectivity
- Try `127.0.0.1` instead of `localhost` if needed

### Configuration Not Loading

**Problem:** Provider doesn't appear in Raycast

**Solutions:**
- Verify JSON syntax: `jq empty ~/.config/raycast/settings.json`
- Restart Raycast (⌘ + Q, then reopen)
- Check file permissions: `chmod 644 ~/.config/raycast/settings.json`
- Verify provider `id` is unique
- Ensure proper file location for your OS

### Model Not Found

**Problem:** Models don't appear in Raycast dropdown

**Solutions:**
- Verify model ID exists: `curl http://localhost:8080/v1/models | jq '.data[].id'`
- Check spelling of model ID exactly matches
- Ensure `provider` field matches model's actual provider
- Restart Raycast after config changes

### GitHub Authentication Expired

**Problem:** Proxy returns authentication error

**Solutions:**
```bash
# Delete cached credentials
rm ~/.copilot_credentials.json

# Run gh-proxy-local to re-authenticate
./bin/gh-proxy-local
# Follow OAuth flow at https://github.com/login/device

# Restart proxy and Raycast
```

### Invalid JSON in Configuration

**Problem:** "Invalid configuration" error

**Solutions:**
- Check JSON syntax: `jq . ~/.config/raycast/settings.json`
- Look for missing commas or quotes
- Use online JSON validator
- Ensure no trailing commas in arrays/objects
- Match exact field names (case-sensitive)

### Wrong Provider Format

**Problem:** Model works but format is incorrect

**Solutions:**
- For OpenAI models (GPT-*): use `"provider": "openai"`
- For Anthropic models (Claude): use `"provider": "anthropic"`
- Check actual provider: `curl http://localhost:8080/v1/models | jq '.data[] | {id, owned_by}'`

## Performance Tips

1. **Start gh-proxy-local first:** Launch proxy before using Raycast AI
2. **Keep it running:** Use Docker for persistent service
3. **Monitor resources:** Check CPU/memory usage
4. **Cache credentials:** Tokens are cached automatically
5. **Use appropriate models:** Smaller models faster for simple tasks

## Security Considerations

1. **Local Use:** Keep proxy on localhost for development
2. **Network Isolation:** Don't expose proxy to untrusted networks
3. **GitHub Terms:** Ensure usage complies with GitHub Copilot terms
4. **Credentials:** Never share `~/.copilot_credentials.json`
5. **Configuration:** Store settings.json securely

## Model Capabilities Reference

### OpenAI Models

```json
{
  "id": "gpt-5-mini",
  "name": "GPT-5 Mini",
  "context": 264000,
  "provider": "openai",
  "abilities": {
    "temperature": { "supported": false },
    "vision": { "supported": true },
    "system_message": { "supported": true },
    "tools": { "supported": true },
    "reasoning_effort": { "supported": false }
  }
}
```

### Anthropic Models

```json
{
  "id": "claude-opus-4.5",
  "name": "Claude Opus 4.5",
  "context": 144000,
  "provider": "anthropic",
  "abilities": {
    "temperature": { "supported": true },
    "vision": { "supported": true },
    "system_message": { "supported": true },
    "tools": { "supported": true },
    "reasoning_effort": { "supported": false }
  }
}
```

## Updating Models

To add new models or update existing ones:

1. Check available models: `curl http://localhost:8080/v1/models`
2. Edit `~/.config/raycast/settings.json`
3. Add new model entries to the `models` array
4. Restart Raycast
5. Models should appear in provider dropdown

## Getting Help

- Check Raycast documentation: https://manual.raycast.com/ai
- View proxy logs: `docker logs -f gh-proxy-local`
- Enable debug mode: `COPILOT_DEBUG=1 ./bin/gh-proxy-local`
- Review [HOSTING.md](./HOSTING.md) for deployment details
- Check GitHub issues: https://github.com/rahulvramesh/gh-proxy-local/issues

## Related Documentation

- [Main README](../README.md) - Project overview and features
- [HOSTING.md](./HOSTING.md) - Detailed hosting and deployment guide
- [Factory AI Droid Setup](./factoryai-droid.md) - Droid configuration guide
- [GitHub Copilot](https://github.com/features/copilot)
- [Raycast Manual](https://manual.raycast.com)
