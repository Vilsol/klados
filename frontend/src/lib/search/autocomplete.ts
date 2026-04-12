export interface Suggestion {
  value: string;
  count?: number;
  description?: string;
}

const QUALIFIERS = [
  {value: "label:", aliases: ["l:"], description: "Filter by label"},
  {value: "annotation:", aliases: ["ann:"], description: "Filter by annotation"},
  {value: "name:", aliases: ["n:"], description: "Filter by name"},
  {value: "namespace:", aliases: ["ns:"], description: "Filter by namespace"},
];

const QUALIFIER_ALIASES: Record<string, string> = {
  "l:": "label:",
  "ann:": "annotation:",
  "n:": "name:",
  "ns:": "namespace:",
};

function extractCurrentToken(input: string, cursor: number): string {
  const before = input.substring(0, cursor);
  const lastSpace = before.lastIndexOf(" ");
  return before.substring(lastSpace + 1);
}

function collectDistinct(
  items: Record<string, unknown>[],
  extractor: (item: Record<string, unknown>) => Record<string, string> | undefined,
): Map<string, number> {
  const counts = new Map<string, number>();
  for (const item of items) {
    const map = extractor(item);
    if (map) {
      for (const key of Object.keys(map)) {
        counts.set(key, (counts.get(key) ?? 0) + 1);
      }
    }
  }
  return counts;
}

function collectValues(
  items: Record<string, unknown>[],
  extractor: (item: Record<string, unknown>) => Record<string, string> | undefined,
  key: string,
): Map<string, number> {
  const counts = new Map<string, number>();
  for (const item of items) {
    const map = extractor(item);
    if (map && key in map) {
      const val = map[key];
      counts.set(val, (counts.get(val) ?? 0) + 1);
    }
  }
  return counts;
}

function collectNamespaces(items: Record<string, unknown>[]): Map<string, number> {
  const counts = new Map<string, number>();
  for (const item of items) {
    const ns = ((item.metadata as Record<string, unknown> | undefined)?.namespace ?? "") as string;
    if (ns) {
      counts.set(ns, (counts.get(ns) ?? 0) + 1);
    }
  }
  return counts;
}

function mapToSuggestions(counts: Map<string, number>, prefix: string): Suggestion[] {
  return Array.from(counts.entries())
    .filter(([key]) => !prefix || key.toLowerCase().startsWith(prefix.toLowerCase()))
    .map(([value, count]) => ({value, count}))
    .sort((a, b) => (b.count ?? 0) - (a.count ?? 0));
}

export function getSuggestions(input: string, cursor: number, items: Record<string, unknown>[]): Suggestion[] {
  const token = extractCurrentToken(input, cursor);
  const stripped = token.startsWith("-") ? token.substring(1) : token;
  const colonIdx = stripped.indexOf(":");

  if (colonIdx === -1) {
    if (stripped === "") {
      return QUALIFIERS.map((q) => ({value: q.value, description: q.description}));
    }
    const lower = stripped.toLowerCase();
    const matches = QUALIFIERS.filter((q) => q.value.startsWith(lower) || q.aliases.some((a) => a.startsWith(lower)));
    if (matches.length > 0) {
      return matches.map((q) => ({value: q.value, description: q.description}));
    }
    return [];
  }

  let qualifier = stripped.substring(0, colonIdx + 1);
  qualifier = QUALIFIER_ALIASES[qualifier] ?? qualifier;
  const afterColon = stripped.substring(colonIdx + 1);
  const eqIdx = afterColon.indexOf("=");

  if (qualifier === "label:") {
    const extractor = (item: Record<string, unknown>) =>
      (item.metadata as Record<string, unknown> | undefined)?.labels as Record<string, string> | undefined;
    if (eqIdx === -1) {
      return mapToSuggestions(collectDistinct(items, extractor), afterColon);
    }
    const key = afterColon.substring(0, eqIdx);
    const valPrefix = afterColon.substring(eqIdx + 1);
    return mapToSuggestions(collectValues(items, extractor, key), valPrefix);
  }

  if (qualifier === "annotation:") {
    const extractor = (item: Record<string, unknown>) =>
      (item.metadata as Record<string, unknown> | undefined)?.annotations as Record<string, string> | undefined;
    if (eqIdx === -1) {
      return mapToSuggestions(collectDistinct(items, extractor), afterColon);
    }
    const key = afterColon.substring(0, eqIdx);
    const valPrefix = afterColon.substring(eqIdx + 1);
    return mapToSuggestions(collectValues(items, extractor, key), valPrefix);
  }

  if (qualifier === "namespace:") {
    return mapToSuggestions(collectNamespaces(items), afterColon);
  }

  return [];
}
