package form

import "github.com/elos/x/html"

var Script = &html.Script{
	Type: html.Script_JAVASCRIPT,
	Content: `
    function isDescendentOf(e1, className) {
      var p = e1.parentElement;
      return p !== null && (p.className === className || isDescendentOf(p, className));
    }

    function setupSlicefieldsets() {
      var i, sliceFieldsets = document.getElementsByClassName('slice-fieldset');

      for (i = 0; i < sliceFieldsets.length; i += 1) {
        (function (f) {
          // Skip fieldset if it already has a button
          if (f.children.length && f.children[f.children.length - 1].type === "button") {
            return;
          }

          // Skip fieldset if it is in a zero-value
          if (isDescendentOf(f, "zero-value")) {
            return;
          }

          var button = document.createElement("input");
          button.type = "button";
          button.value = "Add";

          f.appendChild(button);

          button.onclick = function (e) {
            if (e.target === button) {
              var nextFieldLabel = 0;
              for (j = f.children.length - 1; j >= 0; j -= 1) {
                var e = f.children[j];
                if (e.tagName === "LABEL") {
                  nextFieldLabel = parseInt(e.innerHTML, 10) + 1;
                  break;
                } else if (e.tagName === "FIELDSET") {
                  nextFieldLabel = parseInt(e.getElementsByTagName("legend")[0].innerHTML, 10) + 1;
                  break;
                }
              }
              var firstInput = f.getElementsByTagName("input")[0];
              var newInputName = firstInput.name.replace((/\/.+$/), nextFieldLabel);
              var node = f.getElementsByClassName("zero-value")[0].cloneNode(true);
              node.className = "";

              node.innerHTML = node.innerHTML.replace(/%index%/g, nextFieldLabel);

              for (j = 0; j < node.children.length; j += 1) {
                f.insertBefore(node.children[j].cloneNode(true), button);
              }

              f.insertBefore(document.createElement("br"), button);

              setupSlicefieldsets();
            }
          };
        })(sliceFieldsets[i]);
      }
    };

    window.onload = setupSlicefieldsets;
	`,
}
