package records

import (
	"log"
	"net/http"
	"strings"
	"text/template"

	"github.com/elos/gaia/services"
	"github.com/elos/x/auth"
	"github.com/elos/x/models"
	"github.com/elos/x/records"
	"golang.org/x/net/context"
)

const viewTemplateRaw = `
<html>
	<body>
		{{ with .Flash }} {{ . }} {{ end }}
		{{ if .Record }}
		{{ with .Kind }} This is a "{{ . }}" {{ end }}
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
		{{ if and .Kind .ID }}
			<a href="/records/edit/?kind={{ .Kind }}&id={{ .ID }}"> Edit </a>
		{{ end }}
	</body>
</html>`

var ViewTemplate = template.Must(template.New("records/view").Parse(viewTemplateRaw))

type ViewData struct {
	Flash, Kind, ID string
	Record          map[string]interface{}
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
func ViewGET(ctx context.Context, w http.ResponseWriter, r *http.Request, webui services.WebUIClient) {
	pu, pr := auth.CredentialsFromRequest(r)

	resp, err := webui.ViewGET(ctx, &records.ViewGETRequest{
		Public:  pu,
		Private: pr,
		Kind:    models.Kind(models.Kind_value[strings.ToUpper(r.FormValue("kind"))]),
		Id:      r.FormValue("id"),
	})

	if err != nil {
		log.Printf("webui.ViewGET error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp.ServeHTTP(w, r)
}
