# NATS-LS Scripts

## NATS Test Publisher

A Python tool to spin up multiple NATS publishers for testing message ingestion.

### Installation

```bash
pip install nats-py
```

### Usage

#### Basic Examples

```bash
# Basic usage - spin up publishers by type
python publish.py --normal 5          # 5 core NATS publishers
python publish.py --js 3              # 3 JetStream publishers
python publish.py --normal 5 --js 3 --kv 2  # Mix of types

# Common options
python publish.py --normal 10 --verbose           # See each message
python publish.py --normal 10 --normal-interval 100  # Faster (100ms)
python publish.py --url nats://server:4222 --normal 5  # Custom server

# Use a config file
python publish.py --generate-config > config.json
python publish.py --config config.json
```

#### Subject Diversity (NEW!)

The publisher now supports multiple ways to generate diverse subject patterns for testing:

**Random Subject Fuzzing**

Generate random, realistic subject hierarchies:

```bash
# 10 publishers with random subjects
python publish.py --normal 10 --normal-fuzz

# Control hierarchy depth (1-5 levels deep)
python publish.py --normal 10 --normal-fuzz --normal-depth-max 5 --normal-depth-min 2

# Works with JetStream too
python publish.py --js 5 --js-fuzz --js-depth-max 4
```

Example subjects generated in fuzz mode:
- `users.api.create`
- `payments.v2.processor.id42`
- `events.staging.logs`
- `orders.service.update.v1`

**Multiple Subject Bases**

Rotate publishers through different subject prefixes:

```bash
# Publishers cycle through these bases
python publish.py --normal 6 --normal-bases api.v1 api.v2 worker.tasks events.system

# Results in subjects:
# - api.v1.0
# - api.v2.1
# - worker.tasks.2
# - events.system.3
# - api.v1.4
# - api.v2.5
```

**Combining Modes**

```bash
# 15 publishers with mixed patterns
python publish.py \
  --normal 10 --normal-fuzz --normal-depth-max 4 --normal-interval 500 \
  --js 5 --js-bases payments.prod orders.prod events.prod \
  --verbose
```

This creates:
- 10 normal publishers generating random subjects (1-4 levels deep)
- 5 JetStream publishers rotating through 3 fixed bases
- Publishing every 500ms
- Verbose output to see what's being published

### Publisher Types

| Flag | Type | Description |
|------|------|-------------|
| `--normal N` | Core NATS | Standard pub/sub |
| `--js N` | JetStream | Persistent messaging |
| `--reqrep N` | Request-Reply | Sync request/response |
| `--kv N` | Key-Value | JetStream KV store |
| `--obj N` | Object Store | JetStream object storage |

### Key Options

#### Connection & General
- `--url` - NATS server URL (default: `nats://localhost:4222`)
- `--verbose` / `-v` - Log every message
- `--stats-interval` - Stats reporting interval in seconds (default: 5)
- `--msg-size` - Message payload size in bytes (default: 128)
- `--generate-config` - Output sample JSON config

#### Per-Publisher Type Options

For each publisher type (`normal`, `js`, `reqrep`, `kv`, `obj`):

**Basic:**
- `--{type}-subject` - Subject prefix (default varies by type)
- `--{type}-interval` - Publish interval in ms

**Subject Diversity (Normal & JetStream only):**
- `--{type}-bases` - List of subject bases to rotate through
  - Example: `--normal-bases api.v1 api.v2 events`
- `--{type}-fuzz` - Enable random subject generation
- `--{type}-depth-min` - Minimum subject hierarchy depth (default: 1)
- `--{type}-depth-max` - Maximum subject hierarchy depth (default: 4)

Run `python publish.py --help` for all options.

### Use Cases

**Testing Subject Discovery & Filtering**
```bash
# Generate lots of diverse subjects to test nats-ls filtering
python publish.py --normal 20 --normal-fuzz --normal-depth-max 5
```

**Simulating Multi-Service Architecture**
```bash
# Simulate different services publishing to their own prefixes
python publish.py --normal 10 --normal-bases \
  auth.service orders.service payments.service inventory.service
```

**Load Testing**
```bash
# High-frequency publishing with many subjects
python publish.py --normal 50 --normal-fuzz \
  --normal-interval 100 --normal-depth-max 3
```

**Realistic Production-like Traffic**
```bash
# Mix of fixed and random subjects at varying rates
python publish.py \
  --normal 5 --normal-bases api.v1.prod api.v2.prod \
  --js 10 --js-fuzz --js-depth-max 4 --js-interval 200 \
  --verbose
```

## Requirements for Test Scripts

- Python 3.9+
- `nats-py` package (`pip install nats-py`)
- Running NATS server (JetStream enabled for `--js`, `--kv`, `--obj`)