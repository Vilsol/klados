import {describe, it, expect} from "vitest";
import {bucketize, pickBucketSize, BUCKET_SIZES_MS} from "../event-timeline";

describe("pickBucketSize", () => {
  it("returns a bucket size that fits the range/target ratio", () => {
    expect(pickBucketSize(60 * 60 * 1000, 60)).toBe(60_000);
    expect(pickBucketSize(24 * 60 * 60 * 1000, 60)).toBe(3_600_000);
    expect(pickBucketSize(10 * 60 * 1000, 60)).toBe(15_000);
  });
  it("clamps to the max for very long ranges", () => {
    expect(pickBucketSize(Number.MAX_SAFE_INTEGER, 60)).toBe(BUCKET_SIZES_MS.at(-1));
  });
});

describe("bucketize", () => {
  it("counts warnings and normals into correct buckets", () => {
    const from = Date.parse("2026-04-15T10:00:00Z");
    const to = from + 5 * 60_000;
    const items = [
      {type: "Warning", lastTimestamp: "2026-04-15T10:00:30Z", involvedObject: {}},
      {type: "Normal",  lastTimestamp: "2026-04-15T10:02:15Z", involvedObject: {}},
      {type: "Warning", lastTimestamp: "2026-04-15T10:04:59Z", involvedObject: {}},
    ];
    const result = bucketize(items as any, from, to, 60_000);
    expect(result).toHaveLength(5);
    expect(result[0].warn).toBe(1);
    expect(result[2].normal).toBe(1);
    expect(result[4].warn).toBe(1);
  });
  it("drops events outside the range", () => {
    const from = Date.parse("2026-04-15T10:00:00Z");
    const to = from + 60_000;
    const items = [{type: "Warning", lastTimestamp: "2026-04-15T09:00:00Z", involvedObject: {}}];
    expect(bucketize(items as any, from, to, 60_000)[0].warn).toBe(0);
  });
});
