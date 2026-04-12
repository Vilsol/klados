export interface GVREntry {
  gvr: string;
  kind: string;
}

export interface CRDTreeNode {
  label: string;
  fullSuffix: string;
  directGvrs: GVREntry[];
  children: CRDTreeNode[];
}

interface TrieNode {
  children: Map<string, TrieNode>;
  gvrs: string[];
}

export function extractGroup(gvr: string): string {
  const parts = gvr.split(".");
  const group = parts.slice(0, -2).join(".");
  return group === "core" ? "" : group;
}

export function buildCRDTree(gvrs: string[], getKind: (gvr: string) => string): CRDTreeNode[] {
  const root: TrieNode = {children: new Map(), gvrs: []};

  for (const gvr of gvrs) {
    const group = extractGroup(gvr);
    if (!group) {
      continue;
    }
    const segs = group.split(".").reverse();
    let cur = root;
    for (const seg of segs) {
      if (!cur.children.has(seg)) {
        cur.children.set(seg, {children: new Map(), gvrs: []});
      }
      cur = cur.children.get(seg)!;
    }
    cur.gvrs.push(gvr);
  }

  return enforceMinTopLevel(buildSubtree(root, [], getKind));
}

function buildSubtree(node: TrieNode, parentSegs: string[], getKind: (gvr: string) => string): CRDTreeNode[] {
  return [...node.children.entries()]
    .map(([seg, child]) => {
      const compressedSegs = [seg];
      let cur = child;

      while (cur.gvrs.length === 0 && cur.children.size === 1) {
        const [nextSeg, nextChild] = [...cur.children.entries()][0];
        compressedSegs.push(nextSeg);
        cur = nextChild;
      }

      const label = [...compressedSegs].reverse().join(".");
      const fullSuffix = [...parentSegs, ...compressedSegs].reverse().join(".");

      return {
        label,
        fullSuffix,
        directGvrs: cur.gvrs.map((gvr) => ({gvr, kind: getKind(gvr)})).sort((a, b) => a.kind.localeCompare(b.kind)),
        children: buildSubtree(cur, [...parentSegs, ...compressedSegs], getKind),
      };
    })
    .sort((a, b) => a.label.localeCompare(b.label));
}

function enforceMinTopLevel(nodes: CRDTreeNode[]): CRDTreeNode[] {
  const result: CRDTreeNode[] = [];
  for (const node of nodes) {
    if (!node.label.includes(".")) {
      for (const child of node.children) {
        result.push({...child, label: `${child.label}.${node.label}`});
      }
      if (node.directGvrs.length > 0) {
        result.push(node);
      }
    } else {
      result.push(node);
    }
  }
  return result.sort((a, b) => a.label.localeCompare(b.label));
}
