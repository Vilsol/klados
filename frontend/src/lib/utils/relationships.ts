export interface ControllerRef {
  apiVersion: string;
  kind: string;
  name: string;
  uid: string;
}

export interface APIResource {
  gvr: string;
  kind: string;
  namespaced: boolean;
}

export function getControllerRef(obj: any): ControllerRef | null {
  const refs = obj?.metadata?.ownerReferences;
  if (!Array.isArray(refs)) return null;
  const controller = refs.find((r: any) => r.controller === true);
  if (!controller) return null;
  return {
    apiVersion: controller.apiVersion,
    kind: controller.kind,
    name: controller.name,
    uid: controller.uid,
  };
}

export function gvrToApiVersion(gvr: string): string {
  const parts = gvr.split(".");
  const version = parts.at(-2) ?? "";
  const group = parts.slice(0, -2).join(".");
  if (group === "core" || group === "") return version;
  return `${group}/${version}`;
}

export function buildKindGVRMap(resources: APIResource[]): Map<string, string> {
  const map = new Map<string, string>();
  for (const r of resources) {
    const apiVersion = gvrToApiVersion(r.gvr);
    map.set(`${apiVersion}:${r.kind}`, r.gvr);
  }
  return map;
}

export function resolveGVR(map: Map<string, string>, apiVersion: string, kind: string): string | undefined {
  return map.get(`${apiVersion}:${kind}`);
}
