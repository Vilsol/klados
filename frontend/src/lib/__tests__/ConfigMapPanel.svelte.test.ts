import {describe, it, expect} from "vitest";
import {render, screen} from "@testing-library/svelte";
import ConfigMapPanel from "$lib/components/panels/ConfigMapPanel.svelte";

const obj = {
  metadata: {name: "my-config", namespace: "default"},
  data: {
    "app.yaml": "server:\n  port: 8080",
    "db.json": '{"host":"localhost"}',
    "plain.txt": "hello world",
  },
};

const KEY_COUNT_REGEX = /3 keys/;

describe("ConfigMapPanel", () => {
  it("renders all data keys", () => {
    render(ConfigMapPanel, {props: {obj}});
    expect(screen.getByText("app.yaml")).toBeTruthy();
    expect(screen.getByText("db.json")).toBeTruthy();
    expect(screen.getByText("plain.txt")).toBeTruthy();
  });

  it("renders data values", () => {
    render(ConfigMapPanel, {props: {obj}});
    expect(screen.getByText("hello world")).toBeTruthy();
  });

  it("shows key count", () => {
    render(ConfigMapPanel, {props: {obj}});
    expect(screen.getByText(KEY_COUNT_REGEX)).toBeTruthy();
  });

  it("detects json language", () => {
    render(ConfigMapPanel, {props: {obj}});
    expect(screen.getAllByText("json").length).toBeGreaterThan(0);
  });

  it("detects yaml language", () => {
    render(ConfigMapPanel, {props: {obj}});
    expect(screen.getAllByText("yaml").length).toBeGreaterThan(0);
  });

  it("renders empty state for no data", () => {
    render(ConfigMapPanel, {props: {obj: {metadata: {}, data: {}}}});
    expect(screen.getByText("No data")).toBeTruthy();
  });

  it("shows binary data key names", () => {
    const withBinary = {
      metadata: {},
      data: {},
      binaryData: {"icon.png": "base64data"},
    };
    render(ConfigMapPanel, {props: {obj: withBinary}});
    expect(screen.getByText("icon.png")).toBeTruthy();
    expect(screen.getByText("(binary)")).toBeTruthy();
  });
});
