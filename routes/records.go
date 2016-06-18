package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/elos/data"
	"github.com/elos/data/transfer"
	"github.com/elos/gaia/services"
	"github.com/elos/metis"
	"github.com/elos/models"
	"github.com/elos/models/access"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

// --- /records/ {{{

// --- const recordsTemplateRaw {{{
const recordsTemplateRaw = `
<html>
	<body>
		<form method="get">
			<fieldset>
				<legend>Query:</legend>
				<label for="kind">Kind:</label>
				<input type="text"   name="kind"  placeholder="string"  {{with .Kind}} value={{. | printf "%q"}} {{end}}/>
				<label for="limit">Limit:</label>
				<input type="number" name="limit" placeholder="int" {{with .Limit}} value={{.}} {{end}}/>
				<label for="batch">Batch:</label>
				<input type="number" name="batch" placeholder="int" {{with .Batch}} value={{.}} {{end}}/>
				<label for="skip">Skip:</label>
				<input type="number" name="skip"  placeholder="int"  {{with .Skip}} value={{.}} {{end}}/>
				<input type="submit" />
			</fieldset>
		</form>
		{{with .Flash}}
			{{. | printf "%s"}}
		{{end}}

		{{ if .Model }}
			{{ $traits := .Model.Traits }}
			<table>
				<thead>
					<tr>
					{{range $traits}}
						<th>{{.Name}}</th>
					{{end}}
					</tr>
				</thead>
				<tbody>
					{{range $record := .Records}}
						<tr>
						{{range $traits}}
						<td>
							{{ index $record .Name }}
						</td>
						{{else}}
							<td>No Traits</td>
						{{end}}
						</tr>
					{{else}}
						<tr><td>No records</td></tr>
					{{end}}
				</tbody>
			</table>
		{{else}}
		No model
		{{end}}
	</body>
</html>
`

// --- }}}

var recordsTemplate = template.Must(template.New("records").Parse(recordsTemplateRaw))

type records struct {
	Kind               data.Kind
	Batch, Limit, Skip int
	Flash              string
	Model              *metis.Model
	Records            []map[string]interface{}
}

func RecordsGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	l := logger.WithPrefix("RecordsGET: ")
	s, err := recordsGet(ctx, r, db, logger)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := recordsTemplate.Execute(w, s); err != nil {
		l.Fatal(err)
	}
}

func recordsGet(context context.Context, r *http.Request, db data.DB, logger services.Logger) (*records, error) {
	l := logger.WithPrefix("recordsGet: ")

	s := new(records)

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
	s.Kind = kind

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

// --- }}}

// --- /records/new/ {{{

// --- const recordsNewTemplateRaw {{{
const recordsNewTemplateRaw = `
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

var recordsNewTemplate = template.Must(template.New("recordsNew").Parse(recordsNewTemplateRaw))

type recordsNew struct {
	Flash  string
	Models map[data.Kind]*metis.Model
}

func RecordsNewGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	l := logger.WithPrefix("RecordsNewGET: ")

	rn := &recordsNew{
		Models: make(map[data.Kind]*metis.Model),
	}

	for kind, model := range models.Metis {
		if _, ok := models.ModelFor(kind).(access.Property); ok {
			rn.Models[kind] = model
		}
	}

	if err := recordsNewTemplate.Execute(w, rn); err != nil {
		l.Fatal(err)
	}
}

func RecordsNewPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
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

	re := &recordsEdit{
		Flash:  "You have not created the record yet, you must save",
		Model:  models.Metis[kind],
		Record: make(map[string]interface{}),
	}

	transfer.TransferAttrs(models.ModelFor(kind), &re.Record)

	if err := recordsEditTemplate.Execute(w, re); err != nil {
		l.Fatal(err)
	}
}

// --- }}}

// --- /records/edit/ {{{

// --- const recordsEditTemplateRaw {{{
const recordsEditTemplateRaw = `
<html>
	<body>
	{{with .Flash}}
		{{.}}
	{{end}}

	{{$record := .Record}}
	{{$model := .Model}}

	EXP:
	{{with .Model}}
	<form method="post">
		<table>
		{{range .Traits}}
		<tr>
		<td> <label for="{{.Name}}"> {{.Name}} </label> </td>
		<td> <input type="text" value="{{ index $record .Name}}" /> </td>
		</tr>
		{{end}}
		</table>
		<input type="submit" value="Save">
	</form>
	{{end}}
	<br />
	{{with .JSON}}
	<form method="post" action="/record/?kind={{$model.Kind}}&id={{index $record "id"}}">
		<input type="text" value="{{.}}" />
		<input type="submit" value="Save" />
	</form>
	{{else}}
		No JSON
	{{end}}
	</body>
</html>
`

// --- }}}

var recordsEditTemplate = template.Must(template.New("recordsEdit").Parse(recordsEditTemplateRaw))

type recordsEdit struct {
	Flash  string
	Model  *metis.Model
	Record map[string]interface{}
	JSON   string
}

func RecordsEditGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	l := logger.WithPrefix("RecordsEditGET: ")

	re := &recordsEdit{
		Record: make(map[string]interface{}),
	}

	// Parse the form value
	if err := r.ParseForm(); err != nil {
		l.Printf("error parsing form: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Secure the kind parameter's existence, and superficial validity (i.e., non-empty)
	k := r.FormValue(kindParam)
	if k == "" {
		l.Printf("no kind parameter")
		http.Error(w, fmt.Sprintf("You must specify a '%s' parameter", kindParam), http.StatusBadRequest)
		return
	}
	kind := data.Kind(k)

	// Secure the id parameter's existence, and superficial validity (i.e., non-empty)
	i := r.FormValue(idParam)
	if i == "" {
		l.Printf("no id parameter")
		http.Error(w, fmt.Sprintf("You must specify a '%s' parameter", idParam), http.StatusBadRequest)
		return
	}

	// Ensure the kind is recognized
	if _, ok := models.Kinds[kind]; !ok {
		l.Printf("unrecognized kind: %q", kind)
		http.Error(w, fmt.Sprintf("The kind %q is not recognized", kind), http.StatusBadRequest)
		return
	}

	// Ensure the id is valid
	id, err := db.ParseID(i)
	if err != nil {
		l.Printf("unrecognized id: %q, err: %s", i, err)
		http.Error(w, fmt.Sprintf("The id %q is invalid", i), http.StatusBadRequest)
		return
	}

	m := models.ModelFor(kind)
	m.SetID(id)

	if err := db.PopulateByID(m); err != nil {
		switch err {
		// ErrAccessDenial and ErrNotFound are "normal" courses, in the sense that they
		// may be expected in normal usage.
		case data.ErrAccessDenial:
			fallthrough // don't leak information, make it look like a 404
		case data.ErrNotFound:
			// This is, by far, the most common error case here.
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		// ErrNoConnection, ErrInvalidID and the under-determined errors are all non-normal cases
		case data.ErrNoConnection:
			fallthrough
		case data.ErrInvalidID:
			fallthrough
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		l.Printf("db.PopulateByID error: %s", err)
		return // regardless of what the error was, we are bailing
	}

	// Retrieve the user this request was authenticated as
	u, ok := user.FromContext(ctx)
	if !ok { // This is certainly an issue, and should _never_ happen
		l.Print("failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Now we impose the system access control, beyond the database access control
	// TODO: limit the domain of errors CanRead returns
	if allowed, err := access.CanRead(db, u, m); err != nil {
		switch err {
		// Again, though odd, both of these are arguably expected
		case data.ErrAccessDenial:
			fallthrough // don't leak information
		case data.ErrNotFound:
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		// These are not-expected
		case data.ErrNoConnection:
			fallthrough
		case data.ErrInvalidID:
			fallthrough
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		l.Printf("access.CanRead error: %s", err)
		return
	} else if !allowed {
		// If you can't read the record you are asking for,
		// it "does not exist" as far as you are concerned
		l.Print("access denied")
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	re.Model = models.Metis[kind]
	transfer.TransferAttrs(m, &re.Record)
	bytes, err := json.MarshalIndent(m, "", "	")
	if err != nil {
		l.Printf("error while marshalling json %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	re.JSON = string(bytes)

	if err := recordsEditTemplate.Execute(w, re); err != nil {
		l.Fatal(err)
	}
}

// --- }}}
