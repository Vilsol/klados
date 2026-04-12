import {stringify} from "yaml";

export function exportItems(items: Record<string, unknown>[], gvr: string, format: "yaml" | "json") {
  if (items.length === 0) {
    return;
  }

  let content: string;
  let ext: string;
  if (format === "yaml") {
    content = items.map((item) => stringify(item)).join("---\n");
    ext = "yaml";
  } else {
    content = JSON.stringify(items, null, 2);
    ext = "json";
  }

  const timestamp = new Date().toISOString().replace(/[:.]/g, "-").slice(0, 19);
  const filename = `${gvr}-${timestamp}.${ext}`;
  const blob = new Blob([content], {type: "text/plain"});
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
