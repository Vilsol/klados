import { describe, it, expect } from "vitest";
import {
  getConditions,
  computeHealth,
  findWarnings,
  type Condition,
} from "../conditions";

describe("getConditions", () => {
  it("returns [] when status.conditions missing", () => {
    expect(getConditions({ metadata: {} })).toEqual([]);
    expect(getConditions({ status: {} })).toEqual([]);
  });

  it("returns [] when conditions isn't a valid array of objects", () => {
    expect(getConditions({ status: { conditions: "oops" } })).toEqual([]);
    expect(getConditions({ status: { conditions: [{ notATypeField: true }] } })).toEqual([]);
  });

  it("extracts valid conditions", () => {
    const obj = {
      status: {
        conditions: [
          { type: "Ready", status: "True", reason: "Ok", message: "all good" },
          { type: "Available", status: "False" },
        ],
      },
    };
    const c = getConditions(obj);
    expect(c.length).toBe(2);
    expect(c[0].type).toBe("Ready");
    expect(c[0].status).toBe("True");
  });
});

describe("computeHealth", () => {
  const c = (type: string, status: "True" | "False" | "Unknown"): Condition => ({
    type, status, reason: "", message: "", lastTransitionTime: "",
  });

  it("returns unknown when no conditions", () => {
    expect(computeHealth([])).toEqual({ level: "unknown", reason: "no conditions" });
  });

  it("returns healthy when Ready=True and no negatives", () => {
    expect(computeHealth([c("Ready", "True")]).level).toBe("healthy");
  });

  it("returns unhealthy when Ready=False", () => {
    expect(computeHealth([c("Ready", "False")]).level).toBe("unhealthy");
  });

  it("returns unhealthy when Degraded=True", () => {
    expect(computeHealth([c("Degraded", "True"), c("Ready", "True")]).level).toBe("unhealthy");
  });

  it("returns progressing when only Progressing=True among positives", () => {
    expect(computeHealth([c("Progressing", "True")]).level).toBe("progressing");
  });

  it("falls back to True/False ratio when no recognized types", () => {
    const h = computeHealth([c("CustomOne", "True"), c("CustomTwo", "True"), c("CustomThree", "False")]);
    expect(h.level).toBe("mixed");
    expect(h.reason).toBe("2/3 True");
  });
});

describe("findWarnings", () => {
  const c = (type: string, status: "True" | "False" | "Unknown", reason = "", message = ""): Condition => ({
    type, status, reason, message, lastTransitionTime: "",
  });

  it("flags Ready=False", () => {
    const w = findWarnings([c("Ready", "False", "NotReady", "pod not ready")]);
    expect(w.length).toBe(1);
    expect(w[0].type).toBe("Ready");
    expect(w[0].message).toBe("pod not ready");
  });

  it("flags Degraded=True", () => {
    expect(findWarnings([c("Degraded", "True", "Issues", "degraded")]).length).toBe(1);
  });

  it("does not flag Ready=True or Degraded=False", () => {
    expect(findWarnings([c("Ready", "True"), c("Degraded", "False")])).toEqual([]);
  });
});
