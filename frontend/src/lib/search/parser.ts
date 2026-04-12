import {parse, type TParsedTerm} from "@muhgholy/search-query-parser";

export interface SearchTerm {
  type: string;
  value: string;
  negated: boolean;
}

const KLADOS_OPTIONS = {
  operators: [
    {name: "label", aliases: ["l"], type: "string" as const, allowNegation: true},
    {name: "annotation", aliases: ["ann"], type: "string" as const, allowNegation: true},
    {name: "name", aliases: ["n"], type: "string" as const, allowNegation: true},
    {name: "namespace", aliases: ["ns"], type: "string" as const, allowNegation: true},
  ],
  operatorsAllowed: ["label", "annotation", "name", "namespace"],
};

export function parseSearch(input: string): SearchTerm[] {
  if (!input.trim()) return [];

  const parsed = parse(input, KLADOS_OPTIONS);

  return parsed.map((term: TParsedTerm) => ({
    type: term.type,
    value: term.value,
    negated: term.negated,
  }));
}
