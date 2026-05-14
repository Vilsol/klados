import {stringify} from "yaml";

export function itemToYaml(item: Record<string, unknown>): string {
  const clone = structuredClone(item) as Record<string, unknown>;
  const meta = clone.metadata as Record<string, unknown> | undefined;
  if (meta && "managedFields" in meta) {
    delete meta.managedFields;
  }
  return stringify(clone);
}
