import {Events} from "@wailsio/runtime";
import type {Component} from "svelte";
import {notificationStore} from "$lib/stores/notification.svelte.js";
import * as PluginService from "../../../bindings/github.com/Vilsol/klados/internal/services/pluginservice.js";
import {getLogger} from "$lib/logger";

const log = getLogger("plugins");

const moduleCache = new Map<string, Component<any>>();

export async function loadPluginComponent(pluginName: string, componentPath: string, baseURL: string): Promise<Component<any> | null> {
  const url = `${baseURL}/${pluginName}/${componentPath}`;
  const cached = moduleCache.get(url);
  if (cached !== undefined) {
    return cached;
  }
  try {
    const mod = await import(/* @vite-ignore */ url);
    const component = mod.default as Component<any>;
    moduleCache.set(url, component);
    return component;
  } catch (err) {
    notificationStore.error(`Plugin "${pluginName}" component failed to load`, err instanceof Error ? err.message : String(err));
    PluginService.DisablePlugin(pluginName).catch((e) => log.warn("Failed to auto-disable broken plugin", {pluginName, error: String(e)}));
    return null;
  }
}

if (typeof window !== "undefined") {
  // Clear cached modules for a plugin when it reloads so the new version is fetched.
  Events.On("plugin:reloading", (wailsEvent: any) => {
    const name = wailsEvent?.data?.name ?? wailsEvent?.name;
    if (!name) {
      return;
    }
    for (const key of moduleCache.keys()) {
      if (key.includes(`/${name}/`)) {
        moduleCache.delete(key);
      }
    }
  });
}
