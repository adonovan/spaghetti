package main

// -- dominance --

// Code plundered from go/ssa/dom.go package.
// TODO(adonovan): turn it into a generic dominance package abstracted from representation.

// Dominator tree construction ----------------------------------------
//
// We use the algorithm described in Lengauer & Tarjan. 1979.  A fast
// algorithm for finding dominators in a flowgraph.
// http://doi.acm.org/10.1145/357062.357071
//
// We also apply the optimizations to SLT described in Georgiadis et
// al, Finding Dominators in Practice, JGAA 2006,
// http://jgaa.info/accepted/2006/GeorgiadisTarjanWerneck2006.10.1.pdf
// to avoid the need for buckets of size > 1.

// Idom returns the block that immediately dominates b:
// its parent in the dominator tree, if any.
// Root nodes have no parent.
func (b *node) Idom() *node { return b.dom.idom }

// Dominees returns the list of blocks that b immediately dominates:
// its children in the dominator tree.
func (b *node) Dominees() []*node { return b.dom.children }

// Dominates reports whether b dominates c.
func (b *node) Dominates(c *node) bool {
	return b.dom.pre <= c.dom.pre && c.dom.post <= b.dom.post
}

// domInfo contains a node's dominance information.
type domInfo struct {
	idom      *node   // immediate dominator (parent in domtree)
	children  []*node // nodes immediately dominated by this one
	pre, post int32   // pre- and post-order numbering within domtree
	index     int32   // preorder index within reachable nodes; see "reachable hack"
}

// ltState holds the working state for Lengauer-Tarjan algorithm
// (during which domInfo.pre is repurposed for CFG DFS preorder number).
type ltState struct {
	// Each slice is indexed by domInfo.index.
	sdom     []*node // b's semidominator
	parent   []*node // b's parent in DFS traversal of CFG
	ancestor []*node // b's ancestor with least sdom
}

// dfs implements the depth-first search part of the LT algorithm.
func (lt *ltState) dfs(v *node, i int32, preorder []*node) int32 {
	preorder[i] = v
	v.dom.pre = i // For now: DFS preorder of spanning tree of CFG
	i++
	lt.sdom[v.dom.index] = v
	lt.link(nil, v)
	for _, w := range v.imports {
		if lt.sdom[w.dom.index] == nil {
			lt.parent[w.dom.index] = v
			i = lt.dfs(w, i, preorder)
		}
	}
	return i
}

// eval implements the EVAL part of the LT algorithm.
func (lt *ltState) eval(v *node) *node {
	// TODO(adonovan): opt: do path compression per simple LT.
	u := v
	for ; lt.ancestor[v.dom.index] != nil; v = lt.ancestor[v.dom.index] {
		if lt.sdom[v.dom.index].dom.pre < lt.sdom[u.dom.index].dom.pre {
			u = v
		}
	}
	return u
}

// link implements the LINK part of the LT algorithm.
func (lt *ltState) link(v, w *node) {
	lt.ancestor[w.dom.index] = v
}

// buildDomTree computes the dominator tree of f using the LT algorithm,
// starting from the roots indicated by node.isroot.
func buildDomTree(nodes []*node) {
	// The step numbers refer to the original LT paper; the
	// reordering is due to Georgiadis.

	// Clear any previous domInfo.
	for _, b := range nodes {
		b.dom = domInfo{index: -1}
	}

	// The original implementation had the precondition
	// that all nodes are reachable.
	// Because of broken edges, some nodes may be unreachable.
	// Filter them out now with another DFS.
	// The domInfo.idx node is relative this ordering;
	// see other "reachable hack" comments.
	// TODO: clean this up.
	var reachable []*node
	var visit func(n *node)
	visit = func(n *node) {
		if n.dom.index < 0 {
			n.dom.index = int32(len(reachable))
			reachable = append(reachable, n)
			for _, imp := range n.imports {
				visit(imp)
			}
		}
	}
	for _, n := range nodes {
		if n.isroot {
			visit(n)
		}
	}
	nodes = reachable

	n := len(nodes)
	// Allocate space for 5 contiguous [n]*node arrays:
	// sdom, parent, ancestor, preorder, buckets.
	space := make([]*node, 5*n)
	lt := ltState{
		sdom:     space[0:n],
		parent:   space[n : 2*n],
		ancestor: space[2*n : 3*n],
	}

	// Step 1.  Number vertices by depth-first preorder.
	preorder := space[3*n : 4*n]
	var prenum int32
	for _, w := range nodes {
		if w.isroot {
			prenum = lt.dfs(w, prenum, preorder)
		}
	}

	buckets := space[4*n : 5*n]
	copy(buckets, preorder)

	// In reverse preorder...
	for i := int32(n) - 1; i > 0; i-- {
		w := preorder[i]

		// Step 3. Implicitly define the immediate dominator of each node.
		for v := buckets[i]; v != w; v = buckets[v.dom.pre] {
			u := lt.eval(v)
			if lt.sdom[u.dom.index].dom.pre < i {
				v.dom.idom = u
			} else {
				v.dom.idom = w
			}
		}

		// Step 2. Compute the semidominators of all nodes.
		lt.sdom[w.dom.index] = lt.parent[w.dom.index]
		for _, v := range w.importedBy {
			if v.dom.index < 0 {
				continue // see "reachable hack"
			}
			u := lt.eval(v)
			if lt.sdom[u.dom.index].dom.pre < lt.sdom[w.dom.index].dom.pre {
				lt.sdom[w.dom.index] = lt.sdom[u.dom.index]
			}
		}

		lt.link(lt.parent[w.dom.index], w)

		if lt.parent[w.dom.index] == lt.sdom[w.dom.index] {
			w.dom.idom = lt.parent[w.dom.index]
		} else {
			buckets[i] = buckets[lt.sdom[w.dom.index].dom.pre]
			buckets[lt.sdom[w.dom.index].dom.pre] = w
		}
	}

	// The final 'Step 3' is now outside the loop.
	for v := buckets[0]; v != preorder[0]; v = buckets[v.dom.pre] {
		v.dom.idom = preorder[0]
	}

	// Step 4. Explicitly define the immediate dominator of each
	// node, in preorder.
	for _, w := range preorder[1:] {
		if w.isroot {
			w.dom.idom = nil
		} else {
			if w.dom.idom != lt.sdom[w.dom.index] {
				w.dom.idom = w.dom.idom.dom.idom
			}
			// Calculate Children relation as inverse of Idom.
			w.dom.idom.dom.children = append(w.dom.idom.dom.children, w)
		}
	}

	var pre, post int32
	for _, w := range nodes {
		if w.isroot {
			pre, post = numberDomTree(w, pre, post)
		}
	}
}

// numberDomTree sets the pre- and post-order numbers of a depth-first
// traversal of the dominator tree rooted at v.  These are used to
// answer dominance queries in constant time.
//
func numberDomTree(v *node, pre, post int32) (int32, int32) {
	v.dom.pre = pre
	pre++
	for _, child := range v.dom.children {
		pre, post = numberDomTree(child, pre, post)
	}
	v.dom.post = post
	post++
	return pre, post
}
