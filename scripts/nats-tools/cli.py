# SPDX-License-Identifier: Apache-2.0
# Copyright Evan Allender

import argparse
import json
from dataclasses import asdict

from config import Config


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
  %(prog)s --normal 5                                    # 5 normal publishers
  %(prog)s --normal 5 --js 3                             # 5 normal + 3 JetStream
  %(prog)s --normal 10 --verbose                         # 10 normal with verbose output
  %(prog)s --normal 5 --fuzz                             # 5 publishers with random subjects (pool of 3 each)
  %(prog)s --normal 5 --js 3 --fuzz --fuzz-pool 5        # All publishers use random subjects, pool of 5 each
  %(prog)s --normal 3 --normal-bases api.v1 api.v2 events  # 3 publishers rotating through bases
  %(prog)s --normal 10 --fuzz --fuzz-depth-max 5         # Deep random hierarchies for all publishers
  %(prog)s --config config.json                          # Use config file
  %(prog)s --generate-config > cfg.json                  # Generate sample config
        """,
    )

    parser.add_argument("--url", default="nats://localhost:4222", help="NATS server URL (default: nats://localhost:4222)")
    parser.add_argument("--config", dest="config_file", help="Path to JSON config file")

    fuzz_group = parser.add_argument_group("Fuzzing Options (applies to all publishers)")
    fuzz_group.add_argument("--fuzz", action="store_true", help="Enable random subject generation for all publishers")
    fuzz_group.add_argument("--fuzz-pool", type=int, default=3, help="Number of random subjects per publisher when fuzzing (default: 3)")
    fuzz_group.add_argument("--fuzz-depth-min", type=int, default=1, help="Minimum subject depth for fuzzing (default: 1)")
    fuzz_group.add_argument("--fuzz-depth-max", type=int, default=4, help="Maximum subject depth for fuzzing (default: 4)")

    normal = parser.add_argument_group("Normal Publishers (Core NATS)")
    normal.add_argument("--normal", type=int, default=0, help="Number of normal publishers")
    normal.add_argument("--normal-subject", default="test.normal", help="Subject prefix (default: test.normal)")
    normal.add_argument("--normal-bases", nargs="+", help="Multiple subject bases to rotate through (e.g., api.v1 api.v2 worker.tasks)")
    normal.add_argument("--normal-interval", type=int, default=1000, help="Publish interval in ms (default: 1000)")

    js = parser.add_argument_group("JetStream Publishers")
    js.add_argument("--js", type=int, default=0, help="Number of JetStream publishers")
    js.add_argument("--js-subject", default="test.js", help="Subject prefix (default: test.js)")
    js.add_argument("--js-bases", nargs="+", help="Multiple subject bases to rotate through")
    js.add_argument("--js-stream", default="TEST", help="Stream name (default: TEST)")
    js.add_argument("--js-interval", type=int, default=1000, help="Publish interval in ms (default: 1000)")

    reqrep = parser.add_argument_group("Request-Reply Publishers")
    reqrep.add_argument("--reqrep", type=int, default=0, help="Number of request-reply publishers")
    reqrep.add_argument("--reqrep-subject", default="test.service", help="Subject prefix (default: test.service)")
    reqrep.add_argument("--reqrep-interval", type=int, default=2000, help="Request interval in ms (default: 2000)")
    reqrep.add_argument("--reqrep-timeout", type=int, default=5000, help="Request timeout in ms (default: 5000)")

    kv = parser.add_argument_group("Key-Value Publishers")
    kv.add_argument("--kv", type=int, default=0, help="Number of KV publishers")
    kv.add_argument("--kv-bucket", default="test-bucket", help="KV bucket name (default: test-bucket)")
    kv.add_argument("--kv-key", default="test-key", help="Key prefix (default: test-key)")
    kv.add_argument("--kv-interval", type=int, default=1500, help="Put interval in ms (default: 1500)")

    obj = parser.add_argument_group("Object Store Publishers")
    obj.add_argument("--obj", type=int, default=0, help="Number of Object Store publishers")
    obj.add_argument("--obj-bucket", default="test-objects", help="Object Store bucket (default: test-objects)")
    obj.add_argument("--obj-name", default="test-obj", help="Object name prefix (default: test-obj)")
    obj.add_argument("--obj-interval", type=int, default=5000, help="Put interval in ms (default: 5000)")
    obj.add_argument("--obj-size", type=int, default=1024, help="Object size in bytes (default: 1024)")

    msg = parser.add_argument_group("Message Options")
    msg.add_argument("--msg-size", type=int, default=128, help="Message payload size in bytes (default: 128)")

    out = parser.add_argument_group("Output Options")
    out.add_argument("--verbose", "-v", action="store_true", help="Enable verbose logging")
    out.add_argument("--stats-interval", type=int, default=5, help="Stats reporting interval in seconds (default: 5, 0 to disable)")

    parser.add_argument("--generate-config", action="store_true", help="Generate a sample config file and exit")

    args = parser.parse_args()

    if args.fuzz_pool < 1:
        parser.error("--fuzz-pool must be at least 1")
    if args.fuzz_depth_min < 1:
        parser.error("--fuzz-depth-min must be at least 1")
    if args.fuzz_depth_min > args.fuzz_depth_max:
        parser.error("--fuzz-depth-min must be less than or equal to --fuzz-depth-max")

    publisher_counts = {
        "--normal": args.normal,
        "--js": args.js,
        "--reqrep": args.reqrep,
        "--kv": args.kv,
        "--obj": args.obj,
    }
    for flag, count in publisher_counts.items():
        if count < 0:
            parser.error(f"{flag} must be non-negative")

    positive_intervals = {
        "--normal-interval": args.normal_interval,
        "--js-interval": args.js_interval,
        "--reqrep-interval": args.reqrep_interval,
        "--reqrep-timeout": args.reqrep_timeout,
        "--kv-interval": args.kv_interval,
        "--obj-interval": args.obj_interval,
    }
    for flag, value in positive_intervals.items():
        if value <= 0:
            parser.error(f"{flag} must be greater than 0")

    if args.stats_interval < 0:
        parser.error("--stats-interval must be non-negative")
    if args.msg_size < 0:
        parser.error("--msg-size must be non-negative")
    if args.obj_size < 0:
        parser.error("--obj-size must be non-negative")

    if args.generate_config:
        return None, True

    if args.config_file:
        config = load_config(args.config_file)
    else:
        config = Config(
            nats_url=args.url,
            fuzz=args.fuzz,
            fuzz_pool_size=args.fuzz_pool,
            fuzz_depth_min=args.fuzz_depth_min,
            fuzz_depth_max=args.fuzz_depth_max,
            normal_publishers=args.normal,
            normal_subject_prefix=args.normal_subject,
            normal_subject_bases=args.normal_bases or [],
            normal_interval_ms=args.normal_interval,
            js_publishers=args.js,
            js_subject_prefix=args.js_subject,
            js_subject_bases=args.js_bases or [],
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
