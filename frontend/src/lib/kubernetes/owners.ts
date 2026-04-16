export interface OwnerReference {
  apiVersion: string;
  kind: string;
  name: string;
  uid: string;
  controller?: boolean;
  blockOwnerDeletion?: boolean;
}

export function getOwnerReferences(obj: unknown): OwnerReference[] {
  if (!obj || typeof obj !== "object") return [];
  const m = (obj as any).metadata;
  const refs = m?.ownerReferences;
  if (!Array.isArray(refs)) return [];
  return refs.filter((r) => r && r.kind && r.name && r.uid && r.apiVersion);
}

/**
 * Convert an ownerReference's (apiVersion, kind) to our dot-separated GVR
 * string. Best-effort naive pluralization; for GVRs whose plural doesn't
 * follow standard rules (e.g. "endpoints"), the caller should fall back to
 * discovery metadata lookup when possible.
 */
export function gvrFromAPIVersion(apiVersion: string, kind: string): string {
  const slash = apiVersion.indexOf("/");
  const group = slash >= 0 ? apiVersion.slice(0, slash) : "core";
  const version = slash >= 0 ? apiVersion.slice(slash + 1) : apiVersion;
  return `${group}.${version}.${pluralize(kind.toLowerCase())}`;
}

function pluralize(s: string): string {
  if (s.endsWith("s")) return s;
  if (s.endsWith("y") && !/[aeiou]y$/.test(s)) return s.slice(0, -1) + "ies";
  if (s.endsWith("x") || s.endsWith("ch") || s.endsWith("sh")) return s + "es";
  return s + "s";
}
