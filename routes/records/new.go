package records

import (
	"log"
	"net/http"
	"text/template"

	"github.com/elos/data"
	"github.com/elos/gaia/services"
	"github.com/elos/metis"
	"github.com/elos/models"
	"github.com/elos/x/auth"
	"github.com/elos/x/records"
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

var instantiableModels = make(map[data.Kind]*metis.Model)

func init() {
	for kind, model := range models.Metis {
		switch kind {
		case models.UserKind:
		default:
			instantiableModels[kind] = model
		}
	}
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
func NewGET(ctx context.Context, w http.ResponseWriter, r *http.Request, webui services.WebUIClient) {
	pu, pr := auth.CredentialsFromRequest(r)

	resp, err := webui.NewGET(ctx, &records.NewGETRequest{
		Public:  pu,
		Private: pr,
	})

	if err != nil {
		log.Printf("webui.NewGET error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp.ServeHTTP(w, r)
}
