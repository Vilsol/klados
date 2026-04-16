export const LAST_APPLIED_ANNOTATION = "kubectl.kubernetes.io/last-applied-configuration";

const SERVER_META_FIELDS = [
  "resourceVersion",
  "uid",
  "creationTimestamp",
  "generation",
  "selfLink",
  "managedFields",
];

function getMetadata(obj: unknown): Record<string, any> {
  if (!obj || typeof obj !== "object") return {};
  const m = (obj as any).metadata;
  return m && typeof m === "object" ? m : {};
}

export function getLabels(obj: unknown): Record<string, string> {
  const l = getMetadata(obj).labels;
  return l && typeof l === "object" ? { ...l } : {};
}

export function getAnnotations(obj: unknown): Record<string, string> {
  const a = getMetadata(obj).annotations;
  return a && typeof a === "object" ? { ...a } : {};
}

export function getFinalizers(obj: unknown): string[] {
  const f = getMetadata(obj).finalizers;
  return Array.isArray(f) ? [...f] : [];
}

export function stripServerFields(obj: Record<string, any>): Record<string, any> {
  const copy: Record<string, any> = JSON.parse(JSON.stringify(obj ?? {}));
  if (copy.metadata && typeof copy.metadata === "object") {
    for (const f of SERVER_META_FIELDS) delete copy.metadata[f];
  }
  delete copy.status;
  return copy;
}

export function getLastAppliedConfig(obj: unknown): Record<string, any> | null {
  const ann = getAnnotations(obj);
  const raw = ann[LAST_APPLIED_ANNOTATION];
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
}
