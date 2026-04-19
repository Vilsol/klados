import { SvelteMap } from "svelte/reactivity";

/**
 * Global cache of watched resources, indexed by (contextName, gvr) → (uid → object).
 * Populated by ResourceStore instances as they receive watch events. Used by
 * RelatedResourcesPanel for reverse-lookup by ownerReferences.uid.
 *
 * Uses SvelteMap (not plain Map) so set/delete mutations trigger reactivity
 * in consumers that read the cache inside $derived.
 */
class ResourceCache {
  private cache = new SvelteMap<string, SvelteMap<string, Record<string, unknown>>>();

  private keyFor(ctx: string, gvr: string): string {
    return `${ctx}::${gvr}`;
  }

  upsert(ctx: string, gvr: string, obj: Record<string, unknown>): void {
    const uid = (obj as any)?.metadata?.uid as string | undefined;
    if (!uid) return;
    const key = this.keyFor(ctx, gvr);
    let m = this.cache.get(key);
    if (!m) {
      m = new SvelteMap();
      this.cache.set(key, m);
    }
    m.set(uid, obj);
  }

  remove(ctx: string, gvr: string, uid: string): void {
    const m = this.cache.get(this.keyFor(ctx, gvr));
    m?.delete(uid);
  }

  /**
   * Look up a single object in the given (ctx, gvr) by namespace + name.
   * Returns undefined if the watch isn't active or the object isn't present.
   */
  findByNamespaceName(ctx: string, gvr: string, namespace: string, name: string): Record<string, unknown> | undefined {
    const m = this.cache.get(this.keyFor(ctx, gvr));
    if (!m) return undefined;
    for (const obj of m.values()) {
      const md = (obj as {metadata?: {namespace?: string; name?: string}}).metadata;
      if (md?.namespace === namespace && md?.name === name) return obj;
    }
    return undefined;
  }

  /**
   * Scan all watched GVRs (in the given context) for objects whose
   * ownerReferences include the given ownerUid. Returns results grouped by
   * GVR for display.
   */
  findByOwnerUID(ctx: string, ownerUid: string): Array<{ gvr: string; items: Record<string, unknown>[] }> {
    const results: Array<{ gvr: string; items: Record<string, unknown>[] }> = [];
    for (const [key, byUid] of this.cache.entries()) {
      if (!key.startsWith(`${ctx}::`)) continue;
      const gvr = key.slice(ctx.length + 2);
      const matches: Record<string, unknown>[] = [];
      for (const obj of byUid.values()) {
        const refs = (obj as any)?.metadata?.ownerReferences;
        if (!Array.isArray(refs)) continue;
        if (refs.some((r: any) => r?.uid === ownerUid)) {
          matches.push(obj);
        }
      }
      if (matches.length > 0) results.push({ gvr, items: matches });
    }
    return results;
  }
}

export const resourceCache = new ResourceCache();
