# GitHub Copilot Proxy Server

An OpenAI and Anthropic API-compatible proxy server that routes requests through GitHub Copilot's API. Use Copilot with any tool that supports OpenAI or Anthropic APIs.

## Features

- **OpenAI API Compatible** - `/v1/chat/completions`, `/v1/responses`
- **Anthropic API Compatible** - `/v1/messages`, `/v1/messages/count_tokens`
- **Streaming Support** - Full SSE streaming for both OpenAI and Anthropic formats
- **Model Aliases** - Seamless support for common model names (claude-3.5-sonnet, gpt-4, etc.)
- **Vision Support** - Image content in messages
- **Tool/Function Calling** - Automatic conversion between API formats
- **CORS Enabled** - Works with web-based clients
- **Models Discovery** - List available models from Copilot API

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

### Environment Variables

```bash
COPILOT_HOST=0.0.0.0    # Server host (default: 0.0.0.0)
COPILOT_PORT=8080       # Server port (default: 8080)
COPILOT_DEBUG=1         # Enable debug logging (default: false)
```

### Command Line Flags

```bash
./gh-proxy-local --host 127.0.0.1 --port 3000
./gh-proxy-local -H 127.0.0.1 -p 3000  # Shorthand
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

```bash
curl http://localhost:8080/v1/models
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

## License

MIT

## Contributing

Contributions welcome! Feel free to open issues or PRs.

## Related

- [GitHub Copilot](https://github.com/features/copilot)
- [OpenAI API](https://platform.openai.com/docs)
- [Anthropic API](https://docs.anthropic.com)
