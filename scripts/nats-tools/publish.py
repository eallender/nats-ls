#!/usr/bin/env python3
"""
NATS Test Publisher - A flexible tool to spin up multiple NATS publishers
for testing message ingestion in tools like natsls.

Supports:
- Normal (Core NATS) publishers
- JetStream publishers
- Request-Reply publishers
- Key-Value store publishers
- Object Store publishers
"""

import argparse
import asyncio
import json
import random
import signal
import string
import sys
import time
from dataclasses import asdict, dataclass, field
from datetime import datetime
from typing import Optional

try:
    import nats
    from nats.errors import TimeoutError as NatsTimeoutError
    from nats.js.api import KeyValueConfig, ObjectStoreConfig, StreamConfig
except ImportError:
    print("Error: nats-py is required. Install with: pip install nats-py")
    sys.exit(1)


@dataclass
class Config:
    """Configuration for the test publisher tool."""

    nats_url: str = "nats://localhost:4222"

    # Normal publishers
    normal_publishers: int = 0
    normal_subject_prefix: str = "test.normal"
    normal_interval_ms: int = 1000

    # JetStream publishers
    js_publishers: int = 0
    js_subject_prefix: str = "test.js"
    js_stream_name: str = "TEST"
    js_interval_ms: int = 1000

    # Request-Reply publishers
    reqrep_publishers: int = 0
    reqrep_subject_prefix: str = "test.service"
    reqrep_interval_ms: int = 2000
    reqrep_timeout_ms: int = 5000

    # Key-Value publishers
    kv_publishers: int = 0
    kv_bucket: str = "test-bucket"
    kv_key_prefix: str = "test-key"
    kv_interval_ms: int = 1500

    # Object Store publishers
    obj_publishers: int = 0
    obj_bucket: str = "test-objects"
    obj_name_prefix: str = "test-obj"
    obj_interval_ms: int = 5000
    obj_size_bytes: int = 1024

    # Message options
    message_size_bytes: int = 128
    include_timestamp: bool = True
    include_sequence: bool = True

    # Output options
    verbose: bool = False
    stats_interval_sec: int = 5


@dataclass
class Stats:
    """Tracks publishing statistics."""

    normal_sent: int = 0
    normal_errors: int = 0
    js_sent: int = 0
    js_errors: int = 0
    reqrep_sent: int = 0
    reqrep_errors: int = 0
    kv_sent: int = 0
    kv_errors: int = 0
    obj_sent: int = 0
    obj_errors: int = 0
    start_time: float = field(default_factory=time.time)
    _lock: asyncio.Lock = field(default_factory=asyncio.Lock, repr=False)

    async def increment(self, stat_name: str, amount: int = 1):
        async with self._lock:
            current = getattr(self, stat_name)
            setattr(self, stat_name, current + amount)

    def total_sent(self) -> int:
        return (
            self.normal_sent
            + self.js_sent
            + self.reqrep_sent
            + self.kv_sent
            + self.obj_sent
        )

    def total_errors(self) -> int:
        return (
            self.normal_errors
            + self.js_errors
            + self.reqrep_errors
            + self.kv_errors
            + self.obj_errors
        )

    def rate(self) -> float:
        elapsed = time.time() - self.start_time
        if elapsed > 0:
            return self.total_sent() / elapsed
        return 0.0


def random_string(length: int) -> str:
    """Generate a random alphanumeric string."""
    return "".join(random.choices(string.ascii_letters + string.digits, k=length))


def create_message(
    publisher_id: int, publisher_type: str, sequence: int, config: Config
) -> bytes:
    """Create a JSON message payload."""
    msg = {
        "publisher_id": publisher_id,
        "publisher_type": publisher_type,
        "data": random_string(config.message_size_bytes),
    }
    if config.include_sequence:
        msg["sequence"] = sequence
    if config.include_timestamp:
        msg["timestamp"] = datetime.utcnow().isoformat() + "Z"
    return json.dumps(msg).encode()


async def run_normal_publisher(
    nc: nats.NATS,
    publisher_id: int,
    config: Config,
    stats: Stats,
    stop_event: asyncio.Event,
):
    """Run a normal (core NATS) publisher."""
    subject = f"{config.normal_subject_prefix}.{publisher_id}"
    interval = config.normal_interval_ms / 1000.0
    sequence = 0

    if config.verbose:
        print(
            f"[Normal-{publisher_id}] Started publishing to {subject} every {config.normal_interval_ms}ms"
        )

    while not stop_event.is_set():
        try:
            sequence += 1
            msg = create_message(publisher_id, "normal", sequence, config)
            await nc.publish(subject, msg)
            await stats.increment("normal_sent")

            if config.verbose:
                print(f"[Normal-{publisher_id}] Published seq {sequence} to {subject}")
        except Exception as e:
            await stats.increment("normal_errors")
            if config.verbose:
                print(f"[Normal-{publisher_id}] Error: {e}")

        try:
            await asyncio.wait_for(stop_event.wait(), timeout=interval)
            break
        except asyncio.TimeoutError:
            pass


async def run_js_publisher(
    js: nats.js.JetStreamContext,
    publisher_id: int,
    config: Config,
    stats: Stats,
    stop_event: asyncio.Event,
):
    """Run a JetStream publisher."""
    subject = f"{config.js_subject_prefix}.{publisher_id}"
    interval = config.js_interval_ms / 1000.0
    sequence = 0

    if config.verbose:
        print(
            f"[JS-{publisher_id}] Started publishing to {subject} every {config.js_interval_ms}ms"
        )

    while not stop_event.is_set():
        try:
            sequence += 1
            msg = create_message(publisher_id, "jetstream", sequence, config)
            await js.publish(subject, msg)
            await stats.increment("js_sent")

            if config.verbose:
                print(f"[JS-{publisher_id}] Published seq {sequence} to {subject}")
        except Exception as e:
            await stats.increment("js_errors")
            if config.verbose:
                print(f"[JS-{publisher_id}] Error: {e}")

        try:
            await asyncio.wait_for(stop_event.wait(), timeout=interval)
            break
        except asyncio.TimeoutError:
            pass


async def run_reqrep_publisher(
    nc: nats.NATS,
    publisher_id: int,
    config: Config,
    stats: Stats,
    stop_event: asyncio.Event,
):
    """Run a request-reply publisher."""
    subject = f"{config.reqrep_subject_prefix}.{publisher_id}"
    interval = config.reqrep_interval_ms / 1000.0
    timeout = config.reqrep_timeout_ms / 1000.0
    sequence = 0

    if config.verbose:
        print(
            f"[ReqRep-{publisher_id}] Started requesting to {subject} every {config.reqrep_interval_ms}ms"
        )

    while not stop_event.is_set():
        try:
            sequence += 1
            msg = create_message(publisher_id, "request-reply", sequence, config)
            await nc.request(subject, msg, timeout=timeout)
            await stats.increment("reqrep_sent")

            if config.verbose:
                print(f"[ReqRep-{publisher_id}] Request seq {sequence} got reply")
        except NatsTimeoutError:
            await stats.increment("reqrep_errors")
            if config.verbose:
                print(f"[ReqRep-{publisher_id}] Timeout (no responder?)")
        except Exception as e:
            await stats.increment("reqrep_errors")
            if config.verbose:
                print(f"[ReqRep-{publisher_id}] Error: {e}")

        try:
            await asyncio.wait_for(stop_event.wait(), timeout=interval)
            break
        except asyncio.TimeoutError:
            pass


async def run_kv_publisher(
    kv: nats.js.kv.KeyValue,
    publisher_id: int,
    config: Config,
    stats: Stats,
    stop_event: asyncio.Event,
):
    """Run a Key-Value publisher."""
    key = f"{config.kv_key_prefix}-{publisher_id}"
    interval = config.kv_interval_ms / 1000.0
    sequence = 0

    if config.verbose:
        print(
            f"[KV-{publisher_id}] Started putting to key {key} every {config.kv_interval_ms}ms"
        )

    while not stop_event.is_set():
        try:
            sequence += 1
            msg = create_message(publisher_id, "kv", sequence, config)
            await kv.put(key, msg)
            await stats.increment("kv_sent")

            if config.verbose:
                print(f"[KV-{publisher_id}] Put seq {sequence} to key {key}")
        except Exception as e:
            await stats.increment("kv_errors")
            if config.verbose:
                print(f"[KV-{publisher_id}] Error: {e}")

        try:
            await asyncio.wait_for(stop_event.wait(), timeout=interval)
            break
        except asyncio.TimeoutError:
            pass


async def run_obj_publisher(
    obs: nats.js.object_store.ObjectStore,
    publisher_id: int,
    config: Config,
    stats: Stats,
    stop_event: asyncio.Event,
):
    """Run an Object Store publisher."""
    obj_name = f"{config.obj_name_prefix}-{publisher_id}"
    interval = config.obj_interval_ms / 1000.0
    sequence = 0

    if config.verbose:
        print(
            f"[Obj-{publisher_id}] Started putting object {obj_name} every {config.obj_interval_ms}ms"
        )

    while not stop_event.is_set():
        try:
            sequence += 1
            data = random_string(config.obj_size_bytes).encode()
            await obs.put(obj_name, data)
            await stats.increment("obj_sent")

            if config.verbose:
                print(
                    f"[Obj-{publisher_id}] Put object {obj_name} seq {sequence} ({config.obj_size_bytes} bytes)"
                )
        except Exception as e:
            await stats.increment("obj_errors")
            if config.verbose:
                print(f"[Obj-{publisher_id}] Error: {e}")

        try:
            await asyncio.wait_for(stop_event.wait(), timeout=interval)
            break
        except asyncio.TimeoutError:
            pass


async def stats_reporter(stats: Stats, config: Config, stop_event: asyncio.Event):
    """Periodically report statistics."""
    while not stop_event.is_set():
        try:
            await asyncio.wait_for(stop_event.wait(), timeout=config.stats_interval_sec)
            break
        except asyncio.TimeoutError:
            print(
                f"Stats: Normal={stats.normal_sent} JS={stats.js_sent} "
                f"ReqRep={stats.reqrep_sent} KV={stats.kv_sent} Obj={stats.obj_sent} | "
                f"Total={stats.total_sent()} | Rate={stats.rate():.1f} msg/s"
            )


def print_final_stats(stats: Stats):
    """Print final statistics."""
    elapsed = time.time() - stats.start_time

    print("\n" + "=" * 42)
    print("           Final Statistics")
    print("=" * 42)
    print(f"Runtime: {elapsed:.1f} seconds\n")

    print("Messages Sent:")
    print(f"  Normal:        {stats.normal_sent:>8} (errors: {stats.normal_errors})")
    print(f"  JetStream:     {stats.js_sent:>8} (errors: {stats.js_errors})")
    print(f"  Request-Reply: {stats.reqrep_sent:>8} (errors: {stats.reqrep_errors})")
    print(f"  Key-Value:     {stats.kv_sent:>8} (errors: {stats.kv_errors})")
    print(f"  Object Store:  {stats.obj_sent:>8} (errors: {stats.obj_errors})")
    print()
    print(f"Total: {stats.total_sent()} messages (errors: {stats.total_errors()})")
    print(f"Average Rate: {stats.rate():.2f} msg/s")
    print("=" * 42)


async def ensure_stream(js: nats.js.JetStreamContext, name: str, subject_prefix: str):
    """Ensure a JetStream stream exists."""
    try:
        await js.add_stream(
            name=name,
            subjects=[f"{subject_prefix}.>"],
            storage="memory",
            max_age=3600,  # 1 hour
        )
        print(f"Created stream: {name}")
    except nats.js.errors.BadRequestError:
        # Stream already exists
        pass
    except Exception as e:
        print(f"Warning: Could not ensure stream exists: {e}")


async def ensure_kv_bucket(js: nats.js.JetStreamContext, bucket: str):
    """Ensure a KV bucket exists."""
    try:
        kv = await js.create_key_value(bucket=bucket, ttl=3600)
        print(f"Created KV bucket: {bucket}")
        return kv
    except nats.js.errors.BadRequestError:
        # Bucket already exists
        return await js.key_value(bucket)
    except Exception as e:
        print(f"Warning: Could not ensure KV bucket exists: {e}")
        return None


async def ensure_object_store(js: nats.js.JetStreamContext, bucket: str):
    """Ensure an Object Store bucket exists."""
    try:
        obs = await js.create_object_store(bucket=bucket, ttl=3600)
        print(f"Created Object Store bucket: {bucket}")
        return obs
    except nats.js.errors.BadRequestError:
        # Bucket already exists
        return await js.object_store(bucket)
    except Exception as e:
        print(f"Warning: Could not ensure Object Store bucket exists: {e}")
        return None


async def main(config: Config):
    """Main entry point."""
    # Validate we have at least one publisher
    total_publishers = (
        config.normal_publishers
        + config.js_publishers
        + config.reqrep_publishers
        + config.kv_publishers
        + config.obj_publishers
    )

    if total_publishers == 0:
        print(
            "No publishers configured. Use flags or config file to specify publishers."
        )
        print("Example: nats_test_publisher.py --normal 5 --js 3")
        print(
            "Run with --help for options or --generate-config for a sample config file."
        )
        sys.exit(1)

    # Connect to NATS
    print(f"Connecting to NATS at {config.nats_url}...")
    try:
        nc = await nats.connect(
            config.nats_url,
            name="nats-test-publisher",
            reconnect_time_wait=1,
            max_reconnect_attempts=-1,
        )
    except Exception as e:
        print(f"Failed to connect to NATS: {e}")
        sys.exit(1)

    print("Connected to NATS")

    # Get JetStream context if needed
    js = None
    if (
        config.js_publishers > 0
        or config.kv_publishers > 0
        or config.obj_publishers > 0
    ):
        js = nc.jetstream()
        print("JetStream context created")

    # Setup stop event
    stop_event = asyncio.Event()

    # Handle signals
    def signal_handler():
        print("\nShutting down...")
        stop_event.set()

    loop = asyncio.get_event_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, signal_handler)

    # Initialize stats
    stats = Stats()

    # Collect all publisher tasks
    tasks = []

    print(f"Starting {total_publishers} publishers...")

    # Normal publishers
    for i in range(config.normal_publishers):
        tasks.append(
            asyncio.create_task(run_normal_publisher(nc, i, config, stats, stop_event))
        )

    # JetStream publishers
    if config.js_publishers > 0:
        await ensure_stream(js, config.js_stream_name, config.js_subject_prefix)
        for i in range(config.js_publishers):
            tasks.append(
                asyncio.create_task(run_js_publisher(js, i, config, stats, stop_event))
            )

    # Request-Reply publishers
    for i in range(config.reqrep_publishers):
        tasks.append(
            asyncio.create_task(run_reqrep_publisher(nc, i, config, stats, stop_event))
        )

    # KV publishers
    if config.kv_publishers > 0:
        kv = await ensure_kv_bucket(js, config.kv_bucket)
        if kv:
            for i in range(config.kv_publishers):
                tasks.append(
                    asyncio.create_task(
                        run_kv_publisher(kv, i, config, stats, stop_event)
                    )
                )

    # Object Store publishers
    if config.obj_publishers > 0:
        obs = await ensure_object_store(js, config.obj_bucket)
        if obs:
            for i in range(config.obj_publishers):
                tasks.append(
                    asyncio.create_task(
                        run_obj_publisher(obs, i, config, stats, stop_event)
                    )
                )

    # Stats reporter
    if config.stats_interval_sec > 0:
        tasks.append(asyncio.create_task(stats_reporter(stats, config, stop_event)))

    print("All publishers started. Press Ctrl+C to stop.")

    # Wait for all tasks to complete
    await asyncio.gather(*tasks)

    # Print final stats
    print_final_stats(stats)

    # Close connection
    await nc.close()


def load_config(path: str) -> Config:
    """Load configuration from a JSON file."""
    with open(path, "r") as f:
        data = json.load(f)
    return Config(**data)


def generate_sample_config():
    """Generate and print a sample configuration file."""
    config = Config(
        normal_publishers=5,
        js_publishers=3,
        reqrep_publishers=2,
        kv_publishers=2,
        obj_publishers=1,
    )
    print(json.dumps(asdict(config), indent=2))


def parse_args() -> tuple[Config, bool]:
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(
        description="NATS Test Publisher - Spin up multiple NATS publishers for testing",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s --normal 5                    # 5 normal publishers
  %(prog)s --normal 5 --js 3             # 5 normal + 3 JetStream
  %(prog)s --normal 10 --verbose         # 10 normal with verbose output
  %(prog)s --config config.json          # Use config file
  %(prog)s --generate-config > cfg.json  # Generate sample config
        """,
    )

    # Connection
    parser.add_argument(
        "--url",
        default="nats://localhost:4222",
        help="NATS server URL (default: nats://localhost:4222)",
    )
    parser.add_argument("--config", dest="config_file", help="Path to JSON config file")

    # Normal publishers
    normal = parser.add_argument_group("Normal Publishers (Core NATS)")
    normal.add_argument(
        "--normal", type=int, default=0, help="Number of normal publishers"
    )
    normal.add_argument(
        "--normal-subject",
        default="test.normal",
        help="Subject prefix (default: test.normal)",
    )
    normal.add_argument(
        "--normal-interval",
        type=int,
        default=1000,
        help="Publish interval in ms (default: 1000)",
    )

    # JetStream publishers
    js = parser.add_argument_group("JetStream Publishers")
    js.add_argument("--js", type=int, default=0, help="Number of JetStream publishers")
    js.add_argument(
        "--js-subject", default="test.js", help="Subject prefix (default: test.js)"
    )
    js.add_argument("--js-stream", default="TEST", help="Stream name (default: TEST)")
    js.add_argument(
        "--js-interval",
        type=int,
        default=1000,
        help="Publish interval in ms (default: 1000)",
    )

    # Request-Reply publishers
    reqrep = parser.add_argument_group("Request-Reply Publishers")
    reqrep.add_argument(
        "--reqrep", type=int, default=0, help="Number of request-reply publishers"
    )
    reqrep.add_argument(
        "--reqrep-subject",
        default="test.service",
        help="Subject prefix (default: test.service)",
    )
    reqrep.add_argument(
        "--reqrep-interval",
        type=int,
        default=2000,
        help="Request interval in ms (default: 2000)",
    )
    reqrep.add_argument(
        "--reqrep-timeout",
        type=int,
        default=5000,
        help="Request timeout in ms (default: 5000)",
    )

    # KV publishers
    kv = parser.add_argument_group("Key-Value Publishers")
    kv.add_argument("--kv", type=int, default=0, help="Number of KV publishers")
    kv.add_argument(
        "--kv-bucket",
        default="test-bucket",
        help="KV bucket name (default: test-bucket)",
    )
    kv.add_argument(
        "--kv-key", default="test-key", help="Key prefix (default: test-key)"
    )
    kv.add_argument(
        "--kv-interval",
        type=int,
        default=1500,
        help="Put interval in ms (default: 1500)",
    )

    # Object Store publishers
    obj = parser.add_argument_group("Object Store Publishers")
    obj.add_argument(
        "--obj", type=int, default=0, help="Number of Object Store publishers"
    )
    obj.add_argument(
        "--obj-bucket",
        default="test-objects",
        help="Object Store bucket (default: test-objects)",
    )
    obj.add_argument(
        "--obj-name", default="test-obj", help="Object name prefix (default: test-obj)"
    )
    obj.add_argument(
        "--obj-interval",
        type=int,
        default=5000,
        help="Put interval in ms (default: 5000)",
    )
    obj.add_argument(
        "--obj-size",
        type=int,
        default=1024,
        help="Object size in bytes (default: 1024)",
    )

    # Message options
    msg = parser.add_argument_group("Message Options")
    msg.add_argument(
        "--msg-size",
        type=int,
        default=128,
        help="Message payload size in bytes (default: 128)",
    )

    # Output options
    out = parser.add_argument_group("Output Options")
    out.add_argument(
        "--verbose", "-v", action="store_true", help="Enable verbose logging"
    )
    out.add_argument(
        "--stats-interval",
        type=int,
        default=5,
        help="Stats reporting interval in seconds (default: 5, 0 to disable)",
    )

    # Utility
    parser.add_argument(
        "--generate-config",
        action="store_true",
        help="Generate a sample config file and exit",
    )

    args = parser.parse_args()

    if args.generate_config:
        return None, True

    # Build config
    if args.config_file:
        config = load_config(args.config_file)
    else:
        config = Config(
            nats_url=args.url,
            normal_publishers=args.normal,
            normal_subject_prefix=args.normal_subject,
            normal_interval_ms=args.normal_interval,
            js_publishers=args.js,
            js_subject_prefix=args.js_subject,
            js_stream_name=args.js_stream,
            js_interval_ms=args.js_interval,
            reqrep_publishers=args.reqrep,
            reqrep_subject_prefix=args.reqrep_subject,
            reqrep_interval_ms=args.reqrep_interval,
            reqrep_timeout_ms=args.reqrep_timeout,
            kv_publishers=args.kv,
            kv_bucket=args.kv_bucket,
            kv_key_prefix=args.kv_key,
            kv_interval_ms=args.kv_interval,
            obj_publishers=args.obj,
            obj_bucket=args.obj_bucket,
            obj_name_prefix=args.obj_name,
            obj_interval_ms=args.obj_interval,
            obj_size_bytes=args.obj_size,
            message_size_bytes=args.msg_size,
            verbose=args.verbose,
            stats_interval_sec=args.stats_interval,
        )

    return config, False


if __name__ == "__main__":
    config, generate_only = parse_args()

    if generate_only:
        generate_sample_config()
        sys.exit(0)

    try:
        asyncio.run(main(config))
    except KeyboardInterrupt:
        pass
