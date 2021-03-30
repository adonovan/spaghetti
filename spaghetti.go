// The Spaghetti command runs a local web-server that provides an
// interactive single-user tool for visualizing the package
// dependencies of a Go program with a view to refactoring.
//
// Usage: spaghetti [package...]
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"path"
	"sort"
	"strconv"

	"golang.org/x/tools/go/packages"
)

// TODO:
// - select the root nodes initially
// - need more rigor with IDs:
//  - packages.Package.ID is unique but not so user friendly.
//   - import path is more intuitive.
//   - DOM nodes are not hygienic.
//   - std packages should be placed in a subtree.
//   - PkgPaths are not unique due to versions
// - audit for security
// - test with multiple module versions

func main() {
	log.SetPrefix("spaghetti: ")
	log.SetFlags(0)
	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Fatalf("need package arguments")
	}

	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedModule | packages.NeedFiles,
		// TODO Test: true,
	}
	initial, err := packages.Load(config, flag.Args()...)
	if err != nil {
		log.Fatal(err)
	}

	// Create nodes in deterministic preorder.
	// Node numbering determines search results, and we want stability.
	nodes := make(map[string]*node) // map from Package.ID
	packages.Visit(initial, func(pkg *packages.Package) bool {
		n := &node{Package: pkg, index: len(allnodes)}
		if pkg.Module != nil {
			n.modpath = pkg.Module.Path
			n.modversion = pkg.Module.Version
		} else {
			n.modpath = "std"
			n.modversion = "" // TODO: use Go version?
		}
		allnodes = append(allnodes, n)
		nodes[pkg.ID] = n
		return true
	}, nil)
	for _, pkg := range initial {
		nodes[pkg.ID].isroot = true
	}

	// Create edges, in arbitrary order.
	var makeEdges func(n *node)
	makeEdges = func(n *node) {
		for _, imp := range n.Imports {
			n2 := nodes[imp.ID]
			n2.importedBy = append(n2.importedBy, n)
			n.imports = append(n.imports, n2)
		}
	}
	for _, n := range allnodes {
		makeEdges(n)
	}

	recompute()

	http.Handle("/data", http.HandlerFunc(onData))
	http.Handle("/break", http.HandlerFunc(onBreakEdge))
	http.Handle("/unbreak", http.HandlerFunc(onUnbreakEdge))
	http.Handle("/", http.FileServer(http.FS(content)))

	const addr = "localhost:18080"
	log.Printf("Listening on %s...", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

// Globals shared with HTTP handlers.
// The graph. Populated by main and then updated by 'remove' calls.
// Only some fields of node are modified.
var (
	allnodes []*node    // all package nodes, in packages.Visit order
	rootdir  *dirent    // root of module/package "directory" tree
	broken   [][2]*node // broken edges
)

// recompute redoes all the graph algorithms each time the graph is updated.
func recompute() {
	// Sort edges in numeric order of the adjacent node.
	for _, n := range allnodes {
		sortNodes(n.imports)
		sortNodes(n.importedBy)
		n.from = nil
	}

	// Record the path to every node from the root.
	// The path is arbitrary but determined by edge sort order.
	// Visit nodes and record the first search path from a root node.
	var setPath func(n, from *node)
	setPath = func(n, from *node) {
		if n.from == nil {
			n.from = from
			for _, imp := range n.imports {
				setPath(imp, n)
			}
		}
	}
	for _, n := range allnodes {
		if n.isroot {
			setPath(n, nil)
		}
	}

	// Compute dominator tree.
	buildDomTree(allnodes)

	// Compute node weights, using network flow.
	var weight func(*node) int
	weight = func(n *node) int {
		if n.weight == 0 {
			w := 1 + len(n.GoFiles)
			for _, n2 := range n.imports {
				w += weight(n2) / len(n2.importedBy)
			}
			n.weight = w
		}
		return n.weight
	}
	for _, n := range allnodes {
		weight(n)
	}

	// Create tree of reachable modules/packages.
	rootdir = new(dirent)
	for _, n := range allnodes {
		if n.isroot || n.from != nil {
			getDirent(n.ID, n.modpath, n.modversion).node = n
		}
	}
}

//go:embed index.html style.css code.js
var content embed.FS

type node struct {
	// These fields are immutable.
	*packages.Package
	index               int // in allnodes numbering
	isroot              bool
	modpath, modversion string // module, or ("std", "") for standard packages

	// These fields are recomputed after a graph change.
	imports, importedBy []*node // graph edges
	weight              int     // weight computed by network flow
	from                *node   // next link in path from a root node (nil if root)
	dom                 domInfo
}

func sortNodes(nodes []*node) {
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].index < nodes[j].index })
}

// Three different graphs: directory tree, dependency DAG, dominator tree.

// A dirent is an entry in the package directory tree.
type dirent struct {
	name     string  // slash-separated path name (displayed in tree for non-package dirs)
	node     *node   // may be nil
	parent   *dirent // nil for rootdir
	children map[string]*dirent
}

// id returns the entry's DOM element ID in the jsTree.
func (e *dirent) id() string {
	if e.node != nil {
		// package "directory"
		return fmt.Sprintf("node%d", e.node.index)
	} else if e.parent == nil {
		// top-level "directory"
		return "#"
	} else {
		// non-package "directory"
		return fmt.Sprintf("dir%p", e)
	}
}

// getDirent returns the dirent for a given slash-separated path.
// TODO explain module behavior.
func getDirent(name, modpath, modversion string) *dirent {
	var s string
	var parent *dirent
	if name == modpath {
		// modules are top-level "directories" (child of root)
		parent = rootdir
		s = modpath
		if modversion != "" {
			s += "@" + modversion
		}
		name = s
	} else {
		dir, base := path.Dir(name), path.Base(name)
		if dir == "." {
			dir, base = modpath, name // e.g. "std"
		}
		parent = getDirent(dir, modpath, modversion)
		s = base
	}

	e := parent.children[s]
	if e == nil {
		e = &dirent{name: name, parent: parent}
		if parent.children == nil {
			parent.children = make(map[string]*dirent)
		}
		parent.children[s] = e
	}
	return e
}

// handleData emits the initial JSON data: the directory tree of
// packages (in jsTree form) and the list of packages.
// (This is overkill. The graph is so small we don't need AJAX callbacks.)
func onData(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type treeitem struct {
		// These three fields are used by jsTree
		ID     string `json:"id"` // id of DOM element
		Parent string `json:"parent"`
		Text   string `json:"text"`
		Type   string `json:"type"`

		// Any additional fields will be accessible
		// in the jstree node's .original field.
		Package    *packages.Package
		Imports    []string
		Dominators []string
		Path       []int // from package to root, inclusive of endpoints
	}
	var payload struct {
		Tree     []treeitem
		Packages []*packages.Package
		Broken   [][2]int
		Roots    []string
	}

	// graph nodes (packages)
	for _, n := range allnodes {
		payload.Packages = append(payload.Packages, n.Package)

		if n.isroot {
			payload.Roots = append(payload.Roots, n.Package.PkgPath)
		}
	}

	// broken edges
	for _, edge := range broken {
		payload.Broken = append(payload.Broken, [2]int{edge[0].index, edge[1].index})
	}

	// tree nodes (packages, modules, and directories)
	var visit func(children map[string]*dirent)
	visit = func(children map[string]*dirent) {
		var names []string
		for name := range children {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			e := children[name]

			item := treeitem{ID: e.id(), Text: e.name}
			if e.node != nil {
				// package node: show flow weight
				item.Text = fmt.Sprintf("%s (%d)", e.name, e.node.weight)
				item.Type = "pkg" // TODO also "module", "dir"
				// TODO item.State = { 'opened' : true, 'selected' : true }

				item.Package = e.node.Package
				for _, imp := range e.node.imports {
					item.Imports = append(item.Imports, imp.Package.ID)
				}
				for n := e.node.Idom(); n != nil; n = n.Idom() {
					item.Dominators = append(item.Dominators, n.Package.ID)
				}
				for n := e.node; n != nil; n = n.from {
					item.Path = append(item.Path, n.index)
				}
			}

			if e.parent != nil {
				item.Parent = e.parent.id()
			}
			payload.Tree = append(payload.Tree, item)

			visit(e.children)
		}
	}
	visit(rootdir.children)

	data, err := json.Marshal(payload)
	if err != nil {
		log.Fatal(err)
	}
	w.Write(data)
}

func onBreakEdge(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		log.Println(err)
		return // TODO handle gracefully
	}

	// TODO validation
	to, _ := strconv.Atoi(req.Form.Get("to"))
	toNode := allnodes[to]

	all, _ := strconv.ParseBool(req.Form.Get("all"))
	if all {
		// break all incoming edges to toNode.
		for _, fromNode := range toNode.importedBy {
			log.Printf("Break edge %s --> %s", fromNode, toNode)
			broken = append(broken, [2]*node{fromNode, toNode})
			fromNode.imports = remove(fromNode.imports, toNode)
		}
		toNode.importedBy = nil

	} else {
		// break from-->to
		from, _ := strconv.Atoi(req.Form.Get("from"))
		fromNode := allnodes[from]
		log.Printf("Break edge %s --> %s", fromNode, toNode)
		broken = append(broken, [2]*node{fromNode, toNode})
		fromNode.imports = remove(fromNode.imports, toNode)
		toNode.importedBy = remove(toNode.importedBy, fromNode)
	}

	recompute()

	http.Redirect(w, req, "/index.html", http.StatusTemporaryRedirect)
}

func onUnbreakEdge(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		log.Println(err)
		return // TODO handle gracefully
	}

	// TODO validation
	from, _ := strconv.Atoi(req.Form.Get("from"))
	to, _ := strconv.Atoi(req.Form.Get("to"))

	fromNode := allnodes[from]
	toNode := allnodes[to]

	log.Printf("Unbreak edge %s --> %s", fromNode, toNode)

	// Remove from broken edge list.
	o := broken[:0]
	for _, edge := range broken {
		if edge != [2]*node{fromNode, toNode} {
			o = append(o, edge)
		}
	}
	broken = o

	fromNode.imports = append(fromNode.imports, toNode)
	toNode.importedBy = append(toNode.importedBy, fromNode)

	recompute()

	http.Redirect(w, req, "/index.html", http.StatusTemporaryRedirect)
}

// remove destructively removes all occurrences of x from slice, sorts it, and returns it.
// (It is the opposite of 'append'.)
func remove(slice []*node, x *node) []*node {
	for i := 0; i < len(slice); i++ {
		if slice[i] == x {
			last := len(slice) - 1
			slice[i] = slice[last]
			slice = slice[:last]
			i--
		}
	}
	sortNodes(slice)
	return slice
}
