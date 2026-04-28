import {describe, it, expect, beforeEach} from "vitest";
import {sessionStore, SIDEBAR_MIN_WIDTH, SIDEBAR_MAX_WIDTH, SIDEBAR_DEFAULT_WIDTH} from "../session.svelte";

describe("sessionStore sidebarWidth", () => {
  beforeEach(() => {
    sessionStore.resetSidebarWidth();
  });

  it("setSidebarWidth clamps up to SIDEBAR_MIN_WIDTH", () => {
    sessionStore.setSidebarWidth(50);
    expect(sessionStore.sidebarWidth).toBe(SIDEBAR_MIN_WIDTH);
  });

  it("setSidebarWidth clamps down to SIDEBAR_MAX_WIDTH", () => {
    sessionStore.setSidebarWidth(9999);
    expect(sessionStore.sidebarWidth).toBe(SIDEBAR_MAX_WIDTH);
  });

  it("setSidebarWidth NaN falls back to SIDEBAR_DEFAULT_WIDTH", () => {
    sessionStore.setSidebarWidth(NaN);
    expect(sessionStore.sidebarWidth).toBe(SIDEBAR_DEFAULT_WIDTH);
  });

  it("resetSidebarWidth returns to SIDEBAR_DEFAULT_WIDTH", () => {
    sessionStore.setSidebarWidth(300);
    sessionStore.resetSidebarWidth();
    expect(sessionStore.sidebarWidth).toBe(SIDEBAR_DEFAULT_WIDTH);
  });

  it("restore with undefined sidebarWidth yields SIDEBAR_DEFAULT_WIDTH", () => {
    sessionStore.restore([], 0, false, undefined, undefined);
    expect(sessionStore.sidebarWidth).toBe(SIDEBAR_DEFAULT_WIDTH);
  });

  it("restore with 9999 clamps to SIDEBAR_MAX_WIDTH", () => {
    sessionStore.restore([], 0, false, undefined, 9999);
    expect(sessionStore.sidebarWidth).toBe(SIDEBAR_MAX_WIDTH);
  });
});
