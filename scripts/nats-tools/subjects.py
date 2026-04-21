# SPDX-License-Identifier: Apache-2.0
# Copyright Evan Allender

import random
import string


def random_string(length: int) -> str:
    """Generate a random alphanumeric string."""
    return "".join(random.choices(string.ascii_letters + string.digits, k=length))


def generate_subject_component() -> str:
    """Generate a random subject component (token between dots)."""
    patterns = [
        lambda: random.choice(["users", "orders", "payments", "events", "logs", "metrics", "alerts"]),
        lambda: random.choice(["api", "service", "worker", "handler", "processor"]),
        lambda: random.choice(["v1", "v2", "v3", "prod", "staging", "dev"]),
        lambda: random.choice(["create", "update", "delete", "read", "list"]),
        lambda: f"id{random.randint(1, 999)}",
        lambda: random_string(random.randint(4, 10)).lower(),
    ]
    return random.choice(patterns)()


def generate_random_subject(depth_min: int = 1, depth_max: int = 4) -> str:
    """Generate a random subject with realistic hierarchy."""
    depth = random.randint(depth_min, depth_max)
    components = [generate_subject_component() for _ in range(depth)]
    return ".".join(components)


def get_subject_for_publisher(
    publisher_id: int,
    sequence: int,
    prefix: str,
    bases: list[str],
    fuzz: bool,
    depth_min: int,
    depth_max: int,
) -> str:
    """Return the subject to publish to based on configuration."""
    if fuzz:
        return generate_random_subject(depth_min, depth_max)
    elif bases:
        base = bases[publisher_id % len(bases)]
        return f"{base}.{publisher_id}"
    else:
        return f"{prefix}.{publisher_id}"
