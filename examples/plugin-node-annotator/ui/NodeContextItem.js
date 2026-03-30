import "./disclose-version-DKem_M_z.js";
import * as $ from "svelte/internal/client";
var root = $.from_html(`<button class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover transition-colors">Copy node name</button>`);
function NodeContextItem($$anchor, $$props) {
  $.push($$props, true);
  function copyName() {
    var _a, _b, _c;
    navigator.clipboard.writeText(((_b = (_a = $$props.resource) == null ? void 0 : _a.metadata) == null ? void 0 : _b.name) ?? "");
    (_c = $$props.onclose) == null ? void 0 : _c.call($$props);
  }
  var button = root();
  $.delegated("click", button, copyName);
  $.append($$anchor, button);
  $.pop();
}
$.delegate(["click"]);
export {
  NodeContextItem as default
};
//# sourceMappingURL=NodeContextItem.js.map
