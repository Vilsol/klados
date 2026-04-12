<script lang="ts">
  import Router from "svelte-spa-router";
  import {routes} from "./routes/routes";
  import Layout from "$lib/components/Layout.svelte";
  import {Notification} from "@klados/ui";
  import CommandPalette from "$lib/components/CommandPalette.svelte";
  import {onMount} from "svelte";
  import {clusterStore} from "$lib/stores/cluster.svelte";
  import {sessionStore} from "$lib/stores/session.svelte";
  import {setTheme} from "$lib/theme.svelte";
  import {descriptorRegistry} from "$lib/registry/index";
  import {shortcutStore} from "$lib/stores/shortcuts.svelte";
  import {Events} from "@wailsio/runtime";
  import * as ConfigService from "../bindings/github.com/Vilsol/klados/internal/services/configservice.js";
  import * as AppService from "../bindings/github.com/Vilsol/klados/internal/services/appservice.js";
  import type {TabState} from "../bindings/github.com/Vilsol/klados/internal/session/models.js";
  import PanelWindow from "$lib/components/PanelWindow.svelte";
  import StackTrace from "stacktrace-js";
  import CreateResourceDialog from "$lib/components/CreateResourceDialog.svelte";
  import {createResourceStore} from "$lib/stores/createResource.svelte";
  import ApplyManifestDialog from "$lib/components/ApplyManifestDialog.svelte";
  import {applyManifestStore} from "$lib/stores/applyManifest.svelte";
  import {preferencesStore} from "$lib/stores/preferences.svelte";

  const panelId = new URLSearchParams(window.location.search).get("panel");
  let paletteOpen = $state(false);

  function logToBackend(level: string, message: string, detail: string) {
    AppService.LogFrontend(level, message, detail).catch(() => {});
  }

  async function logError(msg: string, err: Error | undefined) {
    let stack = "";
    if (err) {
      try {
        const frames = await StackTrace.fromError(err);
        stack = frames
          .map((f) => `  at ${f.getFunctionName() ?? "<anonymous>"} (${f.getFileName()}:${f.getLineNumber()}:${f.getColumnNumber()})`)
          .join("\n");
      } catch {
        stack = err.stack ?? "";
      }
    }
    logToBackend("error", msg, stack);
  }

  function argsToLog(args: any[]): [string, Error | undefined] {
    const err = args.find((a) => a instanceof Error) as Error | undefined;
    const msg = args.map((a) => (a instanceof Error ? a.message : String(a))).join(" ");
    return [msg, err];
  }

  onMount(() => {
    const origDebug = console.debug.bind(console);
    const origError = console.error.bind(console);
    const origWarn = console.warn.bind(console);
    console.debug = (...args) => {
      const [m] = argsToLog(args);
      logToBackend("debug", m, "");
    };
    console.error = (...args) => {
      origError(...args);
      const [m, e] = argsToLog(args);
      logError(m, e);
    };
    console.warn = (...args) => {
      origWarn(...args);
      const [m, e] = argsToLog(args);
      logToBackend("warn", m, e?.stack ?? "");
    };
    window.onerror = (_msg, _src, _line, _col, err) => {
      logError(String(_msg), err);
    };
    window.onunhandledrejection = (e) => {
      const err = e.reason instanceof Error ? e.reason : undefined;
      const msg = err ? err.message : String(e.reason);
      logError(`Unhandled rejection: ${msg}`, err);
    };

    let unsubPlugins: (() => void) | undefined;
    let handler: ((e: KeyboardEvent) => void) | undefined;

    (async () => {
      try {
        const sess = await AppService.GetSession();
        if (sess) {
          const tabs = (sess.openTabs ?? []).map((t: TabState) => ({
            clusterContext: t.clusterContext ?? "",
            gvr: t.gvr ?? "",
            namespace: t.namespace ?? "",
            name: t.name ?? "",
            scrollPosition: t.scrollPosition ?? 0,
          }));
          sessionStore.restore(tabs, sess.activeTab ?? 0, sess.sidebarCollapsed ?? false, sess.terminalFontSize || undefined);
        }
      } catch {
        // Session restore not available
      }

      await clusterStore.loadContexts();
      preferencesStore.subscribe();
      await preferencesStore.load(clusterStore.activeContext ?? "");
      await descriptorRegistry.load();

      unsubPlugins = Events.On("plugins:loaded", () => {
        descriptorRegistry.reloadPlugins();
      });

      shortcutStore.register({
        id: "command-palette",
        keys: "Control+k",
        description: "Open command palette",
        modes: ["normal", "editor"],
        action: () => {
          paletteOpen = true;
        },
      });

      handler = (e: KeyboardEvent) => shortcutStore.dispatch(e);
      window.addEventListener("keydown", handler);

      // WebKitGTK doesn't wire Ctrl+Z/Y to undo/redo for native inputs
      window.addEventListener("keydown", (e: KeyboardEvent) => {
        const el = document.activeElement;
        if (!(el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement)) return;
        if (e.ctrlKey && !e.altKey && !e.metaKey) {
          if (e.key === "z" && !e.shiftKey) {
            e.preventDefault();
            document.execCommand("undo");
          } else if ((e.key === "z" && e.shiftKey) || e.key === "y") {
            e.preventDefault();
            document.execCommand("redo");
          }
        }
      });
    })();

    return () => {
      if (handler) window.removeEventListener("keydown", handler);
      unsubPlugins?.();
      preferencesStore.destroy();
    };
  });

  // Reload preferences when active cluster changes
  $effect(() => {
    const ctx = clusterStore.activeContext;
    preferencesStore.load(ctx ?? "");
  });

  $effect(() => {
    const theme = preferencesStore.prefs.theme;
    if (theme) setTheme(theme as "light" | "dark" | "system");
  });

  $effect(() => {
    const color = preferencesStore.prefs.accentColor;
    if (color) {
      document.documentElement.style.setProperty("--color-accent", color);
    } else {
      document.documentElement.style.removeProperty("--color-accent");
    }
  });

  $effect(() => {
    const size = preferencesStore.prefs.fontSize;
    if (size > 0) {
      document.documentElement.style.setProperty("font-size", `${size}px`);
    } else {
      document.documentElement.style.removeProperty("font-size");
    }
  });

  // Persist UI state whenever tabs/sidebar/fontSize change
  $effect(() => {
    const tabs = sessionStore.tabs;
    const activeTab = sessionStore.activeTabIndex;
    const sidebarCollapsed = sessionStore.sidebarCollapsed;
    const terminalFontSize = sessionStore.terminalFontSize;
    AppService.SaveUIState(
      tabs.map((t) => ({
        clusterContext: t.clusterContext,
        gvr: t.gvr,
        namespace: t.namespace,
        name: t.name,
        scrollPosition: t.scrollPosition ?? 0,
      })),
      activeTab,
      sidebarCollapsed,
      terminalFontSize,
    ).catch(() => {});
  });
</script>

{#if panelId}
  <PanelWindow {panelId} />
{:else}
  <Layout> <Router {routes} /> </Layout>
  <CommandPalette bind:open={paletteOpen} />
  <CreateResourceDialog
    bind:open={createResourceStore.open}
    ctxName={clusterStore.activeContext ?? ''}
    gvr={createResourceStore.gvr}
    onsuccess={createResourceStore.onsuccess}
  />
  <ApplyManifestDialog bind:open={applyManifestStore.open} ctxName={clusterStore.activeContext ?? ''} />
{/if}
<Notification />
