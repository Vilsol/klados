// biome-ignore lint/performance/noBarrelFile: intentional shared runtime bundle for plugin system
// biome-ignore lint/performance/noReExportAll: re-exports entire Svelte runtime for plugin consumers
export * from "svelte/internal/client";
// biome-ignore lint/performance/noReExportAll: re-exports entire Svelte API for plugin consumers
export * from "svelte";
