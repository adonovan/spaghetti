
var packages = null // array of packages.Package JSON objects
var path = null // path from root to selected package (elements are indices in 'packages')
var broken = null // array of 2-arrays of int, the node ids of broken edges.

function onLoad() {
    // Grab data from server: package graph, "directory" tree, broken edges.
    jQuery.ajax({url: "/data", success: onData})
}

// onData is called shortly after page load with the result of the /data request.
function onData(data) {
    // Save array of Package objects.
    packages = data.Packages

    // Show initial packages.
    $('#initial').text(data.Initial.map(i => packages[i].PkgPath).join("\n"))
    
    // Show broken edges.
    broken = data.Broken
    var html = ""
    for (var i in broken) {
	edge = broken[i]
	html += "<button type='button' onclick='unbreak(" + edge[0] + ", " + edge[1] + ")'>unbreak</button> "
	    + "<code>" + packages[edge[0]].PkgPath + "</code> ⟶ "
	    + "<code>" + packages[edge[1]].PkgPath + "</code><br/>"
    }
    $('#broken').html(html)

    // Populate package/module "directory" tree.
    $('#tree').jstree({
	"core": {
	    "animation": 0,
	    "check_callback": true,
	    'data': data.Tree,
	},
	"types": {
	    "#": {
	    },
	    "root": {
		"icon": "/static/3.3.11/assets/images/tree_icon.png"
	    },
	    "module": {
		"icon": "https://jstree.com/static/3.3.11/assets/images/tree_icon.png"
	    },
	    "default": {
	    },
	    "pkg": {
		"icon": "https://old.jstree.com//static/v.1.0pre/_demo/file.png"
	    }
	},
	"plugins": ["search", "state", "types", "wholerow"],
	"search": {
	    "case_sensitive": false,
	    "show_only_matches": true,
	}
    })

    // Show package info when a node is clicked.
    $('#tree').on("changed.jstree", function (e, data) {
	if (data.node) {
	    selectPkg(data.node.original)
	}
    })

    // Search the tree when the user types in the search box.
    $("#search").keyup(function () {
        var searchString = $(this).val();
        $('#tree').jstree('search', searchString);
    });  
}

// selectPkg shows package info (if any) about the clicked node.
function selectPkg(json) {
    if (json.Package < 0) {
	// Non-package "directory" node: clear the fields.
	$('#pkgname').text("none")
	$('#doc').text("")
	$('#imports').html("")
	$('#path').text("")
	return
    }

    // A package node was selected.
    var pkg = packages[json.Package]

    // Show selected package.
    $('#pkgname').text(pkg.PkgPath)

    // Set link to Go package documentation.
    $('#doc').html("<a title='doc' target='_blank' href='https://pkg.go.dev/" + pkg.PkgPath + "'><img src='https://pkg.go.dev/favicon.ico' width='16' height='16'/></a>")

    // Show imports in a drop-down menu.
    // Selecting an import acts like clicking on that package in the tree.
    var imports = $('#imports')
    imports.html("")
    var option = document.createElement("option")
    option.textContent = "..."
    option.value = "-1"
    imports.append(option)
    for (var i in json.Imports) {
	var imp = json.Imports[i]
	option = document.createElement("option")
	option.textContent = packages[imp].PkgPath
	option.value = "" + imp // package index, used by onSelectImport
	imports.append(option)
    }
    
    // Show "break edges" buttons.
    var html = ""
    var path = [].concat(json.Path).reverse() // from root to selected package
    for (var i in path) {
	var p = packages[path[i]]
	if (i > 0) {
	    html += "<button type='button' onclick='breakedge(" + path[i-1] + ", " + path[i] + ", false)'>break</button> "
		+ "<button type='button' onclick='breakedge(" + path[i-1] + ", " + path[i] + ", true)'>break all</button> "
		+ "⟶ "
	}
	html += "<code class='" + (json.Dominators.includes(path[i]) ? "dom" : "") + "'>" + p.PkgPath + "</code><br/>"
    }
    $('#path').html(html)
}

function breakedge(i, j, all) {
    // Must reload the page since the graph has changed.
    document.location = "/break?from=" + i + "&to=" + j + "&all=" + all
}

function unbreak(i, j) {
    // Must reload the page since the graph has changed.
    document.location = "/unbreak?from=" + i + "&to=" + j
}

// onSelectImport is called by the imports dropdown.
function onSelectImport(sel) {
    if (sel.value >= 0) {
	// Simulate a click on a tree node corresponding to the selected import.
	$('#tree').jstree('select_node', 'node' + sel.value);
    }
}
