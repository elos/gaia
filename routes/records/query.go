package records

import (
	"html/template"

	"github.com/elos/data"
	"github.com/elos/data/transfer"
	"github.com/elos/metis"
	"github.com/elos/models"
	"github.com/elos/models/access"
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
