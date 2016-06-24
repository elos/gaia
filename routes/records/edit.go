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
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

const (
	idParam = "id"
)

const editTemplateRaw = `
<html>
	<body>
	{{with .Flash}}
		{{.}}
	{{end}}

	{{$record := .Record}}
	{{$model := .Model}}

	{{with .JSON}}
	<form method="post" action="/record/?kind={{$model.Kind}}&id={{index $record "id"}}" id="recordJSON">
		<textarea name="record" form="recordJSON" >{{.}}</textarea>
		<input type="submit" value="Save" />
	</form>
	{{else}}
		No JSON
	{{end}}
	</body>
</html>
`

var EditTemplate = template.Must(template.New("records/edit").Parse(editTemplateRaw))

type EditData struct {
	Flash  string
	Model  *metis.Model
	Record map[string]interface{}
	JSON   string
}

func EditGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	l := logger.WithPrefix("RecordsEditGET: ")

	re := &EditData{
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

	if err := EditTemplate.Execute(w, re); err != nil {
		l.Fatal(err)
	}
}

func EditPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
}
