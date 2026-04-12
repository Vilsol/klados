import type {SavedFilter} from "$lib/stores/preferences.svelte";
import {parseSearch} from "./parser";

export function savedFilterToQuery(filter: SavedFilter): string {
  const parts: string[] = [];

  if (filter.labels) {
    for (const [key, value] of Object.entries(filter.labels)) {
      parts.push(`l:${key}=${value}`);
    }
  }

  if (filter.annotations) {
    for (const [key, value] of Object.entries(filter.annotations)) {
      parts.push(`ann:${key}=${value}`);
    }
  }

  if (filter.search) {
    parts.push(filter.search);
  }

  return parts.join(" ");
}

export function queryToSavedFilter(query: string): Omit<SavedFilter, "name"> {
  if (!query.trim()) return {};

  const terms = parseSearch(query);
  const labels: Record<string, string> = {};
  const annotations: Record<string, string> = {};
  const searchParts: string[] = [];

  for (const term of terms) {
    if (term.type === "label" && term.value.includes("=") && !term.negated) {
      const [key, ...rest] = term.value.split("=");
      labels[key] = rest.join("=");
    } else if (term.type === "annotation" && term.value.includes("=") && !term.negated) {
      const [key, ...rest] = term.value.split("=");
      annotations[key] = rest.join("=");
    } else if (term.type === "text" || term.type === "phrase") {
      searchParts.push(term.negated ? `-${term.value}` : term.value);
    } else {
      // name:, namespace:, negated labels/annotations — preserve as search text
      const prefix = term.negated ? "-" : "";
      searchParts.push(`${prefix}${term.type}:${term.value}`);
    }
  }

  const result: Omit<SavedFilter, "name"> = {};
  if (Object.keys(labels).length > 0) result.labels = labels;
  if (Object.keys(annotations).length > 0) result.annotations = annotations;
  if (searchParts.length > 0) result.search = searchParts.join(" ");
  return result;
}
