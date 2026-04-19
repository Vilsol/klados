import {describe, it, expect, vi, beforeEach} from "vitest";
import {render, screen, fireEvent} from "@testing-library/svelte";

// @klados/ui barrel re-exports YAMLEditor, which imports codemirror-json-schema
// internals that vitest can't resolve against the installed dist. Mock those
// transitive imports so the component under test can load.
vi.mock("codemirror-json-schema", () => ({
  stateExtensions: () => [],
  handleRefresh: () => [],
}));
vi.mock("codemirror-json-schema/yaml", () => ({
  yamlSchemaLinter: () => [],
  yamlSchemaHover: () => [],
  yamlSchema: () => [],
}));
vi.mock("codemirror-yaml-completion", () => ({
  yamlSchemaCompletion: () => [],
}));

import VolumeBrowserSettings from "../VolumeBrowserSettings.svelte";
import type {VolumeBrowserConfig} from "$lib/stores/preferences.svelte";

function defaults(): VolumeBrowserConfig {
  return {
    image: "alpine:edge",
    mountPath: "/mnt/volume",
    orphanCleanupOnStartup: "prompt",
  };
}

describe("VolumeBrowserSettings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders initial values for image and mount path", () => {
    render(VolumeBrowserSettings, {props: {value: defaults()}});
    const image = screen.getByPlaceholderText("alpine:edge") as HTMLInputElement;
    expect(image.value).toBe("alpine:edge");
    const mount = screen.getByPlaceholderText("/mnt/volume") as HTMLInputElement;
    expect(mount.value).toBe("/mnt/volume");
  });

  it("flags invalid mount path (missing leading slash)", async () => {
    const onchange = vi.fn();
    render(VolumeBrowserSettings, {props: {value: defaults(), onchange}});
    const mount = screen.getByPlaceholderText("/mnt/volume") as HTMLInputElement;
    await fireEvent.input(mount, {target: {value: "not-a-path"}});
    expect(screen.getByText(/Mount path must start with/i)).toBeTruthy();
  });

  it("accepts a valid mount path without error", async () => {
    render(VolumeBrowserSettings, {props: {value: defaults()}});
    const mount = screen.getByPlaceholderText("/mnt/volume") as HTMLInputElement;
    await fireEvent.input(mount, {target: {value: "/data"}});
    expect(screen.queryByText(/Mount path must start with/i)).toBeNull();
  });

  it("deadline toggle off yields activeDeadlineSeconds nil; toggle on sets number", async () => {
    const onchange = vi.fn();
    const initial: VolumeBrowserConfig = {...defaults()};
    render(VolumeBrowserSettings, {props: {value: initial, onchange}});

    // Initially no activeDeadlineSeconds → toggle unchecked, no number input
    expect(screen.queryByDisplayValue("3600")).toBeNull();

    const toggle = screen.getByLabelText(/Kill after N seconds/i) as HTMLInputElement;
    await fireEvent.click(toggle);

    const numberInput = screen.getByDisplayValue("3600") as HTMLInputElement;
    expect(numberInput).toBeTruthy();

    const calls = onchange.mock.calls;
    const last = calls[calls.length - 1][0] as VolumeBrowserConfig;
    expect(last.activeDeadlineSeconds).toBe(3600);

    await fireEvent.click(toggle);
    const after = (onchange.mock.calls[onchange.mock.calls.length - 1][0]) as VolumeBrowserConfig;
    expect(after.activeDeadlineSeconds).toBeUndefined();
  });

  it("resources toggle produces nil when off, populated object when on", async () => {
    const onchange = vi.fn();
    render(VolumeBrowserSettings, {props: {value: defaults(), onchange}});

    const toggle = screen.getByLabelText(/Set container resources/i) as HTMLInputElement;
    expect(toggle.checked).toBe(false);

    await fireEvent.click(toggle);
    const cpu = screen.getByPlaceholderText("10m") as HTMLInputElement;
    await fireEvent.input(cpu, {target: {value: "50m"}});

    const last = onchange.mock.calls[onchange.mock.calls.length - 1][0] as VolumeBrowserConfig;
    expect(last.resources).toBeTruthy();
    expect(last.resources?.requests?.cpu).toBe("50m");

    await fireEvent.click(toggle);
    const after = onchange.mock.calls[onchange.mock.calls.length - 1][0] as VolumeBrowserConfig;
    expect(after.resources).toBeUndefined();
  });

  it("invalid tolerations JSON shows error and does not clobber last-known-good value", async () => {
    const onchange = vi.fn();
    const initial: VolumeBrowserConfig = {
      ...defaults(),
      tolerations: [{key: "dedicated", operator: "Equal", value: "gpu", effect: "NoSchedule"}],
    };
    render(VolumeBrowserSettings, {props: {value: initial, onchange}});

    const textarea = screen.getByPlaceholderText(/dedicated/i) as HTMLTextAreaElement;
    expect(textarea.value).toContain("dedicated");

    await fireEvent.input(textarea, {target: {value: "{not valid json"}});
    await fireEvent.blur(textarea);

    expect(screen.getByText(/Invalid JSON/i)).toBeTruthy();

    const lastCall = onchange.mock.calls[onchange.mock.calls.length - 1];
    if (lastCall) {
      const last = lastCall[0] as VolumeBrowserConfig;
      expect(last.tolerations).toEqual(initial.tolerations);
    }
  });
});
