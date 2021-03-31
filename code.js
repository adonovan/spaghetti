
var packages = null // array of packages.Package JSON objects
var path = null // path from root to selected package (elements are indices in 'packages')
var broken = null // array of 2-arrays of int, the node ids of broken edges.

function onLoad() {
    // Grab data from server: package graph, "directory" tree, broken edges.
    // TODO(adonovan): opt: put JSON data in the /index.html page?
    // There's no need for a second HTTP request.
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
	$('#json').text("")
	$('#pkgname').text("none")
	$('#doc').text("")
	$('#imports').text("")
	$('#path').text("")
	return
    }

    // A package node was selected.
    var pkg = packages[json.Package]

    // Show JSON, for debugging.   
    $('#json').html("<code>" + JSON.stringify(json) + "</code>")

    // Show selected package.
    $('#pkgname').text(pkg.PkgPath)

    // Set link to package documentation.
    $('#doc').html("<a target='_blank' href='https://pkg.go.dev/" + pkg.PkgPath + "'>doc</a>")
    
    // TODO(adonovan): display imports as a set of links,
    // with as ImportPath text and "select dir tree node" as action.
    $('#imports').text(json.Imports.join(" "))
    
    // Show "break edges" buttons.
    var html = ""
    var path = [].concat(json.Path).reverse() // from root to selected package
    for (var i in path) {
	var p = packages[path[i]]
	if (i >= 0) {
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
