import {describe, it, expect, vi} from "vitest";
import {render, fireEvent} from "@testing-library/svelte";
import {tick} from "svelte";
import SmartSearch from "$lib/components/SmartSearch.svelte";

const items = [
  {metadata: {name: "nginx-proxy", namespace: "default", labels: {app: "web"}, annotations: {}}},
  {metadata: {name: "redis-master", namespace: "kube-system", labels: {app: "cache"}, annotations: {}}},
];

describe("SmartSearch", () => {
  it("renders the search input", () => {
    const {container} = render(SmartSearch, {props: {items}});
    const input = container.querySelector("input");
    expect(input).toBeTruthy();
  });

  it("calls ontermschange when input changes", async () => {
    const ontermschange = vi.fn();
    const {container} = render(SmartSearch, {props: {items, ontermschange}});
    const input = container.querySelector("input") as HTMLInputElement;

    input.value = "nginx";
    await fireEvent.input(input);
    await tick();

    expect(ontermschange).toHaveBeenCalled();
    const lastCall = ontermschange.mock.calls.at(-1);
    expect(lastCall[0]).toEqual([{type: "text", value: "nginx", negated: false}]);
  });

  it("shows autocomplete when typing a qualifier prefix", async () => {
    const {container} = render(SmartSearch, {props: {items}});
    const input = container.querySelector("input") as HTMLInputElement;

    await fireEvent.focus(input);
    input.value = "lab";
    await fireEvent.input(input);
    await tick();

    const popup = container.querySelector('[role="listbox"]');
    expect(popup).toBeTruthy();
  });

  it("shows placeholder when input is empty", () => {
    const {container} = render(SmartSearch, {props: {items}});
    const input = container.querySelector("input") as HTMLInputElement;
    expect(input.getAttribute("placeholder")).toContain("Filter resources");
  });

  it("clears input text when space creates a chip", async () => {
    const {container} = render(SmartSearch, {props: {items}});
    const input = container.querySelector("input") as HTMLInputElement;

    input.value = "nginx ";
    await fireEvent.input(input);
    await tick();

    expect(input.value).toBe("");
    const chips = container.querySelectorAll("span.inline-flex");
    expect(chips.length).toBe(1);
  });

  it("does not create duplicate chips from repeated spaces", async () => {
    const {container} = render(SmartSearch, {props: {items}});
    const input = container.querySelector("input") as HTMLInputElement;

    input.value = "nginx ";
    await fireEvent.input(input);
    await tick();

    // Press space again — input is empty so no new chip should be created
    input.value = " ";
    await fireEvent.input(input);
    await tick();

    const chips = container.querySelectorAll("span.inline-flex");
    expect(chips.length).toBe(1);
  });

  it("shows clear button when input has content", async () => {
    const {container} = render(SmartSearch, {props: {items}});
    const input = container.querySelector("input") as HTMLInputElement;

    expect(container.querySelector('button[title="Clear filter"]')).toBeNull();

    input.value = "nginx";
    await fireEvent.input(input);
    await tick();

    expect(container.querySelector('button[title="Clear filter"]')).toBeTruthy();
  });

  it("clears everything when clear button is clicked", async () => {
    const ontermschange = vi.fn();
    const {container} = render(SmartSearch, {props: {items, ontermschange}});
    const input = container.querySelector("input") as HTMLInputElement;

    input.value = "nginx ";
    await fireEvent.input(input);
    await tick();

    const clearBtn = container.querySelector('button[title="Clear filter"]') as HTMLButtonElement;
    await fireEvent.click(clearBtn);
    await tick();

    expect(input.value).toBe("");
    expect(container.querySelectorAll("span.inline-flex").length).toBe(0);
  });
});
