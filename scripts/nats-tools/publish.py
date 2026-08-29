# SPDX-License-Identifier: Apache-2.0
# Copyright Evan Allender
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

import asyncio
import signal
import sys

try:
    import nats
except ImportError:
    print("Error: nats-py is required. Install with: pip install nats-py")
    sys.exit(1)

from cli import generate_sample_config, parse_args
from config import Config
from payload import PAYLOAD_TYPES
from publishers import (
    run_js_publisher,
    run_kv_publisher,
    run_normal_publisher,
    run_obj_publisher,
    run_reqrep_publisher,
)
from setup import ensure_kv_bucket, ensure_object_store, ensure_stream
from stats import Stats, print_final_stats, stats_reporter


async def main(config: Config):
    total_publishers = (
        config.normal_publishers
        + config.js_publishers
        + config.reqrep_publishers
        + config.kv_publishers
        + config.obj_publishers
    )

    if total_publishers == 0:
        print("No publishers configured. Use flags or config file to specify publishers.")
        print("Example: publish.py --normal 5 --js 3")
        print("Run with --help for options or --generate-config for a sample config file.")
        sys.exit(1)

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

    js = None
    if config.js_publishers > 0 or config.kv_publishers > 0 or config.obj_publishers > 0:
        js = nc.jetstream()
        print("JetStream context created")

    stop_event = asyncio.Event()

    def signal_handler():
        print("\nShutting down...")
        stop_event.set()

    loop = asyncio.get_event_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, signal_handler)

    stats = Stats()
    tasks = []

    print(f"Starting {total_publishers} publishers...")

    for i in range(config.normal_publishers):
        tasks.append(asyncio.create_task(
            run_normal_publisher(nc, i, PAYLOAD_TYPES[i % len(PAYLOAD_TYPES)], config, stats, stop_event)
        ))

    if config.js_publishers > 0:
        js_subject_patterns = config.js_subject_bases or [config.js_subject_prefix]
        await ensure_stream(js, config.js_stream_name, js_subject_patterns)
        for i in range(config.js_publishers):
            tasks.append(asyncio.create_task(
                run_js_publisher(js, i, PAYLOAD_TYPES[i % len(PAYLOAD_TYPES)], config, stats, stop_event)
            ))

    for i in range(config.reqrep_publishers):
        tasks.append(asyncio.create_task(
            run_reqrep_publisher(nc, i, PAYLOAD_TYPES[i % len(PAYLOAD_TYPES)], config, stats, stop_event)
        ))

    if config.kv_publishers > 0:
        kv = await ensure_kv_bucket(js, config.kv_bucket)
        if kv:
            for i in range(config.kv_publishers):
                tasks.append(asyncio.create_task(
                    run_kv_publisher(kv, i, PAYLOAD_TYPES[i % len(PAYLOAD_TYPES)], config, stats, stop_event)
                ))

    if config.obj_publishers > 0:
        obs = await ensure_object_store(js, config.obj_bucket)
        if obs:
            for i in range(config.obj_publishers):
                tasks.append(asyncio.create_task(
                    run_obj_publisher(obs, i, config, stats, stop_event)
                ))

    if config.stats_interval_sec > 0:
        tasks.append(asyncio.create_task(stats_reporter(stats, config, stop_event)))

    print("All publishers started. Press Ctrl+C to stop.")
    await asyncio.gather(*tasks)

    print_final_stats(stats)
    await nc.close()


if __name__ == "__main__":
    config, generate_only = parse_args()

    if generate_only:
        generate_sample_config()
        sys.exit(0)

    try:
        asyncio.run(main(config))
    except KeyboardInterrupt:
        pass
