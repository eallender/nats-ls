# SPDX-License-Identifier: Apache-2.0
# Copyright Evan Allender

import asyncio
import sys

try:
    import nats
    from nats.errors import TimeoutError as NatsTimeoutError
except ImportError:
    print("Error: nats-py is required. Install with: pip install nats-py")
    sys.exit(1)

from config import Config
from payload import create_payload
from stats import Stats
from subjects import generate_random_subject, get_subject_for_publisher, random_string


async def run_normal_publisher(
    nc: nats.NATS,
    publisher_id: int,
    payload_type: str,
    config: Config,
    stats: Stats,
    stop_event: asyncio.Event,
):
    """Run a normal (core NATS) publisher."""
    interval = config.normal_interval_ms / 1000.0
    sequence = 0

    subject_pool = []
    if config.fuzz:
        subject_pool = [
            f"{generate_random_subject(config.fuzz_depth_min, config.fuzz_depth_max)}.{payload_type}"
            for _ in range(config.fuzz_pool_size)
        ]
        if config.verbose:
            print(f"[Normal-{publisher_id}/{payload_type}] Started fuzzing with pool of {config.fuzz_pool_size} subjects: {subject_pool}")
    elif config.verbose:
        print(f"[Normal-{publisher_id}/{payload_type}] Started publishing every {config.normal_interval_ms}ms")

    while not stop_event.is_set():
        try:
            sequence += 1

            if config.fuzz:
                subject = subject_pool[sequence % len(subject_pool)]
            else:
                base = get_subject_for_publisher(
                    publisher_id, sequence,
                    config.normal_subject_prefix, config.normal_subject_bases,
                    False, config.fuzz_depth_min, config.fuzz_depth_max,
                )
                subject = f"{base}.{payload_type}"

            msg = create_payload(publisher_id, "normal", sequence, config, payload_type)
            await nc.publish(subject, msg)
            await stats.increment("normal_sent")

            if config.verbose:
                print(f"[Normal-{publisher_id}/{payload_type}] Published seq {sequence} to {subject}")
        except Exception as e:
            await stats.increment("normal_errors")
            if config.verbose:
                print(f"[Normal-{publisher_id}/{payload_type}] Error: {e}")

        try:
            await asyncio.wait_for(stop_event.wait(), timeout=interval)
            break
        except asyncio.TimeoutError:
            pass


async def run_js_publisher(
    js: nats.js.JetStreamContext,
    publisher_id: int,
    payload_type: str,
    config: Config,
    stats: Stats,
    stop_event: asyncio.Event,
):
    """Run a JetStream publisher."""
    interval = config.js_interval_ms / 1000.0
    sequence = 0

    subject_pool = []
    if config.fuzz:
        subject_pool = [
            f"{config.js_subject_prefix}.{generate_random_subject(config.fuzz_depth_min, config.fuzz_depth_max)}.{payload_type}"
            for _ in range(config.fuzz_pool_size)
        ]
        if config.verbose:
            print(f"[JS-{publisher_id}/{payload_type}] Started fuzzing with pool of {config.fuzz_pool_size} subjects: {subject_pool}")
    elif config.verbose:
        print(f"[JS-{publisher_id}/{payload_type}] Started publishing every {config.js_interval_ms}ms")

    while not stop_event.is_set():
        try:
            sequence += 1

            if config.fuzz:
                subject = subject_pool[sequence % len(subject_pool)]
            else:
                base = get_subject_for_publisher(
                    publisher_id, sequence,
                    config.js_subject_prefix, config.js_subject_bases,
                    False, config.fuzz_depth_min, config.fuzz_depth_max,
                )
                subject = f"{base}.{payload_type}"

            msg = create_payload(publisher_id, "jetstream", sequence, config, payload_type)
            await js.publish(subject, msg)
            await stats.increment("js_sent")

            if config.verbose:
                print(f"[JS-{publisher_id}/{payload_type}] Published seq {sequence} to {subject}")
        except Exception as e:
            await stats.increment("js_errors")
            if config.verbose:
                print(f"[JS-{publisher_id}/{payload_type}] Error: {e}")

        try:
            await asyncio.wait_for(stop_event.wait(), timeout=interval)
            break
        except asyncio.TimeoutError:
            pass


async def run_reqrep_publisher(
    nc: nats.NATS,
    publisher_id: int,
    payload_type: str,
    config: Config,
    stats: Stats,
    stop_event: asyncio.Event,
):
    """Run a request-reply publisher."""
    subject = f"{config.reqrep_subject_prefix}.{publisher_id}.{payload_type}"
    interval = config.reqrep_interval_ms / 1000.0
    timeout = config.reqrep_timeout_ms / 1000.0
    sequence = 0

    if config.verbose:
        print(f"[ReqRep-{publisher_id}/{payload_type}] Started requesting to {subject} every {config.reqrep_interval_ms}ms")

    while not stop_event.is_set():
        try:
            sequence += 1
            msg = create_payload(publisher_id, "request-reply", sequence, config, payload_type)
            await nc.request(subject, msg, timeout=timeout)
            await stats.increment("reqrep_sent")

            if config.verbose:
                print(f"[ReqRep-{publisher_id}/{payload_type}] Request seq {sequence} got reply")
        except NatsTimeoutError:
            await stats.increment("reqrep_errors")
            if config.verbose:
                print(f"[ReqRep-{publisher_id}/{payload_type}] Timeout (no responder?)")
        except Exception as e:
            await stats.increment("reqrep_errors")
            if config.verbose:
                print(f"[ReqRep-{publisher_id}/{payload_type}] Error: {e}")

        try:
            await asyncio.wait_for(stop_event.wait(), timeout=interval)
            break
        except asyncio.TimeoutError:
            pass


async def run_kv_publisher(
    kv: nats.js.kv.KeyValue,
    publisher_id: int,
    payload_type: str,
    config: Config,
    stats: Stats,
    stop_event: asyncio.Event,
):
    """Run a Key-Value publisher."""
    key = f"{config.kv_key_prefix}-{publisher_id}-{payload_type}"
    interval = config.kv_interval_ms / 1000.0
    sequence = 0

    if config.verbose:
        print(f"[KV-{publisher_id}/{payload_type}] Started putting to key {key} every {config.kv_interval_ms}ms")

    while not stop_event.is_set():
        try:
            sequence += 1
            msg = create_payload(publisher_id, "kv", sequence, config, payload_type)
            await kv.put(key, msg)
            await stats.increment("kv_sent")

            if config.verbose:
                print(f"[KV-{publisher_id}/{payload_type}] Put seq {sequence} to key {key}")
        except Exception as e:
            await stats.increment("kv_errors")
            if config.verbose:
                print(f"[KV-{publisher_id}/{payload_type}] Error: {e}")

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
        print(f"[Obj-{publisher_id}] Started putting object {obj_name} every {config.obj_interval_ms}ms")

    while not stop_event.is_set():
        try:
            sequence += 1
            data = random_string(config.obj_size_bytes).encode()
            await obs.put(obj_name, data)
            await stats.increment("obj_sent")

            if config.verbose:
                print(f"[Obj-{publisher_id}] Put object {obj_name} seq {sequence} ({config.obj_size_bytes} bytes)")
        except Exception as e:
            await stats.increment("obj_errors")
            if config.verbose:
                print(f"[Obj-{publisher_id}] Error: {e}")

        try:
            await asyncio.wait_for(stop_event.wait(), timeout=interval)
            break
        except asyncio.TimeoutError:
            pass
