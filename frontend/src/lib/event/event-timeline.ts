import type {EventItem} from "./event-types";
import {classifySeverity, eventTimestamp} from "./event-columns";

export interface TimelineBucket {
  t0: number; // bucket start (ms)
  t1: number; // bucket end (ms)
  warn: number;
  normal: number;
}

export const BUCKET_SIZES_MS = [
  15_000,
  30_000,
  60_000,
  300_000,
  900_000,
  3_600_000,
];

export function pickBucketSize(rangeMs: number, targetBuckets = 60): number {
  const ideal = rangeMs / targetBuckets;
  for (const size of BUCKET_SIZES_MS) {
    if (size >= ideal) return size;
  }
  return BUCKET_SIZES_MS[BUCKET_SIZES_MS.length - 1];
}

export function bucketize(
  items: EventItem[],
  fromMs: number,
  toMs: number,
  bucketSizeMs: number,
): TimelineBucket[] {
  const buckets: TimelineBucket[] = [];
  for (let t = fromMs; t < toMs; t += bucketSizeMs) {
    buckets.push({t0: t, t1: Math.min(t + bucketSizeMs, toMs), warn: 0, normal: 0});
  }
  for (const e of items) {
    const ts = Date.parse(eventTimestamp(e));
    if (!Number.isFinite(ts) || ts < fromMs || ts >= toMs) continue;
    const idx = Math.min(buckets.length - 1, Math.floor((ts - fromMs) / bucketSizeMs));
    if (classifySeverity(e) === "Warning") buckets[idx].warn++;
    else buckets[idx].normal++;
  }
  return buckets;
}
