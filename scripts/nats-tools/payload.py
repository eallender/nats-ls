# SPDX-License-Identifier: Apache-2.0
# Copyright Evan Allender

import json
import random
from datetime import datetime

from config import Config
from subjects import random_string

PAYLOAD_TYPES = ["json", "text", "binary"]


def create_json_payload(publisher_id: int, publisher_type: str, sequence: int, config: Config) -> bytes:
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


def create_text_payload(publisher_id: int, publisher_type: str, sequence: int, config: Config) -> bytes:
    """Create a plain text payload (triggers text decoder)."""
    lines = [
        f"publisher={publisher_id} type={publisher_type} seq={sequence}",
        f"data={random_string(config.message_size_bytes)}",
    ]
    if config.include_timestamp:
        lines.append(f"timestamp={datetime.utcnow().isoformat()}Z")
    return "\n".join(lines).encode()


def create_binary_payload(size: int) -> bytes:
    """Create a binary payload with enough non-printable bytes to trigger the hex decoder."""
    # Force >10% non-printable so the text decoder rejects it
    non_printable_count = max(size // 5, 4)
    printable_count = size - non_printable_count
    data = bytearray()
    data += bytes([random.choice([0x00, 0x01, 0x02, 0x03, 0x1F, 0xFF, 0xFE]) for _ in range(non_printable_count)])
    data += bytes(random.randint(0, 255) for _ in range(printable_count))
    random.shuffle(data)
    return bytes(data)


def create_payload(
    publisher_id: int, publisher_type: str, sequence: int, config: Config, payload_type: str
) -> bytes:
    """Create a payload of the specified type (json/text/binary)."""
    if payload_type == "json":
        return create_json_payload(publisher_id, publisher_type, sequence, config)
    elif payload_type == "text":
        return create_text_payload(publisher_id, publisher_type, sequence, config)
    else:
        return create_binary_payload(config.message_size_bytes)
