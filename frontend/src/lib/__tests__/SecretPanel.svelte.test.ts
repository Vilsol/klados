import {describe, it, expect, vi} from "vitest";
import {render, screen, fireEvent} from "@testing-library/svelte";
import SecretPanel from "$lib/components/panels/SecretPanel.svelte";

const obj = {
  metadata: {name: "my-secret", namespace: "default"},
  type: "kubernetes.io/tls",
  data: {
    "tls.crt": btoa("my-cert-content"),
    "tls.key": btoa("my-key-content"),
  },
};

describe("SecretPanel", () => {
  it("shows secret type badge", () => {
    render(SecretPanel, {props: {obj}});
    expect(screen.getByText("kubernetes.io/tls")).toBeTruthy();
  });

  it("shows data key names", () => {
    render(SecretPanel, {props: {obj}});
    expect(screen.getByText("tls.crt")).toBeTruthy();
    expect(screen.getByText("tls.key")).toBeTruthy();
  });

  it("hides values by default", () => {
    render(SecretPanel, {props: {obj}});
    // Values should be masked
    expect(screen.getAllByText("••••••••").length).toBeGreaterThan(0);
    // Decoded values should not be visible
    expect(screen.queryByText("my-cert-content")).toBeNull();
  });

  it("reveals value on eye button click", async () => {
    render(SecretPanel, {props: {obj}});
    const eyeButtons = screen.getAllByTitle("Reveal value");
    await fireEvent.click(eyeButtons[0]);
    expect(screen.getByText("my-cert-content")).toBeTruthy();
  });

  it("hides value again after second click", async () => {
    render(SecretPanel, {props: {obj}});
    const eyeButton = screen.getAllByTitle("Reveal value")[0];
    await fireEvent.click(eyeButton);
    const hideButton = screen.getByTitle("Hide value");
    await fireEvent.click(hideButton);
    expect(screen.queryByText("my-cert-content")).toBeNull();
  });

  it("shows key count", () => {
    render(SecretPanel, {props: {obj}});
    expect(screen.getByText(/2 keys/)).toBeTruthy();
  });

  it("renders empty state for no data", () => {
    render(SecretPanel, {props: {obj: {metadata: {}, type: "Opaque", data: {}}}});
    expect(screen.getByText("No data")).toBeTruthy();
  });

  it("copies decoded value on copy button click", async () => {
    const clipboardMock = {writeText: vi.fn().mockResolvedValue(undefined)};
    Object.defineProperty(navigator, "clipboard", {value: clipboardMock, configurable: true});

    render(SecretPanel, {props: {obj}});
    const copyButtons = screen.getAllByTitle("Copy decoded value");
    await fireEvent.click(copyButtons[0]);
    expect(clipboardMock.writeText).toHaveBeenCalledWith("my-cert-content");
  });
});
