import {describe, it, expect} from "vitest";
import {parseSearch, type SearchTerm} from "$lib/search/parser";

describe("parseSearch", () => {
  it("parses bare text as name filter", () => {
    const terms = parseSearch("nginx");
    expect(terms).toEqual([{type: "text", value: "nginx", negated: false}]);
  });

  it("parses label qualifier", () => {
    const terms = parseSearch("label:app=web");
    expect(terms).toEqual([{type: "label", value: "app=web", negated: false}]);
  });

  it("parses label alias l:", () => {
    const terms = parseSearch("l:app=web");
    expect(terms).toEqual([{type: "label", value: "app=web", negated: false}]);
  });

  it("parses annotation qualifier", () => {
    const terms = parseSearch("annotation:helm.sh/chart=myapp");
    expect(terms).toEqual([{type: "annotation", value: "helm.sh/chart=myapp", negated: false}]);
  });

  it("parses annotation alias ann:", () => {
    const terms = parseSearch("ann:owner=team-a");
    expect(terms).toEqual([{type: "annotation", value: "owner=team-a", negated: false}]);
  });

  it("parses name qualifier", () => {
    const terms = parseSearch("name:nginx");
    expect(terms).toEqual([{type: "name", value: "nginx", negated: false}]);
  });

  it("parses name alias n:", () => {
    const terms = parseSearch("n:nginx");
    expect(terms).toEqual([{type: "name", value: "nginx", negated: false}]);
  });

  it("parses namespace qualifier", () => {
    const terms = parseSearch("namespace:kube-system");
    expect(terms).toEqual([{type: "namespace", value: "kube-system", negated: false}]);
  });

  it("parses namespace alias ns:", () => {
    const terms = parseSearch("ns:default");
    expect(terms).toEqual([{type: "namespace", value: "default", negated: false}]);
  });

  it("parses negation", () => {
    const terms = parseSearch("-label:env=dev");
    expect(terms).toEqual([{type: "label", value: "env=dev", negated: true}]);
  });

  it("parses negated bare text", () => {
    const terms = parseSearch("-test");
    expect(terms).toEqual([{type: "text", value: "test", negated: true}]);
  });

  it("parses multiple terms", () => {
    const terms = parseSearch("l:app=web -ns:kube-system nginx");
    expect(terms).toHaveLength(3);
    expect(terms[0]).toEqual({type: "label", value: "app=web", negated: false});
    expect(terms[1]).toEqual({type: "namespace", value: "kube-system", negated: true});
    expect(terms[2]).toEqual({type: "text", value: "nginx", negated: false});
  });

  it("parses quoted phrases", () => {
    const terms = parseSearch('"crash loop"');
    expect(terms).toEqual([{type: "phrase", value: "crash loop", negated: false}]);
  });

  it("returns empty array for empty string", () => {
    const terms = parseSearch("");
    expect(terms).toEqual([]);
  });
});
