# NATS-LS Scripts

## NATS Test Publisher

A Python tool to spin up multiple NATS publishers for testing message ingestion.

### Installation

```bash
pip install nats-py
```

### Usage

```bash
# Basic usage - spin up publishers by type
uv run nats_test_publisher.py --normal 5          # 5 core NATS publishers
uv run nats_test_publisher.py --js 3              # 3 JetStream publishers
uv run nats_test_publisher.py --normal 5 --js 3 --kv 2  # Mix of types

# Common options
uv run nats_test_publisher.py --normal 10 --verbose           # See each message
uv run nats_test_publisher.py --normal 10 --normal-interval 100  # Faster (100ms)
uv run nats_test_publisher.py --url nats://server:4222 --normal 5  # Custom server

# Use a config file
uv run nats_test_publisher.py --generate-config > config.json
uv run nats_test_publisher.py --config config.json
```

### Publisher Types

| Flag | Type | Description |
|------|------|-------------|
| `--normal N` | Core NATS | Standard pub/sub |
| `--js N` | JetStream | Persistent messaging |
| `--reqrep N` | Request-Reply | Sync request/response |
| `--kv N` | Key-Value | JetStream KV store |
| `--obj N` | Object Store | JetStream object storage |

### Key Options

- `--url` - NATS server URL (default: `nats://localhost:4222`)
- `--{type}-subject` - Subject prefix for that publisher type
- `--{type}-interval` - Publish interval in ms
- `--verbose` / `-v` - Log every message
- `--generate-config` - Output sample JSON config

Run `uv run nats_test_publisher.py --help` for all options.

## Requirements for test Scripts

- UV