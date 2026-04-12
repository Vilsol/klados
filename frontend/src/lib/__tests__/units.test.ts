import {describe, it, expect} from "vitest";
import {formatCPU, formatMemory, formatRatio} from "../components/charts/units";

describe("formatCPU", () => {
  it("formats sub-core values as millicores", () => {
    expect(formatCPU(0.5)).toBe("500m");
    expect(formatCPU(0.1)).toBe("100m");
    expect(formatCPU(0.001)).toBe("1m");
    expect(formatCPU(0.2505)).toBe("250.5m");
  });

  it("formats sub-millicores with decimal precision", () => {
    expect(formatCPU(0.0005)).toBe("0.5m");
    expect(formatCPU(0.0003)).toBe("0.3m");
  });

  it("formats >= 1 core as decimal", () => {
    expect(formatCPU(2.5)).toBe("2.5");
    expect(formatCPU(1)).toBe("1");
    expect(formatCPU(4)).toBe("4");
  });
});

describe("formatMemory", () => {
  it("formats GiB", () => {
    expect(formatMemory(1_073_741_824)).toBe("1 GiB");
    expect(formatMemory(1_073_741_824 * 4.2)).toBe("4.2 GiB");
  });

  it("formats MiB", () => {
    expect(formatMemory(134_217_728)).toBe("128 MiB");
  });
});

describe("formatRatio", () => {
  it("formats as percentage", () => {
    expect(formatRatio(0.45)).toBe("45.0%");
    expect(formatRatio(1)).toBe("100.0%");
  });
});
