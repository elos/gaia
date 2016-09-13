package form

import "github.com/elos/x/html"

var Script = &html.Script{
	Type: html.Script_JAVASCRIPT,
	Content: `
	window.onload = function () {
		var expandables = document.getElementsByClassName("expandable");
		var i;
		for (i = 0; i < x.length; i++) {
			var expandable = expandables[i];
			var zero = document.getElementById(expandable.id + "/zero");

			// add plus button
			var b = document.createElement("button");
			b.innerHTML = "+";
			b.onClick = function () { alert("clicked"); }
			expandable.appendChild(b);
		}
	}

	`,
}
