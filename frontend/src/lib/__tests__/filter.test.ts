import {describe, it, expect} from "vitest";
import {filterItems} from "$lib/search/filter";
import type {SearchTerm} from "$lib/search/parser";

function makeItem(name: string, namespace: string, labels: Record<string, string> = {}, annotations: Record<string, string> = {}) {
  return {
    metadata: {name, namespace, labels, annotations},
  };
}

const items = [
  makeItem("nginx-proxy", "default", {app: "web", env: "prod"}, {owner: "team-a"}),
  makeItem("nginx-ingress", "kube-system", {app: "web", env: "dev"}, {owner: "team-b"}),
  makeItem("redis-master", "default", {app: "cache", env: "prod"}, {}),
  makeItem("test-pod", "testing", {app: "test"}, {"helm.sh/chart": "myapp"}),
];

describe("filterItems", () => {
  it("returns all items when no terms", () => {
    expect(filterItems(items, [])).toHaveLength(4);
  });

  it("filters by bare text on name", () => {
    const terms: SearchTerm[] = [{type: "text", value: "nginx", negated: false}];
    const result = filterItems(items, terms);
    expect(result).toHaveLength(2);
    expect(result.map((r: any) => r.metadata.name)).toEqual(["nginx-proxy", "nginx-ingress"]);
  });

  it("filters by phrase on name", () => {
    const terms: SearchTerm[] = [{type: "phrase", value: "redis-master", negated: false}];
    const result = filterItems(items, terms);
    expect(result).toHaveLength(1);
  });

  it("filters by name qualifier", () => {
    const terms: SearchTerm[] = [{type: "name", value: "proxy", negated: false}];
    const result = filterItems(items, terms);
    expect(result).toHaveLength(1);
    expect(result[0].metadata.name).toBe("nginx-proxy");
  });

  it("filters by namespace qualifier", () => {
    const terms: SearchTerm[] = [{type: "namespace", value: "default", negated: false}];
    const result = filterItems(items, terms);
    expect(result).toHaveLength(2);
  });

  it("filters by label key=value", () => {
    const terms: SearchTerm[] = [{type: "label", value: "app=web", negated: false}];
    const result = filterItems(items, terms);
    expect(result).toHaveLength(2);
  });

  it("filters by label key exists", () => {
    const terms: SearchTerm[] = [{type: "label", value: "env", negated: false}];
    const result = filterItems(items, terms);
    expect(result).toHaveLength(3);
  });

  it("filters by annotation key=value", () => {
    const terms: SearchTerm[] = [{type: "annotation", value: "owner=team-a", negated: false}];
    const result = filterItems(items, terms);
    expect(result).toHaveLength(1);
    expect(result[0].metadata.name).toBe("nginx-proxy");
  });

  it("filters by annotation key exists", () => {
    const terms: SearchTerm[] = [{type: "annotation", value: "owner", negated: false}];
    const result = filterItems(items, terms);
    expect(result).toHaveLength(2);
  });

  it("negates text filter", () => {
    const terms: SearchTerm[] = [{type: "text", value: "nginx", negated: true}];
    const result = filterItems(items, terms);
    expect(result).toHaveLength(2);
    expect(result.map((r: any) => r.metadata.name)).toEqual(["redis-master", "test-pod"]);
  });

  it("negates label filter", () => {
    const terms: SearchTerm[] = [{type: "label", value: "env=dev", negated: true}];
    const result = filterItems(items, terms);
    expect(result).toHaveLength(3);
  });

  it("negates namespace filter", () => {
    const terms: SearchTerm[] = [{type: "namespace", value: "kube-system", negated: true}];
    const result = filterItems(items, terms);
    expect(result).toHaveLength(3);
  });

  it("ANDs multiple terms", () => {
    const terms: SearchTerm[] = [
      {type: "label", value: "app=web", negated: false},
      {type: "namespace", value: "default", negated: false},
    ];
    const result = filterItems(items, terms);
    expect(result).toHaveLength(1);
    expect(result[0].metadata.name).toBe("nginx-proxy");
  });

  it("handles annotation with dots and slashes in key", () => {
    const terms: SearchTerm[] = [{type: "annotation", value: "helm.sh/chart=myapp", negated: false}];
    const result = filterItems(items, terms);
    expect(result).toHaveLength(1);
    expect(result[0].metadata.name).toBe("test-pod");
  });
});
