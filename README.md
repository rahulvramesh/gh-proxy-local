# GitHub Copilot Proxy Server

An OpenAI and Anthropic API-compatible proxy server that routes requests through GitHub Copilot's API. Use Copilot with any tool that supports OpenAI or Anthropic APIs.

## ⚠️ Disclaimer

This project is an **unofficial, community-maintained** tool that provides local access to GitHub Copilot through a proxy interface. It is **not affiliated with, endorsed by, or supported by GitHub or Copilot**. 

**Important:**
- This tool uses your personal GitHub Copilot subscription
- By using this project, you accept full responsibility for compliance with GitHub's Terms of Service and Copilot's usage policies
- Misuse of this proxy may violate GitHub Copilot's terms and could result in account suspension
- This is provided "as-is" without warranty - use at your own risk
- The authors are not liable for any consequences of using this tool

Please review [GitHub's Terms of Service](https://docs.github.com/en/site-policy/github-terms/github-terms-of-service) and [GitHub Copilot Terms](https://github.com/features/copilot/terms) before using this tool.

## Features

- **OpenAI API Compatible** - `/v1/chat/completions`, `/v1/responses`
- **Anthropic API Compatible** - `/v1/messages`, `/v1/messages/count_tokens`
- **Streaming Support** - Full SSE streaming for both OpenAI and Anthropic formats
- **Model Aliases** - Seamless support for common model names (claude-3.5-sonnet, gpt-4, etc.)
- **Vision Support** - Image content in messages
- **Tool/Function Calling** - Automatic conversion between API formats
- **CORS Enabled** - Works with web-based clients
- **Models Discovery** - List available models from Copilot API
- **LLM Observability** - Langfuse integration for tracing, cost tracking, and analytics

## Quick Start

### Prerequisites

- Go 1.24+ (for building from source)
- GitHub account (for Copilot authentication)
- macOS, Linux, or Windows

### Installation

#### Option 1: Download Binary

Download the latest release for your platform: https://github.com/rahulvramesh/gh-proxy-local/releases

```bash
# macOS (ARM64)
wget https://github.com/rahulvramesh/gh-proxy-local/releases/download/v1.0.0/gh-proxy-local-darwin-arm64
chmod +x gh-proxy-local-darwin-arm64
./gh-proxy-local-darwin-arm64

# Linux (x86_64)
wget https://github.com/rahulvramesh/gh-proxy-local/releases/download/v1.0.0/gh-proxy-local-linux-amd64
chmod +x gh-proxy-local-linux-amd64
./gh-proxy-local-linux-amd64

# Windows (PowerShell)
# Download gh-proxy-local-windows-amd64.exe and run
.\gh-proxy-local-windows-amd64.exe
```

#### Option 2: Build from Source

```bash
git clone https://github.com/rahulvramesh/gh-proxy-local.git
cd gh-proxy-local
make build
./bin/gh-proxy-local
```

### First Run

On first startup, the server will initiate GitHub OAuth device flow authentication:

```
Checking GitHub Copilot authentication...
Please visit: https://github.com/login/device
Enter code: XXXX-XXXX

✓ Authentication verified!
🚀 Starting GitHub Copilot Proxy Server
   Host: 0.0.0.0
   Port: 8080

📡 Endpoints:
   OpenAI Chat:      http://localhost:8080/v1/chat/completions
   OpenAI Responses: http://localhost:8080/v1/responses
   Anthropic:        http://localhost:8080/v1/messages
   Models:           http://localhost:8080/v1/models
   Health:           http://localhost:8080/health
```

Credentials are cached in `~/.copilot_credentials.json` for subsequent runs.

## Configuration

### Server Configuration

#### Environment Variables

```bash
COPILOT_HOST=0.0.0.0    # Server host (default: 0.0.0.0)
COPILOT_PORT=8080       # Server port (default: 8080)
COPILOT_DEBUG=1         # Enable debug logging (default: false)
COPILOT_API_KEY=key     # Optional API key for bearer auth
```

#### Command Line Flags

```bash
./gh-proxy-local --host 127.0.0.1 --port 3000
./gh-proxy-local -H 127.0.0.1 -p 3000  # Shorthand
```

### Langfuse Observability

Enable LLM observability and monitoring with [Langfuse](https://langfuse.com/) to track all API requests, responses, token usage, and costs.

#### Setup

1. Create a free account at [Langfuse Cloud](https://cloud.langfuse.com) or [self-host](https://langfuse.com/docs/self-host)
2. Create a project and get your API credentials
3. Configure the proxy with environment variables:

```bash
LANGFUSE_ENABLED=true
LANGFUSE_HOST=https://cloud.langfuse.com    # Or your self-hosted URL
LANGFUSE_PUBLIC_KEY=pk-xxxxxxxxxxxxx
LANGFUSE_SECRET_KEY=sk-xxxxxxxxxxxxx

# Optional: Batch configuration
LANGFUSE_BATCH_SIZE=10                      # Events per batch (default: 10)
LANGFUSE_FLUSH_INTERVAL=5s                  # Max wait before flushing (default: 5s)
```

#### What Gets Tracked

- **Request/Response Data**: Full prompt and completion text
- **Token Usage**: Prompt tokens, completion tokens, total tokens, cached tokens
- **Latency**: Request duration and timestamps
- **Models**: Which model was used for each request
- **API Endpoints**: OpenAI chat, Anthropic messages, Responses API
- **Errors**: Failed requests with error details

#### Performance

- **Non-blocking**: Events are queued asynchronously, doesn't block requests
- **Batched**: Events are sent in batches (configurable size/interval)
- **Graceful degradation**: If queue is full, events are dropped with a warning
- **Graceful shutdown**: Pending events are flushed on server shutdown

#### Example: Running with Langfuse

```bash
LANGFUSE_ENABLED=true \
LANGFUSE_HOST=https://cloud.langfuse.com \
LANGFUSE_PUBLIC_KEY=pk-xxxxxxxxxxxxx \
LANGFUSE_SECRET_KEY=sk-xxxxxxxxxxxxx \
./gh-proxy-local

# In dashboard, you'll see:
# - Traces for each API request
# - Token usage per model
# - Cost analysis
# - Latency metrics
# - Error tracking
```

## Usage Examples

### OpenAI Chat Completions

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": false
  }'
```

### Anthropic Messages

```bash
curl http://localhost:8080/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Streaming Response

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Stream this!"}],
    "stream": true
  }'
```

### List Available Models

Retrieve all available models from the Copilot API:

```bash
curl http://localhost:8080/v1/models
```

**Response Format:**

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4o",
      "object": "model",
      "created": 1700000000,
      "owned_by": "openai",
      "name": "GPT-4o",
      "version": "gpt-4o-2024-11-20",
      "limits": {
        "max_context_window_tokens": 128000,
        "max_output_tokens": 4096,
        "max_prompt_tokens": 64000
      },
      "capabilities": {
        "vision": true,
        "tool_calls": true,
        "parallel_tool_calls": true,
        "streaming": true
      }
    },
    {
      "id": "claude-sonnet-4.5",
      "object": "model",
      "created": 1700000000,
      "owned_by": "anthropic",
      "name": "Claude Sonnet 4.5",
      "version": "claude-sonnet-4.5",
      "limits": {
        "max_context_window_tokens": 144000,
        "max_output_tokens": 16000,
        "max_prompt_tokens": 128000
      },
      "capabilities": {
        "vision": true,
        "tool_calls": true,
        "parallel_tool_calls": true,
        "streaming": true
      }
    },
    {
      "id": "gpt-5",
      "object": "model",
      "created": 1700000000,
      "owned_by": "openai",
      "name": "GPT-5",
      "version": "gpt-5",
      "limits": {
        "max_context_window_tokens": 400000,
        "max_output_tokens": 128000,
        "max_prompt_tokens": 128000
      },
      "capabilities": {
        "vision": true,
        "tool_calls": true,
        "parallel_tool_calls": true,
        "streaming": true,
        "structured_outputs": true
      }
    }
  ]
}
```

**Response Fields:**
- `id`: Model identifier (use this in requests)
- `object`: Type of object (always "model")
- `created`: Unix timestamp of model creation
- `owned_by`: Model provider (openai, anthropic, google, xai, etc.)
- `name`: Human-readable model name
- `version`: Specific version identifier
- `limits`: Token limits for the model
  - `max_context_window_tokens`: Maximum context size
  - `max_output_tokens`: Maximum response length
  - `max_prompt_tokens`: Maximum prompt length
- `capabilities`: Feature support (vision, tool_calls, streaming, etc.)
- `preview` (optional): Indicates if model is in preview status

**Example: Get only available model IDs**

```bash
curl -s http://localhost:8080/v1/models | jq '.data[].id'
```

## Model Aliases

Supported model names are automatically resolved to available Copilot models:

- `gpt-4`, `gpt-4-turbo`, `gpt-4o` → Latest GPT model
- `claude-3-5-sonnet`, `claude-3-opus` → Claude models
- `gpt-4-vision` → Vision-capable model
- And many more...

## Building for All Platforms

```bash
# Build for current OS/architecture
make build

# Build for all supported platforms
make build-all

# Clean build artifacts
make clean
```

Binaries are placed in `bin/` directory:
- Linux: `gh-proxy-local-linux-{amd64,arm64}`
- macOS: `gh-proxy-local-darwin-{amd64,arm64}`
- Windows: `gh-proxy-local-windows-amd64.exe`

## Architecture

```
HTTP Request
    ↓
[Proxy Server]
    ↓
[Request Handler] (OpenAI/Anthropic format)
    ↓
[Converter] (→ Copilot format)
    ↓
[Copilot Client]
    ↓
[GitHub Copilot API]
    ↓
[Response Converter] (→ Original format)
    ↓
HTTP Response
```

### Key Components

- **Auth Manager** - GitHub OAuth device flow and token management
- **Copilot Client** - HTTP client for GitHub Copilot API
- **Handlers** - HTTP handlers for different API endpoints
- **Converters** - Format conversion between OpenAI/Anthropic/Copilot
- **Models** - Model definitions and aliases

## Security Notes

- Credentials are stored locally in `~/.copilot_credentials.json`
- Server listens on `0.0.0.0:8080` by default (accessible from network)
- Use firewall rules or reverse proxy for production deployments
- No authentication required at the proxy level (relies on GitHub credentials)

## Troubleshooting

### Authentication Failed

Delete cached credentials and restart:
```bash
rm ~/.copilot_credentials.json
./gh-proxy-local
```

### Enable Debug Logging

```bash
COPILOT_DEBUG=1 ./gh-proxy-local
```

### Port Already in Use

```bash
./gh-proxy-local --port 3000
```

## Tool-Specific Documentation

This proxy is compatible with various AI tools. See the detailed setup guides for your specific tool:

### IDE & Development Tools

- **[Factory AI Droid](./docs/factoryai-droid.md)** - Complete setup guide for Factory AI's Droid agent
  - Custom model configuration
  - Master key authentication
  - Multiple model setup
  - Troubleshooting guide

- **[Raycast AI](./docs/raycast-ai.md)** - Integration guide for Raycast AI
  - Provider configuration
  - Model abilities setup
  - Multiple model examples
  - Keyboard shortcuts

### Deployment & Hosting

- **[HOSTING.md](./docs/HOSTING.md)** - Comprehensive hosting and deployment guide
  - Docker setup and configuration
  - Docker Compose for local development
  - Authentication and credentials management
  - Troubleshooting and monitoring

## License

MIT

## Contributing

Contributions welcome! Feel free to open issues or PRs.

## Related

- [GitHub Copilot](https://github.com/features/copilot)
- [OpenAI API](https://platform.openai.com/docs)
- [Anthropic API](https://docs.anthropic.com)
