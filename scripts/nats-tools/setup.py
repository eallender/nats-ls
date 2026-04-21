# SPDX-License-Identifier: Apache-2.0
# Copyright Evan Allender

import sys

try:
    import nats
except ImportError:
    print("Error: nats-py is required. Install with: pip install nats-py")
    sys.exit(1)


async def ensure_stream(
    js: nats.js.JetStreamContext,
    name: str,
    subject_pattern: str,
):
    """Ensure a JetStream stream exists."""
    try:
        # Use subject prefix with wildcard to avoid conflicts with KV/Object Store
        # Never use ">" as it claims ALL subjects and conflicts with $KV.> and $O.>
        subjects = [f"{subject_pattern}.>"]
        await js.add_stream(
            name=name,
            subjects=subjects,
            storage="memory",
            max_age=3600,
        )
        print(f"Created stream: {name} (subjects: {subjects})")
    except nats.js.errors.BadRequestError:
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
        return await js.object_store(bucket)
    except Exception as e:
        print(f"Warning: Could not ensure Object Store bucket exists: {e}")
        return None
