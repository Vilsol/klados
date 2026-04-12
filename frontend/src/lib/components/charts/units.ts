export function formatCPU(cores: number): string {
  if (cores === 0) {
    return "0m";
  }
  const m = cores * 1000;
  if (m < 1) {
    // sub-millicores: keep 2 significant digits (e.g. 0.35m)
    return `${parseFloat(m.toPrecision(2))}m`;
  }
  if (cores < 1) {
    return `${parseFloat(m.toFixed(2))}m`;
  }
  return parseFloat(cores.toFixed(2)).toString();
}

export function formatMemory(bytes: number): string {
  const gib = bytes / (1024 * 1024 * 1024);
  if (gib >= 1) {
    return `${parseFloat(gib.toFixed(2))} GiB`;
  }
  const mib = bytes / (1024 * 1024);
  if (mib >= 1) {
    return `${parseFloat(mib.toFixed(2))} MiB`;
  }
  const kib = bytes / 1024;
  if (kib >= 1) {
    return `${parseFloat(kib.toFixed(2))} KiB`;
  }
  return `${bytes} B`;
}

export function formatRatio(r: number): string {
  return `${(r * 100).toFixed(1)}%`;
}

export function getFormatter(unit: string): (val: number) => string {
  switch (unit) {
    case "cores":
      return formatCPU;
    case "bytes":
      return formatMemory;
    case "ratio":
      return formatRatio;
    default:
      return (v) => v.toFixed(2);
  }
}
