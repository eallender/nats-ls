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

#### Subject Diversity

The publisher supports multiple ways to generate diverse subject patterns for testing:

**Random Subject Fuzzing**

Generate random, realistic subject hierarchies. Each publisher creates a fixed pool of random subjects and rotates through them, allowing you to see message flow on the same subjects:

```bash
# Enable fuzzing for all publishers (pool of 3 subjects per publisher)
python publish.py --normal 10 --fuzz

# Larger pool for more variety per publisher
python publish.py --normal 10 --fuzz --fuzz-pool 5

# Control hierarchy depth (1-5 levels deep)
python publish.py --normal 10 --fuzz --fuzz-depth-max 5 --fuzz-depth-min 2

# Works with all publisher types
python publish.py --normal 5 --js 5 --fuzz --fuzz-pool 4
```

Example subjects generated in fuzz mode:
- `users.api.create`
- `payments.v2.processor.id42`
- `events.staging.logs`
- `orders.service.update.v1`

**How Fuzzing Works:**
- Each publisher generates a pool of N random subjects at startup (default: 3)
- The publisher rotates through its pool, so you see repeated messages on the same subjects
- This lets you observe message flow patterns instead of creating thousands of unique subjects

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
# Mix fuzzing with different publisher types
python publish.py \
  --normal 10 --js 5 --fuzz --fuzz-pool 4 --fuzz-depth-max 4 \
  --normal-interval 500 --verbose
```

This creates:
- 10 normal publishers, each with a pool of 4 random subjects (1-4 levels deep)
- 5 JetStream publishers, each with a pool of 4 random subjects
- Normal publishers publishing every 500ms
- Verbose output to see what's being published

You can also mix fuzzing with subject bases:
```bash
# Some publishers use bases, enable fuzzing for variety
python publish.py \
  --normal 5 --normal-bases api.v1 api.v2 \
  --js 5 --fuzz --fuzz-pool 3
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

#### Connection & General
- `--url` - NATS server URL (default: `nats://localhost:4222`)
- `--verbose` / `-v` - Log every message
- `--stats-interval` - Stats reporting interval in seconds (default: 5)
- `--msg-size` - Message payload size in bytes (default: 128)
- `--generate-config` - Output sample JSON config

#### Fuzzing Options (applies to all publisher types)
- `--fuzz` - Enable random subject generation for all publishers
- `--fuzz-pool` - Number of random subjects per publisher (default: 3)
- `--fuzz-depth-min` - Minimum subject hierarchy depth (default: 1)
- `--fuzz-depth-max` - Maximum subject hierarchy depth (default: 4)

#### Per-Publisher Type Options

For each publisher type (`normal`, `js`, `reqrep`, `kv`, `obj`):

**Basic:**
- `--{type}-subject` - Subject prefix (default varies by type)
- `--{type}-interval` - Publish interval in ms

**Subject Bases (Normal & JetStream only):**
- `--{type}-bases` - List of subject bases to rotate through
  - Example: `--normal-bases api.v1 api.v2 events`

Run `python publish.py --help` for all options.

### Use Cases

**Testing Subject Discovery & Filtering**
```bash
# Generate lots of diverse subjects to test nats-ls filtering
# 20 publishers × 5 subjects each = 100 unique subjects with repeated messages
python publish.py --normal 20 --fuzz --fuzz-pool 5 --fuzz-depth-max 5
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
# 50 publishers × 3 subjects each = 150 subjects with repeated messages
python publish.py --normal 50 --fuzz \
  --normal-interval 100 --fuzz-depth-max 3
```

**Realistic Production-like Traffic**
```bash
# Mix of fixed and random subjects at varying rates
python publish.py \
  --normal 5 --normal-bases api.v1.prod api.v2.prod \
  --js 10 --fuzz --fuzz-pool 4 --fuzz-depth-max 4 --js-interval 200 \
  --verbose
```

## Requirements for Test Scripts

- Python 3.9+
- `nats-py` package (`pip install nats-py`)
- Running NATS server (JetStream enabled for `--js`, `--kv`, `--obj`)