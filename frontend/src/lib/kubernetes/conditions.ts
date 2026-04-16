export interface Condition {
  type: string;
  status: "True" | "False" | "Unknown" | string;
  reason?: string;
  message?: string;
  lastTransitionTime?: string;
}

export type HealthLevel = "healthy" | "unhealthy" | "progressing" | "mixed" | "unknown";

export interface Health {
  level: HealthLevel;
  reason: string;
}

const POSITIVE_TYPES = new Set([
  "Ready",
  "Available",
  "Initialized",
  "ContainersReady",
  "PodScheduled",
  "Succeeded",
  "Complete",
  "Synced",
  "Established",
  "NamesAccepted",
]);

const NEGATIVE_TYPES = new Set([
  "Degraded",
  "MemoryPressure",
  "DiskPressure",
  "PIDPressure",
  "NetworkUnavailable",
  "Failed",
  "ReplicaFailure",
  "Stalled",
]);

const PROGRESSING_TYPES = new Set(["Progressing", "Reconciling"]);

export function getConditions(obj: unknown): Condition[] {
  if (!obj || typeof obj !== "object") return [];
  const status = (obj as any).status;
  const arr = status?.conditions;
  if (!Array.isArray(arr)) return [];
  const out: Condition[] = [];
  for (const c of arr) {
    if (!c || typeof c !== "object") continue;
    if (typeof c.type !== "string" || typeof c.status !== "string") continue;
    out.push({
      type: c.type,
      status: c.status,
      reason: c.reason,
      message: c.message,
      lastTransitionTime: c.lastTransitionTime,
    });
  }
  return out;
}

export function computeHealth(conditions: Condition[]): Health {
  if (conditions.length === 0) return { level: "unknown", reason: "no conditions" };

  let anyNegativeTrue = false;
  let anyPositiveFalse = false;
  let anyPositiveTrue = false;
  let anyProgressingTrue = false;

  for (const c of conditions) {
    const isPos = POSITIVE_TYPES.has(c.type);
    const isNeg = NEGATIVE_TYPES.has(c.type);
    const isProg = PROGRESSING_TYPES.has(c.type);

    if (isNeg && c.status === "True") anyNegativeTrue = true;
    if (isPos && c.status === "False") anyPositiveFalse = true;
    if (isPos && c.status === "True") anyPositiveTrue = true;
    if (isProg && c.status === "True") anyProgressingTrue = true;
  }

  if (anyNegativeTrue || anyPositiveFalse) {
    return { level: "unhealthy", reason: "negative condition active" };
  }
  if (anyPositiveTrue && !anyProgressingTrue) {
    return { level: "healthy", reason: "positive conditions met" };
  }
  if (anyProgressingTrue) {
    return { level: "progressing", reason: "progressing" };
  }
  const total = conditions.length;
  const trues = conditions.filter((c) => c.status === "True").length;
  return { level: "mixed", reason: `${trues}/${total} True` };
}

export interface Warning {
  type: string;
  reason: string;
  message: string;
}

export function findWarnings(conditions: Condition[]): Warning[] {
  const warns: Warning[] = [];
  for (const c of conditions) {
    if (POSITIVE_TYPES.has(c.type) && c.status === "False") {
      warns.push({ type: c.type, reason: c.reason ?? "", message: c.message ?? "" });
    } else if (NEGATIVE_TYPES.has(c.type) && c.status === "True") {
      warns.push({ type: c.type, reason: c.reason ?? "", message: c.message ?? "" });
    }
  }
  return warns;
}
