import type { APIResource } from "../../../bindings/github.com/Vilsol/klados/internal/cluster/index.js";
import {
  Descriptor,
  Column,
  Action,
  OverviewField,
} from "../../../bindings/github.com/Vilsol/klados/internal/resource/index.js";

export type RenderType = "text" | "badge" | "age" | "progress";

const UNIVERSAL_PANELS = [
  "overview",
  "yaml",
  "events",
  "conditions",
  "metadata",
  "related",
  "drift",
];

/**
 * Convert a JSONPath expression (CRD additionalPrinterColumns) to a CEL
 * expression. Only simple dotted paths are supported — filter predicates
 * return "" so callers can skip those columns.
 */
export function jsonPathToCEL(jsonPath: string): string {
  if (!jsonPath) return "";
  let p = jsonPath.startsWith("$.") ? jsonPath.slice(2) : jsonPath;
  if (p.startsWith(".")) p = p.slice(1);
  if (p.includes("[") || p.includes("?") || p.includes("@")) return "";
  return p;
}

export function crdTypeToRenderType(t: string): RenderType {
  switch (t) {
    case "date":
      return "age";
    case "boolean":
      return "badge";
    default:
      return "text";
  }
}

/** Build a Descriptor from an enriched APIResource (discovery payload). */
export function generateDescriptor(r: APIResource): Descriptor {
  const [group, version, resourceName] = splitGVR(r.gvr);

  const columns: Column[] = [];
  columns.push(new Column({ name: "Name", expr: "metadata.name", renderType: "text" }));
  if (r.namespaced) {
    columns.push(new Column({ name: "Namespace", expr: "metadata.namespace", renderType: "text" }));
  }
  columns.push(new Column({ name: "Age", expr: "metadata.creationTimestamp", renderType: "age" }));

  const existingNames = new Set(columns.map((c) => c.name));
  for (const pc of r.printerColumns ?? []) {
    const expr = jsonPathToCEL(pc.jsonPath);
    if (!expr) continue;
    if (existingNames.has(pc.name)) continue;
    columns.push(new Column({
      name: pc.name,
      expr,
      renderType: crdTypeToRenderType(pc.type),
      hidden: (pc.priority ?? 0) > 0,
    }));
    existingNames.add(pc.name);
  }

  if (r.subresources?.scale) {
    const specPath = jsonPathToCEL(r.scaleSpec?.specReplicasPath ?? ".spec.replicas");
    if (!existingNames.has("Replicas") && specPath) {
      columns.push(new Column({ name: "Replicas", expr: specPath, renderType: "text" }));
      existingNames.add("Replicas");
    }
  }

  const panels = [...UNIVERSAL_PANELS];
  if (r.subresources?.status) panels.push("status");

  const actions: Action[] = [
    new Action({ name: "edit-yaml", label: "Edit YAML" }),
    new Action({ name: "delete", label: "Delete" }),
  ];
  if (r.subresources?.scale) {
    actions.unshift(new Action({ name: "scale", label: "Scale" }));
  }

  const overviewFields: OverviewField[] = [];
  if (r.namespaced) {
    overviewFields.push(new OverviewField({ label: "Namespace", expr: "metadata.namespace", renderType: "text" }));
  }
  overviewFields.push(new OverviewField({ label: "Age", expr: "metadata.creationTimestamp", renderType: "age" }));

  return new Descriptor({
    group,
    version,
    resource: resourceName,
    kind: r.kind,
    columns,
    overviewFields,
    detailPanels: panels,
    actions,
    clusterScoped: !r.namespaced,
  });
}

function splitGVR(gvr: string): [string, string, string] {
  const lastDot = gvr.lastIndexOf(".");
  const secondLast = gvr.lastIndexOf(".", lastDot - 1);
  if (secondLast === -1) return ["", "", gvr];
  const group = gvr.slice(0, secondLast);
  const version = gvr.slice(secondLast + 1, lastDot);
  const resourceName = gvr.slice(lastDot + 1);
  return [group, version, resourceName];
}
