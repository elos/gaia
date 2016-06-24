package records

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/elos/data"
	"github.com/elos/data/transfer"
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
				<form id="{{.Kind}}" method="post">
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
	l := logger.WithPrefix("RecordsNewGET: ")

	rn := &NewData{
		Models: make(map[data.Kind]*metis.Model),
	}

	for kind, model := range models.Metis {
		if _, ok := models.ModelFor(kind).(access.Property); ok {
			rn.Models[kind] = model
		}
	}

	if err := NewTemplate.Execute(w, rn); err != nil {
		l.Fatal(err)
	}
}

func NewPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	l := logger.WithPrefix("RecordsNewPOST: ")

	k := r.FormValue(kindParam)
	if k == "" {
		l.Printf("no kind parameter")
		http.Error(w, fmt.Sprintf("You must specify a '%s' parameter", kindParam), http.StatusBadRequest)
		return
	}
	kind := data.Kind(k)
	// Ensure the kind is recognized
	if _, ok := models.Kinds[kind]; !ok {
		l.Printf("unrecognized kind: %q", kind)
		http.Error(w, fmt.Sprintf("The kind %q is not recognized", kind), http.StatusBadRequest)
		return
	}

	re := &EditData{
		Flash:  "You have not created the record yet, you must save",
		Model:  models.Metis[kind],
		Record: make(map[string]interface{}),
	}

	transfer.TransferAttrs(models.ModelFor(kind), &re.Record)

	bytes, err := json.MarshalIndent(re.Record, "", "	")
	if err != nil {
		l.Printf("error while marshalling json %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	re.JSON = string(bytes)

	if err := EditTemplate.Execute(w, re); err != nil {
		l.Fatal(err)
	}
}
