import type {SearchTerm} from "./parser";

function matchKeyValue(map: Record<string, string> | undefined, filter: string): boolean {
  if (!map) {
    return false;
  }
  const eqIdx = filter.indexOf("=");
  if (eqIdx === -1) {
    return filter in map;
  }
  const key = filter.substring(0, eqIdx);
  const val = filter.substring(eqIdx + 1);
  return map[key] === val;
}

function matchesTerm(item: Record<string, unknown>, term: SearchTerm): boolean {
  const meta = (item.metadata ?? {}) as Record<string, unknown>;
  const name: string = ((meta.name ?? "") as string).toLowerCase();

  let matches: boolean;
  switch (term.type) {
    case "text":
    case "phrase":
      matches = name.includes(term.value.toLowerCase());
      break;
    case "name":
      matches = name.includes(term.value.toLowerCase());
      break;
    case "namespace":
      matches = ((meta.namespace ?? "") as string) === term.value;
      break;
    case "label":
      matches = matchKeyValue(meta.labels as Record<string, string> | undefined, term.value);
      break;
    case "annotation":
      matches = matchKeyValue(meta.annotations as Record<string, string> | undefined, term.value);
      break;
    default:
      matches = true;
  }

  return term.negated ? !matches : matches;
}

export function filterItems(items: Record<string, unknown>[], terms: SearchTerm[]): Record<string, unknown>[] {
  if (terms.length === 0) {
    return items;
  }
  return items.filter((item) => terms.every((term) => matchesTerm(item, term)));
}
