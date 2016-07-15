package records

import (
	"net/http"
	"text/template"

	"github.com/elos/data"
	"github.com/elos/gaia/services"
	"github.com/elos/metis"
	"github.com/elos/models"
	"golang.org/x/net/context"
)

const newTemplateRaw = `
<html>
	<body>
		{{ with .Flash -}}
			{{ . }}
		{{- end }}
		<table>
		<tbody>
			{{ range .Models }}
			<tr>
			<td>
				<form id="{{ .Kind }}" method="get" action="/records/create/">
					<input type="text" name="kind" value="{{ .Kind }}" /> </td>
				</form>
			<td> <button type="submit" form="{{ .Kind }}"> New </button> </td>
			</tr>
			</form>
			{{ end }}
		</tbody>
		</table>
	</body>
</html>
`

var NewTemplate = template.Must(template.New("records/new").Parse(newTemplateRaw))

type NewData struct {
	Flash  string
	Models map[data.Kind]*metis.Model
}

// NewGET handles a `GET` request to the `/records/new/` route of the records web UI.
//
// Parameters: {}
//
// NewGET supplies an index of kinds of records that can be created.
//
// Success:
//		* StatusOK
//			- html page with form to create the possible kinds
//
// Errors:
//		* StatusInternalServerError
//			- NewTemplate.Execute error
func NewGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	if err := NewTemplate.Execute(w, &NewData{
		Models: models.Metis,
	}); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
