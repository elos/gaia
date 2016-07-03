package records

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/elos/data"
	"github.com/elos/data/transfer"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/access"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

const (
	idParam = "id"
)

func EditGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	l := logger.WithPrefix("RecordsEditGET: ")

	cd := &CreateData{
		JSON: make(map[string]interface{}),
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

	cd.Model = models.Metis[kind]
	transfer.TransferAttrs(m, &cd.JSON)
	if err := CreateTemplate.Execute(w, cd); err != nil {
		l.Fatal(err)
	}
}

func EditPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	l := logger.WithPrefix("EditPOST")

	if err := r.ParseForm(); err != nil {
		l.Printf("r.ParseForm() error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	k := r.FormValue(kindParam)
	if k == "" {
		l.Printf("r.FormValue(kindParam): got %q, want a non-empty string", k)
		http.Error(w, fmt.Sprintf("You must specify a %q parameter", kindParam), http.StatusBadRequest)
		return
	}
	kind := data.Kind(k)

	// Lookup the kind to ensure its existence.
	if _, ok := models.Kinds[kind]; !ok {
		l.Printf("_, ok := models.Kind[kind]: got %t, want true", ok)
		http.Error(w, fmt.Sprintf("The kind %q is not recognized", kind), http.StatusNotFound)
		return
	}

	// Secure the id parameter's existence, and superficial validity (i.e., non-empty)
	i := r.FormValue(idParam)
	if i == "" {
		l.Printf("no id parameter")
		http.Error(w, fmt.Sprintf("You must specify a '%s' parameter", idParam), http.StatusBadRequest)
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

	var requestBody []byte

	// Now we must read the body of the request
	defer r.Body.Close() // don't forget to close it
	if requestBody, err = ioutil.ReadAll(r.Body); err != nil {
		l.Printf("error while reading request body: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Now we unmarshal that into the record
	if err = json.Unmarshal(requestBody, m); err != nil {
		l.Printf("info: request body:\n%s", string(requestBody))
		l.Printf("error: while unmarshalling request body, %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	m.SetID(id)

	// Retrieve our user
	u, ok := user.FromContext(ctx)
	if !ok {
		l.Print("failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	allowed, err := access.CanWrite(db, u, m)

	if err != nil {
		l.Printf("access.CanWrite error: %s", err)
		switch err {
		// This indicates that no, you have no access
		case data.ErrAccessDenial:
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		// All of these are bad, and considered an internal error
		case data.ErrNotFound:
			fallthrough
		case data.ErrNoConnection:
			fallthrough
		case data.ErrInvalidID:
			fallthrough
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	} else if !allowed {
		l.Printf("access denied at create/update stage")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// If we have made it this far, it only remains to commit the record
	if err = db.Save(m); err != nil {
		l.Printf("error saving record: %s", err)
		switch err {
		case data.ErrAccessDenial:
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		// These are all equally distressing
		case data.ErrNotFound: // TODO shouldn't a not found not be fing impossible for a Save?
			fallthrough
		case data.ErrNoConnection:
			fallthrough
		case data.ErrInvalidID:
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/records/view/?kind=%s&id=%s", m.Kind(), m.ID()), http.StatusFound)
}
