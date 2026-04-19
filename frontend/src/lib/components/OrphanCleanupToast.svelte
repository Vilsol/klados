<script lang="ts">
  import {onMount} from "svelte";
  import {Events} from "@wailsio/runtime";
  import {clusterStore} from "$lib/stores/cluster.svelte";
  import {notificationStore} from "$lib/stores/notification.svelte";
  import {CleanupOrphans} from "../../../bindings/github.com/Vilsol/klados/internal/services/volumebrowserservice.js";
  import type {OrphanPodDTO} from "../../../bindings/github.com/Vilsol/klados/internal/services/models.js";
  import {getLogger} from "$lib/logger";

  const log = getLogger("volumebrowser.orphans");

  const subscriptions = new Map<string, () => void>();

  function subscribe(ctxName: string) {
    if (subscriptions.has(ctxName)) {
      return;
    }
    const unsub = Events.On(`volumebrowser:orphans:${ctxName}`, (wailsEvent: {data?: OrphanPodDTO[]}) => {
      const orphans = wailsEvent.data ?? [];
      if (orphans.length === 0) {
        return;
      }
      const details = orphans.map((o) => `${o.namespace}/${o.podName} (pvc ${o.pvcName})`).join("\n");
      let toastId = "";
      toastId = notificationStore.push(
        `Found ${orphans.length} leftover volume browser pod${orphans.length === 1 ? "" : "s"} on ${ctxName}`,
        "info",
        {
          details,
          actions: [
            {
              label: "Clean up",
              onClick: () => {
                CleanupOrphans(ctxName).catch((err: unknown) => {
                  log.error("CleanupOrphans failed", err);
                  notificationStore.error(`Failed to clean up orphans on ${ctxName}`, String(err));
                });
              },
            },
            {
              label: "Dismiss",
              onClick: () => {
                notificationStore.dismiss(toastId);
              },
            },
          ],
        },
      );
    });
    subscriptions.set(ctxName, unsub);
  }

  onMount(() => {
    // Subscribe reactively for every known context. Contexts are static for the
    // lifetime of the app (reload on kubeconfig change), so one subscription per
    // context is sufficient.
    return () => {
      for (const unsub of subscriptions.values()) {
        unsub();
      }
      subscriptions.clear();
    };
  });

  $effect(() => {
    for (const ctx of clusterStore.contexts) {
      subscribe(ctx.name);
    }
  });
</script>
