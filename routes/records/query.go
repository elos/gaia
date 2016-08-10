package records

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/elos/data"
	"github.com/elos/data/transfer"
	"github.com/elos/gaia/services"
	"github.com/elos/metis"
	"github.com/elos/models"
	"github.com/elos/models/access"
	"github.com/elos/x/auth"
	"github.com/elos/x/records"
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
		<form method="get">
			{{ .QueryFormHTML }}
			<input type="submit" value="Query">
		</form>
		{{ with .Flash }} {{- . -}} {{ end }}
		{{ $model := .Model }}
		{{ if $model }}
			{{ $traits := $model.Traits }}
			<table>
				<thead>
					<tr>
					{{ range $traits }}
						<th>{{ .Name }}</th>
					{{ end }}
					<th> Edit </th>
					</tr>
				</thead>
				<tbody>
					{{ range $record := .Records }}
						<tr>
						{{ range $traits }}
						<td> {{ index $record .Name }} </td>
						{{ else }}
						<td>No Traits</td>
						{{ end }}
						<td> <a href="/records/edit/?kind={{ $model.Kind }}&id={{ index $record "id" }}"> Edit </a> </td>
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
	Order  []string
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
//			query/Order  []string               (optional)
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
func QueryGET(ctx context.Context, w http.ResponseWriter, r *http.Request, webui services.WebUIClient) {
	pu, pr := auth.CredentialsFromRequest(r) // parses form

	defer r.Body.Close()
	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp, err := webui.QueryGET(ctx, &records.QueryGETRequest{
		Public:  pu,
		Private: pr,
		Url:     r.URL.String(),
		Body:    bytes,
	})

	if err != nil {
		log.Printf("webui.QueryPOST error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp.ServeHTTP(w, r)
}

// execute retrieves the records matching a *query, which the user can read, and marshals them to attr maps.
//
// Errors:
//		- db.Query error
//		- access.CanRead error
//		- iter.Close error
func execute(db data.DB, u *models.User, q *query) ([]map[string]interface{}, error) {
	rs := make([]map[string]interface{}, 0)

	iter, err := db.Query(q.Kind).Limit(q.Limit).Batch(q.Batch).Skip(q.Skip).Select(q.Select).Order(q.Order...).Execute()
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
