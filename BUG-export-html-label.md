# Bug: `export html` puts `label` inside `extra`, Cytoscape can't render nodes

## Summary

`graphize export html` nests the `label` field inside `data.extra` instead of at `data.label`. Cytoscape.js styles reference `data(label)` which only reads top-level `data` properties, so nodes render with no labels and the graph appears empty (header/stats show but no visible nodes).

The `export htmlsite` exporter does NOT have this bug — it correctly places `label` at the top level of `data`.

## Steps to Reproduce

```bash
graphize init
# populate .graphize/nodes/ and .graphize/edges/ with valid graphfs JSON
graphize export html -o graph.html --dark
```

## Expected

Node elements should have `label` at the top level:

```json
{"group": "nodes", "data": {"id": "svc:ecm", "label": "ecm", "type": "service", "extra": {"image": "ecm"}}}
```

## Actual

`label` is nested inside `extra`:

```json
{"group": "nodes", "data": {"id": "svc:ecm", "type": "service", "extra": {"label": "ecm", "image": "ecm"}}}
```

Cytoscape logs warnings for every node:

```
Do not assign mappings to elements without corresponding data
(i.e. ele `svc:ecm` has no mapping for property `label` with data field `label`)
```

## Root Cause

The `export html` command serializes `attrs` from graphfs nodes into a nested `extra` object. The `label` key in `attrs` should be hoisted to the top level of the Cytoscape `data` object, since the HTML template's styles reference `data(label)`.

The `export htmlsite` command already does this correctly — it flattens all attrs into `data` directly.

## Workaround

Post-process the HTML to hoist `label`:

```python
import json, re
with open('graph.html') as f:
    html = f.read()
m = re.search(r'const elements = (\[.*?\]);', html)
elements = json.loads(m.group(1))
for el in elements:
    d = el['data']
    if 'extra' in d and 'label' in d['extra']:
        d['label'] = d['extra']['label']
html = html[:m.start(1)] + json.dumps(elements) + html[m.end(1):]
with open('graph.html', 'w') as f:
    f.write(html)
```

## Fix

The `export html` serializer should match `export htmlsite` behavior: flatten `attrs` into the Cytoscape `data` object, or at minimum hoist `label` to the top level.
