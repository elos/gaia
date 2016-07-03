package records

import (
	"net/http"
	"text/template"

	"github.com/elos/data"
	"github.com/elos/gaia/services"
	"github.com/elos/metis"
	"github.com/elos/models"
	"github.com/elos/models/access"
	"golang.org/x/net/context"
)

// --- const newTemplateRaw {{{
const newTemplateRaw = `
<html>
	<body>
		{{with .Flash}}
			{{.}}
		{{end}}
		<table>
		<tbody>
			{{range .Models}}
			<tr>
			<td>
				<form id="{{.Kind}}" method="get" action="/records/create/">
					<input type="text" name="kind" value="{{.Kind}}" /> </td>
				</form>
			<td> <button type="submit" form="{{.Kind}}"> New </button> </td>
			</tr>
			</form>
			{{end}}
		</tbody>
		</table>
	</body>
</html>
`

// --- }}}

var NewTemplate = template.Must(template.New("records/new").Parse(newTemplateRaw))

type NewData struct {
	Flash  string
	Models map[data.Kind]*metis.Model
}

func NewGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	rn := &NewData{
		Models: make(map[data.Kind]*metis.Model),
	}

	for kind, model := range models.Metis {
		if _, ok := models.ModelFor(kind).(access.Property); ok {
			rn.Models[kind] = model
		}
	}

	if err := NewTemplate.Execute(w, rn); err != nil {
		logger.Fatalf("NewTemplate.Execute(w, rn) error: %v", err)
	}
}
