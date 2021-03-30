
var packages = null; // array of packages.Package JSON objects
var path = null; // path from root to selected package (elements are indices in 'packages')
var broken = null; // array of 2-arrays of int, the node ids of broken edges.

function onLoad() {
    // TODO send the data in /index.html; no need for AJAX.
    jQuery.ajax({
	url: "/data",
	success: function(json) {
	    packages = json.Packages

	    // Show initial packages.
	    $('#roots').text(json.Roots.join("\n"))

	    // Show broken edges.
	    broken = json.Broken
	    var html = ""
	    for (var i in broken) {
		edge = broken[i]
		html += "<button type='button' onclick='unbreak(" + edge[0] + ", " + edge[1] + ")'>unbreak</button> "
		    + "<code>" + packages[edge[0]].PkgPath + "</code> ⟶ "
		    + "<code>" + packages[edge[1]].PkgPath + "</code><br/>"
	    }
	    $('#broken').html(html)
	    
	    $('#tree').jstree({
		"core" : {
		    "animation" : 0,
		    "check_callback" : true,
		    'data' : json.Tree,
		},
		"types" : {
		    "#" : {
		    },
		    "root" : {
			"icon" : "/static/3.3.11/assets/images/tree_icon.png"
		    },
		    "module": {
			"icon" : "https://jstree.com/static/3.3.11/assets/images/tree_icon.png"
		    },
		    "default": {
		    },
		    "pkg" : {
			"icon": "https://old.jstree.com//static/v.1.0pre/_demo/file.png"
			// "glyphicon glyphicon-file",
		    }
		},
		"plugins" : ["contextmenu", "dnd", "search", "state", "types", "wholerow"],
	    })

	    // Show package info when a node is clicked.
	    $('#tree').on("changed.jstree", function (e, data) {
		if (data.node) {
		    selectPkg(data.node.original);
		}
	    })
	}})
}

// selectPkg shows info about the selected package, or hides it if json.pkg is null
// (for a non-package "directory").
function selectPkg(json) {
    if (json.Package != null) { // TODO: no need to dup the Package JSON: pass an index 
	$('#json').html("<code>" + JSON.stringify(json) + "</code>")
	$('#pkgname').text(json.Package.PkgPath)
	$('#doc').html("<a target='_blank' href='https://pkg.go.dev/" + json.Package.PkgPath + "'>doc</a>")

	// TODO make imports a set of links of ImportPath to graph node?
	if (json.Imports != null) {
	    $('#imports').text(json.Imports.join(" "))
	}

	// Dominators
	var html = "";
	var doms = [].concat(json.Dominators).reverse();
	for (var i in doms) {
	    html += (i > 0 ? " ⟶ " : "") + "<code>" + doms[i] + "</code>"
	}
	$('#dom').html(html)

	// Show "break edges" buttons.
	var html = "";
	var path = [].concat(json.Path).reverse(); // from root to selected package
	for (var i in path) {
	    var pkg = packages[path[i]];
	    if (i == 0) { // root
		html += "<code>" + pkg.PkgPath + "</code><br/>";	
	    } else {
		html += "<button type='button' onclick='breakedge(" + path[i-1] + ", " + path[i] + ", false)'>break</button> "
		    + "<button type='button' onclick='breakedge(" + path[i-1] + ", " + path[i] + ", true)'>break all</button> "
		    + "⟶ <code>" +  pkg.PkgPath + "</code><br/>";
	    }
	}
	$('#path').html(html)

    } else {
	// Non-package "directory" node: grey out the fields.
	$('#json').text("")
	$('#pkgname').text("N/A")
	$('#doc').text("");
	$('#imports').text("");
	$('#dom').text("");
	$('#path').text("");
    }
}

function breakedge(i, j, all) {
    // Must reload the page since the graph has changed.
    document.location = "/break?from=" + i + "&to=" + j + "&all=" + all;
}

function unbreak(i, j) {
    // Must reload the page since the graph has changed.
    document.location = "/unbreak?from=" + i + "&to=" + j;   
}
