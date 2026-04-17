export interface Condition {
  type: string;
  status: "True" | "False" | "Unknown" | string;
  reason?: string;
  message?: string;
  lastTransitionTime?: string;
}

export type HealthLevel = "healthy" | "unhealthy" | "progressing" | "unknown";

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

export function computeHealth(obj: unknown): Health {
  // Pod phase is authoritative for terminal states — Ready=False is expected on
  // Succeeded pods (e.g. completed Job pods), so condition-based evaluation would
  // mislabel them unhealthy.
  const phase = (obj as any)?.status?.phase;
  if (typeof phase === "string") {
    switch (phase) {
      case "Succeeded":
        return { level: "healthy", reason: "Succeeded" };
      case "Failed":
        return { level: "unhealthy", reason: "Failed" };
      case "Pending":
        return { level: "progressing", reason: "Pending" };
    }
  }

  const conditions = getConditions(obj);
  if (conditions.length === 0) return { level: "unknown", reason: "no conditions" };

  let anyNegativeTrue = false;
  let anyPositiveFalse = false;
  let anyPositiveTrue = false;
  let anyProgressingTrue = false;
  let recognized = 0;

  for (const c of conditions) {
    const isPos = POSITIVE_TYPES.has(c.type);
    const isNeg = NEGATIVE_TYPES.has(c.type);
    const isProg = PROGRESSING_TYPES.has(c.type);
    if (isPos || isNeg || isProg) recognized++;

    if (isNeg && c.status === "True") anyNegativeTrue = true;
    if (isPos && c.status === "False") anyPositiveFalse = true;
    if (isPos && c.status === "True") anyPositiveTrue = true;
    if (isProg && c.status === "True") anyProgressingTrue = true;
  }

  if (anyNegativeTrue || anyPositiveFalse) {
    return { level: "unhealthy", reason: "negative condition active" };
  }
  // A positive condition being True dominates Progressing=True — a stable
  // Deployment reports both Available=True and Progressing=True (reason:
  // NewReplicaSetAvailable), and that should read as healthy.
  if (anyPositiveTrue) {
    return { level: "healthy", reason: "positive conditions met" };
  }
  if (anyProgressingTrue) {
    return { level: "progressing", reason: "progressing" };
  }
  // No recognized condition types (e.g. Longhorn Replica exposes only custom
  // boolean flags like FilesystemReadOnly). Render nothing rather than a
  // confusing ratio.
  if (recognized === 0) return { level: "unknown", reason: "no recognized conditions" };
  return { level: "unknown", reason: "indeterminate" };
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
