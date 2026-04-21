# SPDX-License-Identifier: Apache-2.0
# Copyright Evan Allender

from dataclasses import dataclass, field


@dataclass
class Config:
    """Configuration for the test publisher tool."""

    nats_url: str = "nats://localhost:4222"

    # Fuzzing options (shared across all publisher types)
    fuzz: bool = False
    fuzz_pool_size: int = 3
    fuzz_depth_min: int = 1
    fuzz_depth_max: int = 4

    # Normal publishers
    normal_publishers: int = 0
    normal_subject_prefix: str = "test.normal"
    normal_subject_bases: list[str] = field(default_factory=list)
    normal_interval_ms: int = 1000

    # JetStream publishers
    js_publishers: int = 0
    js_subject_prefix: str = "test.js"
    js_subject_bases: list[str] = field(default_factory=list)
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
