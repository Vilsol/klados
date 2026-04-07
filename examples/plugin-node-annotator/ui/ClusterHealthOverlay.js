import "./disclose-version-DKem_M_z.js";
import * as $ from "svelte/internal/client";
var root_1 = $.from_html(`<div class="flex items-center gap-2 text-sm text-muted"><div class="w-4 h-4 border-2 border-accent border-t-transparent rounded-full animate-spin"></div> Loading…</div>`);
var root_2 = $.from_html(`<div class="rounded border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"> </div>`);
var root_4 = $.from_html(`<tr class="border-b border-border hover:bg-surface-hover"><td class="py-1.5 px-2 font-mono"> </td><td> </td><td class="py-1.5 px-2 text-right"> </td></tr>`);
var root_6 = $.from_html(`<tr class="border-b border-border hover:bg-surface-hover"><td class="py-1.5 px-2 font-mono text-destructive"> </td><td class="py-1.5 px-2 text-muted"> </td><td class="py-1.5 px-2 text-right"> </td></tr>`);
var root_5 = $.from_html(`<div><h3 class="text-xs font-semibold text-muted uppercase tracking-wide mb-2">Taint Key Breakdown</h3> <table class="w-full text-xs border-collapse"><thead><tr class="border-b border-border text-muted"><th class="text-left py-1.5 px-2 font-medium">Key</th><th class="text-left py-1.5 px-2 font-medium">Effects</th><th class="text-right py-1.5 px-2 font-medium">Nodes</th></tr></thead><tbody></tbody></table></div>`);
var root_3 = $.from_html(`<div class="rounded border border-border bg-surface p-3 flex gap-6 text-xs"><div><div class="text-muted mb-0.5">Nodes Ready</div> <div> </div></div> <div><div class="text-muted mb-0.5">Unique Taint Keys</div> <div class="font-semibold text-sm"> </div></div> <div><div class="text-muted mb-0.5">Cluster</div> <div class="font-semibold text-sm font-mono"> </div></div></div> <div><h3 class="text-xs font-semibold text-muted uppercase tracking-wide mb-2">Nodes</h3> <table class="w-full text-xs border-collapse"><thead><tr class="border-b border-border text-muted"><th class="text-left py-1.5 px-2 font-medium">Name</th><th class="text-left py-1.5 px-2 font-medium">Readiness</th><th class="text-right py-1.5 px-2 font-medium">Taints</th></tr></thead><tbody></tbody></table></div> <!>`, 1);
var root = $.from_html(`<div class="fixed inset-0 z-50 flex items-center justify-center" style="background: rgba(0,0,0,0.5);" role="dialog" aria-modal="true"><div class="bg-surface border border-border rounded-lg shadow-xl w-[560px] max-h-[80vh] flex flex-col overflow-hidden" style="font-family: inherit;"><div class="flex items-center justify-between px-4 py-3 border-b border-border shrink-0"><h2 class="text-sm font-semibold">Cluster Node Health</h2> <button class="text-muted hover:text-fg transition-colors text-lg leading-none" aria-label="Close">✕</button></div> <div class="flex-1 overflow-y-auto p-4 space-y-4"><!></div></div></div>`);
function ClusterHealthOverlay($$anchor, $$props) {
  $.push($$props, true);
  let overlay;
  let loading = $.state(true);
  let errorMsg = $.state(null);
  let nodes = $.state($.proxy([]));
  let taintKeys = $.state($.proxy([]));
  $.user_effect(() => {
    var _a;
    if (!((_a = $$props.ctx) == null ? void 0 : _a.k8s)) {
      $.set(loading, false);
      return;
    }
    (async () => {
      var _a2;
      try {
        const items = await $$props.ctx.k8s.list("core.v1.nodes");
        $.set(
          nodes,
          items.map((n) => {
            var _a3, _b, _c, _d, _e, _f;
            return {
              name: ((_a3 = n.metadata) == null ? void 0 : _a3.name) ?? "",
              readiness: ((_b = n.status) == null ? void 0 : _b.readinessSummary) ?? "Unknown",
              ready: (((_c = n.status) == null ? void 0 : _c.readinessSummary) ?? "") === "Ready",
              taintCount: ((_d = n.status) == null ? void 0 : _d.taintCount) ?? ((_f = (_e = n.spec) == null ? void 0 : _e.taints) == null ? void 0 : _f.length) ?? 0
            };
          }),
          true
        );
        const keyMap = /* @__PURE__ */ new Map();
        for (const n of items) {
          for (const t of ((_a2 = n.spec) == null ? void 0 : _a2.taints) ?? []) {
            const entry = keyMap.get(t.key) ?? { count: 0, effects: /* @__PURE__ */ new Set() };
            entry.count++;
            if (t.effect) entry.effects.add(t.effect);
            keyMap.set(t.key, entry);
          }
        }
        $.set(taintKeys, [...keyMap.entries()].map(([key, v]) => ({ key, count: v.count, effects: [...v.effects] })), true);
        $.set(loading, false);
      } catch (e) {
        $.set(errorMsg, e instanceof Error ? e.message : String(e), true);
        $.set(loading, false);
      }
    })();
  });
  const readyCount = $.derived(() => $.get(nodes).filter((n) => n.ready).length);
  function dismiss() {
    overlay == null ? void 0 : overlay.remove();
  }
  var div = root();
  var div_1 = $.child(div);
  var div_2 = $.child(div_1);
  var button = $.sibling($.child(div_2), 2);
  $.reset(div_2);
  var div_3 = $.sibling(div_2, 2);
  var node_1 = $.child(div_3);
  {
    var consequent = ($$anchor2) => {
      var div_4 = root_1();
      $.append($$anchor2, div_4);
    };
    var consequent_1 = ($$anchor2) => {
      var div_5 = root_2();
      var text = $.child(div_5, true);
      $.reset(div_5);
      $.template_effect(() => $.set_text(text, $.get(errorMsg)));
      $.append($$anchor2, div_5);
    };
    var alternate = ($$anchor2) => {
      var fragment = root_3();
      var div_6 = $.first_child(fragment);
      var div_7 = $.child(div_6);
      var div_8 = $.sibling($.child(div_7), 2);
      let classes;
      var text_1 = $.child(div_8);
      $.reset(div_8);
      $.reset(div_7);
      var div_9 = $.sibling(div_7, 2);
      var div_10 = $.sibling($.child(div_9), 2);
      var text_2 = $.child(div_10, true);
      $.reset(div_10);
      $.reset(div_9);
      var div_11 = $.sibling(div_9, 2);
      var div_12 = $.sibling($.child(div_11), 2);
      var text_3 = $.child(div_12, true);
      $.reset(div_12);
      $.reset(div_11);
      $.reset(div_6);
      var div_13 = $.sibling(div_6, 2);
      var table = $.sibling($.child(div_13), 2);
      var tbody = $.sibling($.child(table));
      $.each(tbody, 21, () => $.get(nodes), (node) => node.name, ($$anchor3, node) => {
        var tr = root_4();
        var td = $.child(tr);
        var text_4 = $.child(td, true);
        $.reset(td);
        var td_1 = $.sibling(td);
        let classes_1;
        var text_5 = $.child(td_1, true);
        $.reset(td_1);
        var td_2 = $.sibling(td_1);
        var text_6 = $.child(td_2, true);
        $.reset(td_2);
        $.reset(tr);
        $.template_effect(() => {
          $.set_text(text_4, $.get(node).name);
          classes_1 = $.set_class(td_1, 1, "py-1.5 px-2 font-medium", null, classes_1, {
            "text-green-500": $.get(node).ready,
            "text-red-500": !$.get(node).ready && $.get(node).readiness !== "Unknown",
            "text-muted": $.get(node).readiness === "Unknown"
          });
          $.set_text(text_5, $.get(node).readiness);
          $.set_text(text_6, $.get(node).taintCount);
        });
        $.append($$anchor3, tr);
      });
      $.reset(tbody);
      $.reset(table);
      $.reset(div_13);
      var node_2 = $.sibling(div_13, 2);
      {
        var consequent_2 = ($$anchor3) => {
          var div_14 = root_5();
          var table_1 = $.sibling($.child(div_14), 2);
          var tbody_1 = $.sibling($.child(table_1));
          $.each(tbody_1, 21, () => $.get(taintKeys), (tk) => tk.key, ($$anchor4, tk) => {
            var tr_1 = root_6();
            var td_3 = $.child(tr_1);
            var text_7 = $.child(td_3, true);
            $.reset(td_3);
            var td_4 = $.sibling(td_3);
            var text_8 = $.child(td_4, true);
            $.reset(td_4);
            var td_5 = $.sibling(td_4);
            var text_9 = $.child(td_5, true);
            $.reset(td_5);
            $.reset(tr_1);
            $.template_effect(
              ($0) => {
                $.set_text(text_7, $.get(tk).key);
                $.set_text(text_8, $0);
                $.set_text(text_9, $.get(tk).count);
              },
              [() => $.get(tk).effects.join(", ") || "—"]
            );
            $.append($$anchor4, tr_1);
          });
          $.reset(tbody_1);
          $.reset(table_1);
          $.reset(div_14);
          $.append($$anchor3, div_14);
        };
        $.if(node_2, ($$render) => {
          if ($.get(taintKeys).length > 0) $$render(consequent_2);
        });
      }
      $.template_effect(() => {
        classes = $.set_class(div_8, 1, "font-semibold text-sm", null, classes, {
          "text-green-500": $.get(readyCount) === $.get(nodes).length,
          "text-red-500": $.get(readyCount) < $.get(nodes).length
        });
        $.set_text(text_1, `${$.get(readyCount) ?? ""} / ${$.get(nodes).length ?? ""}`);
        $.set_text(text_2, $.get(taintKeys).length);
        $.set_text(text_3, $$props.ctx.cluster.name || "—");
      });
      $.append($$anchor2, fragment);
    };
    $.if(node_1, ($$render) => {
      if ($.get(loading)) $$render(consequent);
      else if ($.get(errorMsg)) $$render(consequent_1, 1);
      else $$render(alternate, -1);
    });
  }
  $.reset(div_3);
  $.reset(div_1);
  $.reset(div);
  $.bind_this(div, ($$value) => overlay = $$value, () => overlay);
  $.delegated("click", button, dismiss);
  $.append($$anchor, div);
  $.pop();
}
$.delegate(["click"]);
export {
  ClusterHealthOverlay as default
};
//# sourceMappingURL=ClusterHealthOverlay.js.map
