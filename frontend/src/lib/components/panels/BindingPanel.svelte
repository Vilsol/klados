<script lang="ts">
  import {push} from "svelte-spa-router";
  import {SectionHeader, EmptyState} from "@klados/ui";

  let {obj, ctxName}: {obj: Record<string, any>; ctxName: string} = $props();

  const roleRef = $derived(obj.roleRef ?? {});
  const subjects = $derived<any[]>(obj.subjects ?? []);

  function roleRefURL(): string {
    if (roleRef.kind === "ClusterRole") {
      return `/c/${ctxName}/rbac.authorization.k8s.io.v1.clusterroles/_/${roleRef.name}`;
    }
    return `/c/${ctxName}/rbac.authorization.k8s.io.v1.roles/${obj.metadata?.namespace}/${roleRef.name}`;
  }
</script>

<div class="p-4 space-y-6">
  <section>
    <SectionHeader>Role Reference</SectionHeader>
    <div class="flex items-center gap-2">
      <span class="px-1.5 py-0.5 rounded text-xs font-medium bg-surface border border-border text-muted"> {roleRef.kind ?? ''} </span>
      <button onclick={() => push(roleRefURL())} class="text-xs text-accent hover:underline font-medium">{roleRef.name ?? ''}</button>
    </div>
  </section>

  <section>
    <SectionHeader>Subjects</SectionHeader>
    {#if subjects.length === 0}
      <EmptyState message="No subjects" />
    {:else}
      <div class="border border-border rounded overflow-hidden">
        <table class="w-full text-xs">
          <thead class="bg-surface">
            <tr>
              <th class="px-3 py-2 text-left font-medium text-muted w-28">Kind</th>
              <th class="px-3 py-2 text-left font-medium text-muted">Name</th>
              <th class="px-3 py-2 text-left font-medium text-muted">Namespace</th>
            </tr>
          </thead>
          <tbody>
            {#each subjects as subj}
              <tr class="border-t border-border">
                <td class="px-3 py-2 text-muted">{subj.kind ?? ''}</td>
                <td class="px-3 py-2 font-medium">
                  {#if subj.kind === 'ServiceAccount'}
                    <button
                      onclick={() => push(`/c/${ctxName}/core.v1.serviceaccounts/${subj.namespace}/${subj.name}`)}
                      class="text-accent hover:underline"
                    >
                      {subj.name ?? ''}
                    </button>
                  {:else}
                    {subj.name ?? ''}
                  {/if}
                </td>
                <td class="px-3 py-2 text-muted">{subj.namespace ?? ''}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </section>
</div>
