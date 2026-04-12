---
name: YAML autocomplete empty-line issue
description: codemirror-json-schema yamlCompletion returns 0 results on empty lines / Ctrl+Space without prefix — needs supplementary completion source
type: project
---

Schema-driven YAML autocomplete is wired up and works for prefix matching (typing a character),
but Ctrl+Space on empty lines returns nothing. Root cause is in codemirror-json-schema v0.8.1
(latest, no fix planned — GitHub #121). Full findings documented in `.wolf/yaml-autocomplete-findings.md`.

**Why:** The library's `getNodeAtPosition` resolves to the document root on empty lines, misrouting
to value completions instead of property completions.

**How to apply:** Next session should read `.wolf/yaml-autocomplete-findings.md` for full context,
then brainstorm and implement a supplementary completion source. Debug instrumentation is still
in `CreateResourceDialog.svelte` (DOM keydown logger + yamlLanguage.data completion logger) —
remove when fix is implemented.
