package records

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/elos/data"
	"github.com/elos/data/transfer"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/access"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

const viewTemplateRaw = `
<html>
	<body>
		{{ with .Flash }} {{ . }} {{ end }}
		{{ if .Record }}
		{{ with .Kind }} {{ . }} {{ end }}
		<table>
			<thead>
				<tr> <th> Attribute </th> <th> Value </th>
			</thead>
			<tbody>
				{{ range $attr, $val := .Record -}}
				<tr> <td> {{ $attr }} </td> <td> {{ $val }} </td>  </tr>
				{{- end }}
			</tbody>
		</table>
		{{ else }}
			No Model
		{{ end }}
	</body>
</html>`

var ViewTemplate = template.Must(template.New("records/view").Parse(viewTemplateRaw))

type ViewData struct {
	Flash  string
	Kind   data.Kind
	Record map[string]interface{}
}

// ViewGET handles a `GET` request to the `/records/view/` route of the records web UI.
//
// Parameters:
//		{
//			kind string
//			id string
//		}
//
// ViewGET matches against the `kind` and `id` parameters and marshals the record to a table of attributes.
//
// Success:
//		* StatusFound
//			- html page with the record's info
//
// Errors:
//		* StatusBadRequest
//			- missing kind
//			- unrecognized kind
//			- missing id
//			- invalid id
//		* StatusNotFound
//			- db.PopulateByID data.ErrNotFound (still html, with a flash message)
//			- access.CanRead false (still html, with a flash message)
//		* StatusInternalServerError
//			- r.ParseForm error
//			- db.PopulateByID error
//			- ctx missing user
//			- access.CanRead error
//			- transfer.TransferAttrs error
//			- ViewTemplate.Execute error (by 3 paths)
func ViewGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	k := r.FormValue("kind")
	if k == "" {
		http.Error(w, "Missing parameter \"kind\"", http.StatusBadRequest)
		return
	}
	kind := data.Kind(k)

	if !models.Kinds[kind] {
		http.Error(w, fmt.Sprintf("Unrecognized kind: %q", k), http.StatusBadRequest)
		return
	}

	i := r.FormValue("id")
	if i == "" {
		http.Error(w, "Missing parameter \"id\"", http.StatusBadRequest)
		return
	}

	id, err := db.ParseID(i)
	if err != nil {
		http.Error(w, "Invalid \"id\"", http.StatusBadRequest)
		return
	}

	m := models.ModelFor(kind)
	m.SetID(id)
	if err := db.PopulateByID(m); err != nil {
		if err == data.ErrNotFound {
			w.WriteHeader(http.StatusNotFound)
			if err := ViewTemplate.Execute(w, &ViewData{
				Flash: fmt.Sprintf("(%s, %d) was not found", k, i),
			}); err != nil {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			}
			return
		}

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	u, ok := user.FromContext(ctx)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ok, err = access.CanRead(db, u, m)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		if err := ViewTemplate.Execute(w, &ViewData{
			Flash: fmt.Sprintf("(%s, %d) was not found", k, i),
		}); err != nil {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}
	}

	attrMap := make(map[string]interface{})
	if err := transfer.TransferAttrs(m, &attrMap); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := ViewTemplate.Execute(w, &ViewData{
		Kind:   kind,
		Record: attrMap,
	}); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
