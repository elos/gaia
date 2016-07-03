package records

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/elos/data"
	"github.com/elos/data/transfer"
	"github.com/elos/gaia/services"
	"github.com/elos/metis"
	"github.com/elos/models"
	"golang.org/x/net/context"
)

const (
	kindParam   = "kind"
	limitParam  = "limit"
	batchParam  = "batch"
	skipParam   = "skip"
	selectParam = "select"
)

// --- queryTemplateRaw {{{

const queryTemplateRaw = `
<html>
	<body>
		<form id="queryForm" method="get">
			<fieldset>
				<legend>Query:</legend>
				<label for="kind">Kind:</label>
				<input type="text"   name="kind"  placeholder="string"  {{ with .Kind -}}  value="{{- . -}}" {{- end -}} />
				<label for="limit">Limit:</label>
				<input type="number" name="limit" placeholder="int"     {{ with .Limit -}} value="{{- . -}}" {{- end -}} />
				<label for="batch">Batch:</label>
				<input type="number" name="batch" placeholder="int"     {{ with .Batch -}} value="{{- . -}}" {{- end -}} />
				<label for="skip">Skip:</label>
				<input type="number" name="skip"  placeholder="int"     {{ with .Skip -}}  value="{{- . -}}" {{- end -}} />
				<label for="select">Select:</label>
				<textarea name="select" form="queryForm"> {{- with .Select -}} {{- . -}} {{- end -}} </textarea>
				<input type="submit" />
			</fieldset>
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

// --- }}}

var QueryTemplate = template.Must(template.New("records/query").Parse(queryTemplateRaw))

type QueryData struct {
	Kind               data.Kind
	Batch, Limit, Skip int
	Select             string
	Flash              string
	Model              *metis.Model
	Records            []map[string]interface{}
}

func QueryGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	l := logger.WithPrefix("RecordsQueryGET: ")
	s, err := Query(ctx, r, db, logger)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := QueryTemplate.Execute(w, s); err != nil {
		l.Fatal(err)
	}
}

func Query(ctx context.Context, r *http.Request, db data.DB, logger services.Logger) (*QueryData, error) {
	l := logger.WithPrefix("recordsGet: ")

	s := new(QueryData)

	// Secure the kind parameter's existence, and superficial validity (i.e., non-empty)
	k := r.FormValue(kindParam)
	if k == "" {
		l.Printf("no kind parameter")
		s.Flash = fmt.Sprintf("You must specify a '%s' parameter", kindParam)
		return s, nil
	}
	kind := data.Kind(k)

	// Ensure the kind is recognized
	if _, ok := models.Kinds[kind]; !ok {
		l.Printf("unrecognized kind: %q", kind)
		s.Flash = fmt.Sprintf("The kind %q is not recognized", kind)
		return s, nil
	}

	s.Kind = kind

	// Retrieve the limit, batch and skip parameters
	lim := r.FormValue(limitParam)
	bat := r.FormValue(batchParam)
	ski := r.FormValue(skipParam)

	// Set up the variables to apply to the query
	var limit, batch, skip int
	if lim != "" {
		limit, _ = strconv.Atoi(lim)
	} else if bat != "" {
		batch, _ = strconv.Atoi(bat)
	} else if ski != "" {
		skip, _ = strconv.Atoi(ski)
	}

	s.Limit = limit
	s.Batch = batch
	s.Skip = skip

	selectString := r.FormValue(selectParam)
	attrs := data.AttrMap{}
	if selectString != "" {
		if err := json.Unmarshal([]byte(selectString), &attrs); err != nil {
			l.Printf("json.Unmarshal(bytes(selectString), &attrs) error: %v", err)
			s.Flash = fmt.Sprintf("Failure unmarshalling json: %v", err)
			return s, nil
		}
	}
	bytes, err := json.MarshalIndent(attrs, "", "	")
	if err != nil {
		l.Printf("json.MarshalIndent error: %v", err)
		s.Flash = fmt.Sprintf("Failure marshalling json: %v", err)
		return s, nil
	}
	s.Select = string(bytes)

	iter, err := db.Query(s.Kind).Limit(limit).Batch(batch).Skip(skip).Execute()
	if err != nil {
		return nil, err
	}
	m := models.ModelFor(s.Kind)
	for iter.Next(m) {
		temp := make(map[string]interface{})
		transfer.TransferAttrs(m, &temp)
		s.Records = append(s.Records, temp)
		m = models.ModelFor(s.Kind)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}

	s.Model = models.Metis[s.Kind]

	return s, nil
}
