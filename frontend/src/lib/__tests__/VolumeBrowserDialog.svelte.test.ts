import {describe, it, expect, vi, beforeEach} from "vitest";
import {render, screen, fireEvent} from "@testing-library/svelte";
import {tick} from "svelte";

// @klados/ui barrel may pull codemirror; mock the usual suspects
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

import VolumeBrowserDialog from "$lib/components/VolumeBrowserDialog.svelte";
import type {VolumeBrowserConfig} from "$lib/stores/preferences.svelte";

function defaults(): VolumeBrowserConfig {
  return {
    image: "alpine:edge",
    mountPath: "/mnt/volume",
    orphanCleanupOnStartup: "prompt",
  };
}

describe("VolumeBrowserDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders pre-filled image and mount path", () => {
    render(VolumeBrowserDialog, {
      props: {
        open: true,
        namespace: "ns1",
        pvcName: "data-pvc",
        initial: defaults(),
        onsubmit: vi.fn(),
        oncancel: vi.fn(),
      },
    });
    const image = screen.getByPlaceholderText("alpine:edge") as HTMLInputElement;
    expect(image.value).toBe("alpine:edge");
    const mount = screen.getByPlaceholderText("/mnt/volume") as HTMLInputElement;
    expect(mount.value).toBe("/mnt/volume");
    expect(screen.getByText(/ns1\/data-pvc/)).toBeTruthy();
  });

  it("emits submit with overrides on Browse click", async () => {
    const onsubmit = vi.fn();
    render(VolumeBrowserDialog, {
      props: {
        open: true,
        namespace: "ns1",
        pvcName: "data-pvc",
        initial: defaults(),
        onsubmit,
        oncancel: vi.fn(),
      },
    });
    await tick();
    const btn = screen.getByRole("button", {name: /^Browse$/});
    await fireEvent.click(btn);
    expect(onsubmit).toHaveBeenCalledTimes(1);
    const arg = onsubmit.mock.calls[0][0];
    expect(arg.image).toBe("alpine:edge");
    expect(arg.mountPath).toBe("/mnt/volume");
  });

  it("disables Browse when mount path is invalid", async () => {
    const onsubmit = vi.fn();
    render(VolumeBrowserDialog, {
      props: {
        open: true,
        namespace: "ns1",
        pvcName: "data-pvc",
        initial: defaults(),
        onsubmit,
        oncancel: vi.fn(),
      },
    });
    const mount = screen.getByPlaceholderText("/mnt/volume") as HTMLInputElement;
    await fireEvent.input(mount, {target: {value: "bad-path"}});
    await tick();
    const btn = screen.getByRole("button", {name: /^Browse$/}) as HTMLButtonElement;
    expect(btn.disabled).toBe(true);
  });

  it("invokes oncancel on Cancel click", async () => {
    const oncancel = vi.fn();
    render(VolumeBrowserDialog, {
      props: {
        open: true,
        namespace: "ns1",
        pvcName: "data-pvc",
        initial: defaults(),
        onsubmit: vi.fn(),
        oncancel,
      },
    });
    await tick();
    const btn = screen.getByRole("button", {name: /Cancel/});
    await fireEvent.click(btn);
    expect(oncancel).toHaveBeenCalled();
  });
});
