import "./disclose-version-DKem_M_z.js";
import * as $ from "svelte/internal/client";
var root = $.from_html(`<span> </span>`);
function NodeTaintBadge($$anchor, $$props) {
  $.push($$props, true);
  const count = $.derived(() => {
    var _a, _b, _c, _d, _e;
    return ((_b = (_a = $$props.resource) == null ? void 0 : _a.status) == null ? void 0 : _b.taintCount) ?? ((_e = (_d = (_c = $$props.resource) == null ? void 0 : _c.spec) == null ? void 0 : _d.taints) == null ? void 0 : _e.length) ?? 0;
  });
  var span = root();
  let classes;
  var text = $.child(span);
  $.reset(span);
  $.template_effect(() => {
    classes = $.set_class(span, 1, "inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium border", null, classes, {
      "text-destructive": $.get(count) > 0,
      "border-destructive": $.get(count) > 0,
      "bg-destructive": $.get(count) > 0,
      "text-muted": $.get(count) === 0,
      "border-border": $.get(count) === 0
    });
    $.set_text(text, `${$.get(count) ?? ""} ${$.get(count) === 1 ? "taint" : "taints"}`);
  });
  $.append($$anchor, span);
  $.pop();
}
export {
  NodeTaintBadge as default
};
//# sourceMappingURL=NodeTaintBadge.js.map
