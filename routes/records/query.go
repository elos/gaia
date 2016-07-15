package records

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/elos/data"
	"github.com/elos/data/transfer"
	"github.com/elos/gaia/routes/records/form"
	"github.com/elos/gaia/services"
	"github.com/elos/metis"
	"github.com/elos/models"
	"github.com/elos/models/access"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

const (
	kindParam   = "kind"
	limitParam  = "limit"
	batchParam  = "batch"
	skipParam   = "skip"
	selectParam = "select"
)

const queryTemplateRaw = `
<html>
	<body>
		<form id="queryForm" method="get">
			{{ .QueryFormHTML }}
		</form>
		{{ with .Flash }} {{- . -}} {{ end }}
		{{ if .Model }}
			{{ $traits := .Model.Traits }}
			<table>
				<thead>
					<tr>
					{{ range $traits }}
						<th>{{ .Name }}</th>
					{{ end }}
					</tr>
				</thead>
				<tbody>
					{{ range $record := .Records }}
						<tr>
						{{ range $traits }}
						<td>
							{{ index $record .Name }}
						</td>
						{{ else }}
							<td>No Traits</td>
						{{ end }}
						</tr>
					{{ else }}
						<tr><td>No records</td></tr>
					{{ end }}
				</tbody>
			</table>
		{{ else }}
			No model
		{{ end }}
	</body>
</html>
`

var QueryTemplate = template.Must(template.New("records/query").Parse(queryTemplateRaw))

type QueryData struct {
	Flash         string
	QueryFormHTML template.HTML
	Model         *metis.Model
	Records       []map[string]interface{}
}

type query struct {
	Kind   data.Kind
	Limit  int
	Batch  int
	Skip   int
	Select map[string]interface{}
}

// QueryGET handles a `GET` request to the `/records/query/` route of the records web UI.
//
// Parameters:
//		{
//			query/Kind   string                 (optional)
//			query/Limit  int                    (optional)
//			query/Batch  int                    (optional)
//			query/Skip   int                    (optional)
//			query/Select map[string]interface{} (optional)
//		}
//
// QueryGET first checks for a kind. If it recieves none, it returns a form to submit a query.
// Next it issues a query based on the query structure unmarshaled from the form namespaced
// to query, and then servers a table of the matching records.
//
// Success:
//		* StatusOK
//			- html page with form to issue query (and possibly a table of records matching the present query)
//
// Errors:
//		* StatusBadRequest
//			- unrecognized kind
//		* StatusInternalServerError
//			- r.ParseForm error
//			- form.Unmarshal error
//			- form.Marshal error
//			- QueryTemplate.Execute error (by 3 paths)
//			- ctx missing user
//			- execute error
func QueryGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	q := new(query)
	if err := form.Unmarshal(r.Form, q, "query"); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	formbytes, err := form.Marshal(q, "query")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if q.Kind == data.Kind("") {
		if QueryTemplate.Execute(w, &QueryData{
			Flash:         "Missing kind",
			QueryFormHTML: template.HTML(string(formbytes)),
		}); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	if !models.Kinds[q.Kind] {
		if QueryTemplate.Execute(w, &QueryData{
			Flash:         "Invalid kind",
			QueryFormHTML: template.HTML(string(formbytes)),
		}); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	u, ok := user.FromContext(ctx)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rs, err := execute(db, u, q)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := QueryTemplate.Execute(w, &QueryData{
		Flash:         fmt.Sprintf("%d results.", len(rs)),
		QueryFormHTML: template.HTML(string(formbytes)),
		Model:         models.Metis[q.Kind],
		Records:       rs,
	}); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// execute retrieves the records matching a *query, which the user can read, and marshals them to attr maps.
//
// Errors:
//		- db.Query error
//		- access.CanRead error
//		- iter.Close error
func execute(db data.DB, u *models.User, q *query) ([]map[string]interface{}, error) {
	rs := make([]map[string]interface{}, 0)

	iter, err := db.Query(q.Kind).Limit(q.Limit).Batch(q.Batch).Skip(q.Skip).Select(q.Select).Execute()
	if err != nil {
		return nil, err
	}

	m := models.ModelFor(q.Kind)
	for iter.Next(m) {
		ok, err := access.CanRead(db, u, m)

		if err != nil {
			return nil, err
		}

		if !ok {
			continue
		}

		temp := make(map[string]interface{})
		transfer.TransferAttrs(m, &temp)
		rs = append(rs, temp)
		m = models.ModelFor(q.Kind)
	}

	if err := iter.Close(); err != nil {
		return nil, err
	}

	return rs, nil
}
