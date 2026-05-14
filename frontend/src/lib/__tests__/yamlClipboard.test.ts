import {describe, it, expect} from "vitest";
import {itemToYaml} from "$lib/utils/yamlClipboard";

describe("itemToYaml", () => {
  it("serializes a Kubernetes resource to YAML", () => {
    const pod = {
      apiVersion: "v1",
      kind: "Pod",
      metadata: {name: "test-pod", namespace: "default"},
      spec: {containers: [{name: "main", image: "nginx"}]},
    };
    const yaml = itemToYaml(pod);
    expect(yaml).toContain("apiVersion: v1");
    expect(yaml).toContain("kind: Pod");
    expect(yaml).toContain("name: test-pod");
    expect(yaml).toContain("image: nginx");
  });

  it("strips managedFields but keeps status", () => {
    const obj = {
      apiVersion: "v1",
      kind: "Pod",
      metadata: {name: "p", managedFields: [{manager: "k"}]},
      status: {phase: "Running"},
    };
    const yaml = itemToYaml(obj);
    expect(yaml).not.toContain("managedFields");
    expect(yaml).toContain("phase: Running");
  });
});
