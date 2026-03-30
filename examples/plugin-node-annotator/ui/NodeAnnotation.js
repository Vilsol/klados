import "./disclose-version-DKem_M_z.js";
import * as $ from "svelte/internal/client";
var root_2 = $.from_html(`<span class="text-xs bg-surface border border-border rounded px-1.5 py-0.5 font-mono"> </span>`);
var root_1 = $.from_html(`<div class="mt-3"><p class="text-xs text-muted mb-1">Taints:</p> <div class="flex flex-wrap gap-1"></div></div>`);
var root_3 = $.from_html(`<div class="flex items-center gap-2 text-sm text-muted"><div class="w-4 h-4 border-2 border-accent border-t-transparent rounded-full animate-spin"></div> Loading…</div>`);
var root_4 = $.from_html(`<div class="rounded border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"> </div>`);
var root_6 = $.from_html(`<tr class="border-b border-border hover:bg-surface-hover transition-colors"><td class="py-1.5 px-2 font-mono"> </td><td> </td><td class="py-1.5 px-2 text-right"> </td></tr>`);
var root_7 = $.from_html(`<tr><td colspan="3" class="py-4 px-2 text-center text-muted">No nodes found</td></tr>`);
var root_5 = $.from_html(`<table class="w-full text-xs border-collapse"><thead><tr class="border-b border-border text-muted"><th class="text-left py-1.5 px-2 font-medium">Name</th><th class="text-left py-1.5 px-2 font-medium">Readiness</th><th class="text-right py-1.5 px-2 font-medium">Taints</th></tr></thead><tbody><!><!></tbody></table>`);
var root = $.from_html(`<div class="p-4 overflow-auto h-full" style="font-family: inherit;"><div class="mb-4 rounded border border-border bg-surface p-3"><h3 class="text-xs font-semibold mb-2">This Node</h3> <div class="flex flex-wrap gap-4 text-xs"><div><span class="text-muted">Readiness</span> <div> </div></div> <div><span class="text-muted">Taint Count</span> <div class="mt-0.5 font-medium"> </div></div></div> <!></div> <h3 class="text-xs font-semibold mb-2"> </h3> <!></div>`);
function NodeAnnotation($$anchor, $$props) {
  $.push($$props, true);
  const status = $.derived(() => {
    var _a;
    return ((_a = $$props.resource) == null ? void 0 : _a.status) ?? {};
  });
  const spec = $.derived(() => {
    var _a;
    return ((_a = $$props.resource) == null ? void 0 : _a.spec) ?? {};
  });
  const taints = $.derived(() => $.get(spec).taints ?? []);
  const taintCount = $.derived(() => $.get(status).taintCount ?? $.get(taints).length);
  const readinessSummary = $.derived(() => $.get(status).readinessSummary ?? "Unknown");
  const ready = $.derived(() => $.get(readinessSummary) === "Ready");
  let nodes = $.state($.proxy([]));
  let loading = $.state(true);
  let errorMsg = $.state(null);
  $.user_effect(() => {
    var _a;
    if (!((_a = $$props.ctx) == null ? void 0 : _a.k8s)) {
      $.set(loading, false);
      return;
    }
    $$props.ctx.k8s.list("core.v1.nodes").then((items) => {
      $.set(
        nodes,
        items.map((n) => {
          var _a2, _b, _c, _d, _e;
          return {
            name: ((_a2 = n.metadata) == null ? void 0 : _a2.name) ?? "",
            taintCount: ((_b = n.status) == null ? void 0 : _b.taintCount) ?? ((_d = (_c = n.spec) == null ? void 0 : _c.taints) == null ? void 0 : _d.length) ?? 0,
            readiness: ((_e = n.status) == null ? void 0 : _e.readinessSummary) ?? "Unknown"
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
  var div_2 = $.sibling($.child(div_1), 2);
  var div_3 = $.child(div_2);
  var div_4 = $.sibling($.child(div_3), 2);
  let classes;
  var text = $.child(div_4, true);
  $.reset(div_4);
  $.reset(div_3);
  var div_5 = $.sibling(div_3, 2);
  var div_6 = $.sibling($.child(div_5), 2);
  var text_1 = $.child(div_6, true);
  $.reset(div_6);
  $.reset(div_5);
  $.reset(div_2);
  var node_1 = $.sibling(div_2, 2);
  {
    var consequent = ($$anchor2) => {
      var div_7 = root_1();
      var div_8 = $.sibling($.child(div_7), 2);
      $.each(div_8, 21, () => $.get(taints), $.index, ($$anchor3, taint) => {
        var span = root_2();
        var text_2 = $.child(span);
        $.reset(span);
        $.template_effect(() => $.set_text(text_2, `${$.get(taint).key ?? ""}${$.get(taint).value ? "=" + $.get(taint).value : ""}:${$.get(taint).effect ?? ""}`));
        $.append($$anchor3, span);
      });
      $.reset(div_8);
      $.reset(div_7);
      $.append($$anchor2, div_7);
    };
    $.if(node_1, ($$render) => {
      if ($.get(taints).length > 0) $$render(consequent);
    });
  }
  $.reset(div_1);
  var h3 = $.sibling(div_1, 2);
  var text_3 = $.child(h3);
  $.reset(h3);
  var node_2 = $.sibling(h3, 2);
  {
    var consequent_1 = ($$anchor2) => {
      var div_9 = root_3();
      $.append($$anchor2, div_9);
    };
    var consequent_2 = ($$anchor2) => {
      var div_10 = root_4();
      var text_4 = $.child(div_10, true);
      $.reset(div_10);
      $.template_effect(() => $.set_text(text_4, $.get(errorMsg)));
      $.append($$anchor2, div_10);
    };
    var alternate = ($$anchor2) => {
      var table = root_5();
      var tbody = $.sibling($.child(table));
      var node_3 = $.child(tbody);
      $.each(node_3, 17, () => $.get(nodes), (node) => node.name, ($$anchor3, node) => {
        const nodeReady = $.derived(() => $.get(node).readiness === "Ready");
        var tr = root_6();
        var td = $.child(tr);
        var text_5 = $.child(td, true);
        $.reset(td);
        var td_1 = $.sibling(td);
        let classes_1;
        var text_6 = $.child(td_1, true);
        $.reset(td_1);
        var td_2 = $.sibling(td_1);
        var text_7 = $.child(td_2, true);
        $.reset(td_2);
        $.reset(tr);
        $.template_effect(() => {
          $.set_text(text_5, $.get(node).name);
          classes_1 = $.set_class(td_1, 1, "py-1.5 px-2 font-medium", null, classes_1, {
            "text-green-500": $.get(nodeReady),
            "text-red-500": !$.get(nodeReady) && $.get(node).readiness !== "Unknown",
            "text-muted": $.get(node).readiness === "Unknown"
          });
          $.set_text(text_6, $.get(node).readiness);
          $.set_text(text_7, $.get(node).taintCount);
        });
        $.append($$anchor3, tr);
      });
      var node_4 = $.sibling(node_3);
      {
        var consequent_3 = ($$anchor3) => {
          var tr_1 = root_7();
          $.append($$anchor3, tr_1);
        };
        $.if(node_4, ($$render) => {
          if ($.get(nodes).length === 0) $$render(consequent_3);
        });
      }
      $.reset(tbody);
      $.reset(table);
      $.append($$anchor2, table);
    };
    $.if(node_2, ($$render) => {
      if ($.get(loading)) $$render(consequent_1);
      else if ($.get(errorMsg)) $$render(consequent_2, 1);
      else $$render(alternate, -1);
    });
  }
  $.reset(div);
  $.template_effect(() => {
    classes = $.set_class(div_4, 1, "mt-0.5 font-medium", null, classes, {
      "text-green-500": $.get(ready),
      "text-red-500": !$.get(ready)
    });
    $.set_text(text, $.get(readinessSummary));
    $.set_text(text_1, $.get(taintCount));
    $.set_text(text_3, `All Nodes (${$.get(nodes).length ?? ""})`);
  });
  $.append($$anchor, div);
  $.pop();
}
export {
  NodeAnnotation as default
};
//# sourceMappingURL=NodeAnnotation.js.map
