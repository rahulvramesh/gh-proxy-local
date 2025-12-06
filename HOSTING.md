# Local Hosting Guide for GitHub Copilot Proxy Server

This guide covers Docker and Docker Compose setup for local development of the GitHub Copilot Proxy Server.

## Table of Contents

1. [Authentication & Credentials](#authentication--credentials)
2. [Docker Setup](#docker-setup)
3. [Docker Compose](#docker-compose)
4. [Configuration](#configuration)
5. [Troubleshooting](#troubleshooting)

## Authentication & Credentials

### Overview

The proxy server requires GitHub permissions to access the Copilot API. Authorization is handled through GitHub's device flow, which stores access info locally in a file.

### Stored Information Structure

Access information is stored in `~/.copilot_credentials.json` as a JSON file containing three fields:

1. **github_token** (required): GitHub authorization obtained from device flow
2. **copilot_token** (optional): Cached Copilot API access (automatically refreshed)
3. **copilot_expires** (optional): Expiration time in milliseconds

The file is automatically created and managed by the application when you authenticate for the first time.

### GitHub Device Flow Setup

The proxy uses GitHub's device flow for headless authentication:

1. **First Run Setup:**
   ```bash
   ./bin/gh-proxy-local
   # Output:
   # === GitHub Copilot Authentication ===
   # Please visit: https://github.com/login/device
   # Enter code: XXXX-XXXX
   # Waiting for authorization...
   ```

2. **User Actions:**
   - Open https://github.com/login/device in your browser
   - Enter the code displayed by the application
   - Authorize the application
   - Return to terminal (polling completes automatically)

3. **Token Storage:**
   - GitHub token is saved to `~/.copilot_credentials.json`
   - File permissions are set to `0600` (readable only by owner)
   - Credentials persist across application restarts

### Setting Up Credentials for Docker

#### Option 1: Pre-Generate Credentials (Recommended)

```bash
# Generate credentials on your machine first
./bin/gh-proxy-local
# Complete the OAuth flow
# Credentials stored in ~/.copilot_credentials.json

# Now use Docker with the credentials
docker run -d \
  -v ~/.copilot_credentials.json:/home/copilot/.copilot_credentials.json:ro \
  gh-proxy-local:latest
```

#### Option 2: Docker Compose with Pre-Generated Credentials

```bash
# Ensure credentials exist first
./bin/gh-proxy-local  # Complete auth if needed

# Start with docker-compose
docker-compose up -d
# docker-compose.yml handles mounting credentials automatically
```

### Token Lifecycle

1. **GitHub Token:**
   - Obtained via device flow OAuth
   - Never expires (until revoked)
   - Used to request Copilot tokens

2. **Copilot Token:**
   - Short-lived token (typically 1 hour)
   - Automatically refreshed before expiration
   - Cached for performance

### Managing Credentials

#### Regenerate Credentials
```bash
# Delete cached credentials to trigger re-authentication
rm ~/.copilot_credentials.json

# Run application for re-authentication
./bin/gh-proxy-local
# Complete OAuth flow again
```

#### View Current Credentials
```bash
# Check if credentials file exists
cat ~/.copilot_credentials.json | jq .

# Don't expose the tokens in logs or version control!
```

#### Revoke Access
```bash
# Remove credentials locally
rm ~/.copilot_credentials.json

# Optionally revoke on GitHub:
# Settings → Developer settings → Personal access tokens → Revoke
```

### Troubleshooting Authentication

#### "Re-authenticate by deleting ~/.copilot_credentials.json"
- GitHub token has expired or been revoked
- Delete credentials and run again to re-authenticate

#### "Device code request failed"
- Check internet connectivity
- Verify GitHub is not blocked by firewall
- Try again in a few moments

#### "Failed to parse credentials"
- Credentials file is corrupted
- Delete file and re-authenticate
- Verify JSON syntax if manually editing

#### Docker Container Can't Access Credentials
```bash
# Verify credentials file exists
ls -la ~/.copilot_credentials.json

# Check Docker mount is correct
docker inspect gh-proxy-local | grep -A5 Mounts

# Fix permissions if needed
chmod 600 ~/.copilot_credentials.json
```

## Docker Setup

### Building the Docker Image

The project includes a multi-stage Dockerfile that optimizes the final image size:

```bash
# Build the Docker image
docker build -t gh-proxy-local:latest .

# Build with a specific tag
docker build -t gh-proxy-local:v1.0.0 .
```

### Running a Single Container

#### Basic Usage
```bash
docker run -d \
  -p 8080:8080 \
  --name gh-proxy-local \
  gh-proxy-local:latest
```

#### With Environment Variables
```bash
docker run -d \
  -p 8080:8080 \
  -e COPILOT_HOST=0.0.0.0 \
  -e COPILOT_PORT=8080 \
  -e COPILOT_DEBUG=0 \
  --name gh-proxy-local \
  gh-proxy-local:latest
```

#### With Credentials Volume
```bash
docker run -d \
  -p 8080:8080 \
  -v ~/.copilot_credentials.json:/home/copilot/.copilot_credentials.json:ro \
  --name gh-proxy-local \
  gh-proxy-local:latest
```

#### Full Example with All Options
```bash
docker run -d \
  --name gh-proxy-local \
  --restart unless-stopped \
  -p 8080:8080 \
  -e COPILOT_HOST=0.0.0.0 \
  -e COPILOT_PORT=8080 \
  -e COPILOT_DEBUG=0 \
  -v ~/.copilot_credentials.json:/home/copilot/.copilot_credentials.json:ro \
  --health-cmd="wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1" \
  --health-interval=30s \
  --health-timeout=3s \
  --health-retries=3 \
  gh-proxy-local:latest
```

### Container Management

```bash
# View running containers
docker ps

# View all containers (including stopped)
docker ps -a

# Stop the container
docker stop gh-proxy-local

# Start the container
docker start gh-proxy-local

# Restart the container
docker restart gh-proxy-local

# Remove the container
docker rm gh-proxy-local

# View container logs
docker logs gh-proxy-local

# Follow logs in real-time
docker logs -f gh-proxy-local

# View container stats
docker stats gh-proxy-local
```

## Docker Compose

### Using Docker Compose for Development

Docker Compose simplifies local development by managing containers, networks, and volumes:

```bash
# Start the service
docker-compose up -d

# Start with output in terminal
docker-compose up

# Rebuild the image and start
docker-compose up --build

# Stop the service
docker-compose down

# View logs
docker-compose logs -f gh-proxy-local

# Restart the service
docker-compose restart
```

### Docker Compose Configuration

The `docker-compose.yml` file includes:

- Automatic image building from Dockerfile
- Port mapping (8080:8080)
- Environment variables configuration
- Credentials volume mounting
- Auto-restart policy
- Health checks
- Network setup

### Customizing Docker Compose

Edit `docker-compose.yml` to customize:

```yaml
services:
  gh-proxy-local:
    environment:
      COPILOT_HOST: 0.0.0.0
      COPILOT_PORT: 8080
      COPILOT_DEBUG: 1  # Enable debug logging for development
    ports:
      - "3000:8080"  # Map to different port
```

### Credentials Setup with Docker Compose

Before running Docker Compose, generate credentials:

```bash
# Run the binary directly to authenticate
./bin/gh-proxy-local
# This creates ~/.copilot_credentials.json

# Now start docker-compose
docker-compose up -d
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `COPILOT_HOST` | `0.0.0.0` | Server host address |
| `COPILOT_PORT` | `8080` | Server port |
| `COPILOT_DEBUG` | `0` | Enable debug logging (1 or 0) |

### Port Mapping

```bash
# Map to different host port
docker run -p 3000:8080 gh-proxy-local:latest

# Map to specific host interface
docker run -p 127.0.0.1:8080:8080 gh-proxy-local:latest
```

### Enabling Debug Logging

```bash
# Via environment variable
docker run -e COPILOT_DEBUG=1 gh-proxy-local:latest

# Or with docker-compose, update HOSTING.md:
# environment:
#   COPILOT_DEBUG: 1
```

## Troubleshooting

### Connection Issues

#### Connection Refused
```bash
# Check if container is running
docker ps | grep gh-proxy-local

# Check port mapping
docker port gh-proxy-local

# Check logs
docker logs gh-proxy-local
```

#### Port Already in Use
```bash
# Find process using port 8080
lsof -i :8080

# Or use different port
docker run -p 3000:8080 gh-proxy-local:latest
```

### Container Issues

#### Out of Memory
```bash
# Check memory usage
docker stats gh-proxy-local

# Increase memory limit
docker run -m 1g gh-proxy-local:latest
```

#### Container Exits Immediately
```bash
# Check logs for errors
docker logs gh-proxy-local

# Run interactively to see output
docker run -it gh-proxy-local:latest
```

### Image Building Issues

#### Build Fails
```bash
# Check build logs with verbose output
docker build -t gh-proxy-local:latest . --progress=plain

# Ensure Go dependencies are available
docker build --build-arg BUILDKIT_INLINE_CACHE=1 -t gh-proxy-local:latest .
```

### Health Check

The container includes a health check endpoint:

```bash
# Check container health
docker inspect gh-proxy-local --format='{{.State.Health.Status}}'

# Or test directly
curl http://localhost:8080/health
# Response: {"status": "ok"}
```

### View Logs

```bash
# Docker
docker logs gh-proxy-local

# Docker Compose
docker-compose logs gh-proxy-local

# Follow logs in real-time
docker logs -f gh-proxy-local

# Get last 50 lines
docker logs --tail 50 gh-proxy-local
```

### Testing the Proxy

Once running, test the proxy with sample requests:

```bash
# Health check
curl http://localhost:8080/health

# List available models
curl http://localhost:8080/v1/models

# Chat completion (OpenAI format)
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": false
  }'

# Streaming response
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Stream this!"}],
    "stream": true
  }'
```

## Quick Start Summary

1. **Generate credentials:**
   ```bash
   ./bin/gh-proxy-local
   # Complete OAuth flow
   ```

2. **Build Docker image:**
   ```bash
   docker build -t gh-proxy-local:latest .
   ```

3. **Start with Docker Compose:**
   ```bash
   docker-compose up -d
   ```

4. **Verify it's running:**
   ```bash
   curl http://localhost:8080/health
   ```

5. **View logs:**
   ```bash
   docker-compose logs -f gh-proxy-local
   ```

6. **Stop when done:**
   ```bash
   docker-compose down
   ```
