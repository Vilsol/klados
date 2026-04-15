import type {EventItem, EventRow, GroupedEvent, InvolvedObjectRef, Severity} from "./event-types";

export function classifySeverity(e: EventItem | GroupedEvent): Severity {
  if ("severity" in e && e.severity) return e.severity;
  return (e as EventItem).type === "Warning" ? "Warning" : "Normal";
}

export function eventTimestamp(e: EventItem): string {
  return e.lastTimestamp ?? e.eventTime ?? e.metadata?.creationTimestamp ?? "";
}

export function eventFirstTimestamp(e: EventItem): string {
  return e.firstTimestamp ?? eventTimestamp(e);
}

export function involvedObjectOf(e: EventItem): InvolvedObjectRef {
  const io = e.involvedObject ?? {};
  return {
    kind: io.kind ?? "",
    apiVersion: io.apiVersion ?? "",
    name: io.name ?? "",
    namespace: io.namespace ?? e.metadata?.namespace ?? "",
    uid: io.uid ?? "",
  };
}

export function involvedObjectKey(e: EventItem): string {
  const io = involvedObjectOf(e);
  return io.uid ? io.uid : `${io.namespace}/${io.kind}/${io.name}`;
}

export function formatObject(io: InvolvedObjectRef): string {
  if (!io.kind && !io.name) return "";
  return `${io.kind}/${io.name}`;
}

export function rowReason(row: EventRow): string {
  return (row as EventItem).reason ?? (row as GroupedEvent).reason ?? "";
}

export function rowMessage(row: EventRow): string {
  return (row as EventItem).message ?? (row as GroupedEvent).message ?? "";
}

export function rowCount(row: EventRow): number {
  if ("count" in row && typeof row.count === "number") return row.count;
  return 1;
}

export function rowLastSeen(row: EventRow): string {
  if ("lastSeen" in row && row.lastSeen) return row.lastSeen;
  return eventTimestamp(row as EventItem);
}

export function rowFirstSeen(row: EventRow): string {
  if ("firstSeen" in row && row.firstSeen) return row.firstSeen;
  return eventFirstTimestamp(row as EventItem);
}

export function rowInvolvedObject(row: EventRow): InvolvedObjectRef {
  if ("involvedObject" in row && (row as GroupedEvent).involvedObject) {
    return (row as GroupedEvent).involvedObject;
  }
  return involvedObjectOf(row as EventItem);
}

export function rowSample(row: EventRow): EventItem {
  return (row as GroupedEvent).sample ?? (row as EventItem);
}
