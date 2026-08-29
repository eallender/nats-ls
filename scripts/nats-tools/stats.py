# SPDX-License-Identifier: Apache-2.0
# Copyright Evan Allender

import asyncio
import time
from dataclasses import dataclass, field

from config import Config


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
        return self.normal_sent + self.js_sent + self.reqrep_sent + self.kv_sent + self.obj_sent

    def total_errors(self) -> int:
        return self.normal_errors + self.js_errors + self.reqrep_errors + self.kv_errors + self.obj_errors

    def rate(self) -> float:
        elapsed = time.time() - self.start_time
        if elapsed > 0:
            return self.total_sent() / elapsed
        return 0.0


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
