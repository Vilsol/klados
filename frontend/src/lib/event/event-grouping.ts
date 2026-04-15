import type {EventItem, GroupedEvent} from "./event-types";
import {
  classifySeverity,
  eventTimestamp,
  eventFirstTimestamp,
  involvedObjectKey,
  involvedObjectOf,
} from "./event-columns";

export function groupEvents(items: EventItem[]): GroupedEvent[] {
  const groups = new Map<string, GroupedEvent>();
  for (const e of items) {
    const key = `${e.reason ?? ""}|${involvedObjectKey(e)}`;
    const ts = eventTimestamp(e);
    const fts = eventFirstTimestamp(e);
    const count = e.count ?? 1;
    const sev = classifySeverity(e);
    const existing = groups.get(key);
    if (!existing) {
      groups.set(key, {
        key,
        reason: e.reason ?? "",
        involvedObject: involvedObjectOf(e),
        severity: sev,
        count,
        firstSeen: fts,
        lastSeen: ts,
        message: e.message ?? "",
        sample: e,
      });
      continue;
    }
    existing.count += count;
    if (fts && (!existing.firstSeen || fts < existing.firstSeen)) {
      existing.firstSeen = fts;
    }
    if (ts && ts > existing.lastSeen) {
      existing.lastSeen = ts;
      existing.message = e.message ?? existing.message;
      existing.sample = e;
    }
    if (sev === "Warning") existing.severity = "Warning";
  }
  return Array.from(groups.values());
}
