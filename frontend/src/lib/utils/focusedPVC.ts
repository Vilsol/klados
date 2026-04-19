import {clusterStore} from "$lib/stores/cluster.svelte";
import {selectionStore} from "$lib/stores/selection.svelte";

const PVC_GVR = "core.v1.persistentvolumeclaims";

export function focusedPVC(): {contextName: string; namespace: string; name: string} | null {
  const hash = typeof location !== "undefined" ? location.hash : "";
  const detailMatch = hash.match(/^#\/c\/([^/]+)\/([^/]+)\/([^/]+)\/([^/]+)$/);
  if (detailMatch && decodeURIComponent(detailMatch[2]) === PVC_GVR) {
    return {
      contextName: decodeURIComponent(detailMatch[1]),
      namespace: decodeURIComponent(detailMatch[3]),
      name: decodeURIComponent(detailMatch[4]),
    };
  }
  if (selectionStore.selectedGVR === PVC_GVR && selectionStore.count === 1) {
    const item = selectionStore.items()[0];
    const ns = (item?.metadata as {namespace?: string} | undefined)?.namespace ?? "";
    const nm = (item?.metadata as {name?: string} | undefined)?.name ?? "";
    const ctx = clusterStore.activeContext ?? "";
    if (ctx && nm) {
      return {contextName: ctx, namespace: ns, name: nm};
    }
  }
  return null;
}
