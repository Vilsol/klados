var _a;
import * as $ from "svelte/internal/client";
const PUBLIC_VERSION = "5";
if (typeof window !== "undefined") {
  ((_a = window.__svelte ?? (window.__svelte = {})).v ?? (_a.v = /* @__PURE__ */ new Set())).add(PUBLIC_VERSION);
}
var root_1 = $.from_html(`<div class="flex items-center gap-2 text-sm text-muted"><div class="w-4 h-4 border-2 border-accent border-t-transparent rounded-full animate-spin"></div> Loading deployments…</div>`);
var root_2 = $.from_html(`<div class="rounded border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"> </div>`);
var root_4 = $.from_html(`<tr class="border-b border-border hover:bg-surface-hover transition-colors"><td class="py-1.5 px-2 font-mono"> </td><td class="py-1.5 px-2 text-muted"> </td><td class="py-1.5 px-2 text-right"> </td><td> </td><td class="py-1.5 px-2 text-right"> </td></tr>`);
var root_5 = $.from_html(`<tr><td colspan="5" class="py-4 px-2 text-center text-muted">No deployments found</td></tr>`);
var root_3 = $.from_html(`<table class="w-full text-xs border-collapse"><thead><tr class="border-b border-border text-muted"><th class="text-left py-1.5 px-2 font-medium">Name</th><th class="text-left py-1.5 px-2 font-medium">Namespace</th><th class="text-right py-1.5 px-2 font-medium">Replicas</th><th class="text-right py-1.5 px-2 font-medium">Ready</th><th class="text-right py-1.5 px-2 font-medium">Available</th></tr></thead><tbody><!><!></tbody></table> <p class="mt-2 text-xs text-muted"> </p>`, 1);
var root = $.from_html(`<div class="p-4 overflow-auto h-full" style="font-family: inherit;"><div class="mb-4"><p class="text-xs text-muted mb-1"><span class="font-medium">Cluster:</span> <span class="font-medium">Namespace:</span> </p> <p class="text-xs text-muted">Listing all Deployments in namespace via <code>ctx.k8s.list('apps.v1.deployments')</code></p></div> <!></div>`);
function HelloTab($$anchor, $$props) {
  $.push($$props, true);
  let rows = $.state($.proxy([]));
  let loading = $.state(true);
  let errorMsg = $.state(null);
  $.user_effect(() => {
    var _a2, _b, _c;
    if (!((_a2 = $$props.ctx) == null ? void 0 : _a2.k8s)) {
      $.set(errorMsg, "k8s context not available — check plugin permissions");
      $.set(loading, false);
      return;
    }
    const ns = ((_c = (_b = $$props.resource) == null ? void 0 : _b.metadata) == null ? void 0 : _c.namespace) ?? "";
    $$props.ctx.k8s.list("apps.v1.deployments", ns).then((items) => {
      $.set(
        rows,
        items.map((d) => {
          var _a3, _b2, _c2, _d, _e;
          return {
            name: ((_a3 = d.metadata) == null ? void 0 : _a3.name) ?? "",
            namespace: ((_b2 = d.metadata) == null ? void 0 : _b2.namespace) ?? "",
            replicas: ((_c2 = d.spec) == null ? void 0 : _c2.replicas) ?? 0,
            ready: ((_d = d.status) == null ? void 0 : _d.readyReplicas) ?? 0,
            available: ((_e = d.status) == null ? void 0 : _e.availableReplicas) ?? 0
          };
        }),
        true
      );
      $.set(loading, false);
    }).catch((e) => {
      $.set(errorMsg, e instanceof Error ? e.message : String(e), true);
      $.set(loading, false);
    });
  });
  var div = root();
  var div_1 = $.child(div);
  var p = $.child(div_1);
  var text = $.sibling($.child(p));
  var text_1 = $.sibling(text, 2);
  $.reset(p);
  $.next(2);
  $.reset(div_1);
  var node = $.sibling(div_1, 2);
  {
    var consequent = ($$anchor2) => {
      var div_2 = root_1();
      $.append($$anchor2, div_2);
    };
    var consequent_1 = ($$anchor2) => {
      var div_3 = root_2();
      var text_2 = $.child(div_3, true);
      $.reset(div_3);
      $.template_effect(() => $.set_text(text_2, $.get(errorMsg)));
      $.append($$anchor2, div_3);
    };
    var alternate = ($$anchor2) => {
      var fragment = root_3();
      var table = $.first_child(fragment);
      var tbody = $.sibling($.child(table));
      var node_1 = $.child(tbody);
      $.each(node_1, 17, () => $.get(rows), (row) => row.name + "/" + row.namespace, ($$anchor3, row) => {
        var tr = root_4();
        var td = $.child(tr);
        var text_3 = $.child(td, true);
        $.reset(td);
        var td_1 = $.sibling(td);
        var text_4 = $.child(td_1, true);
        $.reset(td_1);
        var td_2 = $.sibling(td_1);
        var text_5 = $.child(td_2, true);
        $.reset(td_2);
        var td_3 = $.sibling(td_2);
        let classes;
        var text_6 = $.child(td_3, true);
        $.reset(td_3);
        var td_4 = $.sibling(td_3);
        var text_7 = $.child(td_4, true);
        $.reset(td_4);
        $.reset(tr);
        $.template_effect(() => {
          $.set_text(text_3, $.get(row).name);
          $.set_text(text_4, $.get(row).namespace);
          $.set_text(text_5, $.get(row).replicas);
          classes = $.set_class(td_3, 1, "py-1.5 px-2 text-right", null, classes, {
            "text-destructive": $.get(row).ready < $.get(row).replicas,
            "text-accent": $.get(row).ready >= $.get(row).replicas && $.get(row).replicas > 0
          });
          $.set_text(text_6, $.get(row).ready);
          $.set_text(text_7, $.get(row).available);
        });
        $.append($$anchor3, tr);
      });
      var node_2 = $.sibling(node_1);
      {
        var consequent_2 = ($$anchor3) => {
          var tr_1 = root_5();
          $.append($$anchor3, tr_1);
        };
        $.if(node_2, ($$render) => {
          if ($.get(rows).length === 0) $$render(consequent_2);
        });
      }
      $.reset(tbody);
      $.reset(table);
      var p_1 = $.sibling(table, 2);
      var text_8 = $.child(p_1);
      $.reset(p_1);
      $.template_effect(() => $.set_text(text_8, `${$.get(rows).length ?? ""} deployment${$.get(rows).length === 1 ? "" : "s"} total`));
      $.append($$anchor2, fragment);
    };
    $.if(node, ($$render) => {
      if ($.get(loading)) $$render(consequent);
      else if ($.get(errorMsg)) $$render(consequent_1, 1);
      else $$render(alternate, -1);
    });
  }
  $.reset(div);
  $.template_effect(() => {
    var _a2, _b;
    $.set_text(text, ` ${$$props.ctx.cluster.name ?? ""}
       ·  `);
    $.set_text(text_1, ` ${((_b = (_a2 = $$props.resource) == null ? void 0 : _a2.metadata) == null ? void 0 : _b.namespace) ?? "—" ?? ""}`);
  });
  $.append($$anchor, div);
  $.pop();
}
export {
  HelloTab as default
};
//# sourceMappingURL=HelloTab.js.map
