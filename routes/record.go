package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/elos/data"
	"github.com/elos/data/transfer"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/access"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
	"golang.org/x/net/websocket"
)

// --- URL Parameter Names {{{

// The URL params the /record/ routes expect
const (
	// kindParam is the parameter which specifies the record's kind
	kindParam = "kind"
	// idParam is the parameter which specifies the record's id
	idParam = "id"

	// /record/query/ specific:
	limitParam = "limit"
	batchParam = "batch"
	skipParam  = "skip"

	// /record/changes/ specific:
	publicParam  = "public"
	privateParam = "private"
)

// --- }}}

// --- Authenticate {{{

// Authenticate checks a request's basic authentication, and associates a *models.User with the context
// if so. Otherwise it handles responding to and closing the request.
//
//		contextWithUser, authWasSuccessful := routes.Authenticate(ctx, w, r, logger, db)
//
// Use for any requests which expect to act on behalf of a user (which is most)
func Authenticate(ctx context.Context, w http.ResponseWriter, r *http.Request, l services.Logger, db services.DB) (context.Context, bool) {
	l = l.WithPrefix("routes.Authenticate: ")
	var (
		c   *models.Credential
		u   *models.User
		err error
	)

	public, private, ok := r.BasicAuth()
	if !ok {
		l.Printf("authentication reverting from basic auth to session")
		// assume std lib didn't make a mistake, and the BasicAuth simply wasn't given
		// fall back to cookie

		if sesh, err := session(r, db); err != nil {
			switch err {
			case http.ErrNoCookie:
				l.Printf("no session cookie")
			case data.ErrNotFound:
				l.Printf("session token not found")
			default:
				l.Printf("session(r, db) error: %s", err)
			}

			l.Printf("authentication reverting from cookie to form values")
			public, private = r.FormValue(publicParam), r.FormValue(privateParam)
		} else if sesh.Valid() {
			if u, err := sesh.Owner(db); err != nil {
				l.Printf("sesh.Owner(db) error: %s", err)
				public, private = r.FormValue(publicParam), r.FormValue(privateParam)
			} else {
				return user.NewContext(ctx, u), true
			}
		} else {
			l.Printf("session no longer valid")
			public, private = r.FormValue(publicParam), r.FormValue(privateParam)
		}
	}

	if c, err = access.Authenticate(db, public, private); err != nil {
		l.Printf("authentication of (%s, %s) failed: couldn't find credential: %s", public, private, err)
		goto unauthorized // this error is on us, but it manifests as a failure to authenticate
	}

	if u, err = c.Owner(db); err != nil {
		l.Printf("authentication failed: couldn't load user's owner: %s", err)
		goto unauthorized // this error is also on us, but also manifests as a failure to authenticate
	}

	// successful authentications
	return user.NewContext(ctx, u), true

	// rejection
unauthorized:
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	return nil, false
}

// --- }}}

// --- RecordGET {{{

// RecordGET implements gaia's response to a GET request to the '/record/' endpoint.
//
// Assumptions: The user has been authenticated.
//
// Proceedings: Parses the url parameters, retrieving the kind and id parameters (both required).
// Then it loads that record, checks if the user is allowed to access it, if so it returns the model as JSON.
//
// Success:
//		* StatusOK with the record as JSON
//
// Errors:
//		* InternalServerError: failure to parse the parameters, database connections, json marshalling
//		* BadRequest: no kind param, no id param, unrecognized kind, invalid id
//		* NotFound: unauthorized, record actually doesn't exist
func RecordGET(ctx context.Context, w http.ResponseWriter, r *http.Request, logger services.Logger, db services.DB) {
	l := logger.WithPrefix("RecordGet: ")

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

	bytes, err := json.MarshalIndent(m, "", "	")
	if err != nil {
		l.Printf("error while marshalling json %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

// --- }}}

// --- RecordPOST {{{

// RecordPOST implements gaia's response to a POST request to the '/record/' endpoint.
//
// Assumptions: The user has been authenticated.
//
// Proceedings: Parses the url parameters, and retrieves the kind parameter (required).
// Then it checks whether the payload record's id is declared, generating one if not.
// Finally, it saves or updates the record, returning the record with corresponding status.
//
// Success:
//		* StatusOK with the record as JSON, meaning the record was _updated_
//		* StatusCreated with the record as JSON, meaning the record was _created_
//
// Errors:
//		* InternalServerError: failure to parse the parameters, database connections, json marshalling
//		* BadRequest: no kind param, unrecognized kind
//		* NotFound: unauthorized, record actually doesn't exist
//		* Unauthorized: not authorized to create/update that record, database access denial
func RecordPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, logger services.Logger, db services.DB) {
	l := logger.WithPrefix("RecordPOST: ")

	// Parse the form
	if err := r.ParseForm(); err != nil {
		l.Printf("error parsing form: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	//	l.Printf("%+v", r)

	// Retrieve the kind parameter
	k := r.FormValue(kindParam)
	if k == "" {
		l.Printf("no kind specified")
		http.Error(w, fmt.Sprintf("You must specify a %q parameter", kindParam), http.StatusBadRequest)
		return
	}
	kind := data.Kind(k)

	// Verify it is a recognized kind
	if _, ok := models.Kinds[kind]; !ok {
		http.Error(w, fmt.Sprintf("The kind %q is not recognized", kind), http.StatusBadRequest)
		return
	}

	m := models.ModelFor(kind)

	var requestBody []byte
	var err error

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

	// Now we can determine whether we are creating a new record
	// or updating an existing one. We expect to be updating...
	creation := false

	// ...unless the id is empty, in which case we are creating
	if m.ID().String() == "" {
		m.SetID(db.NewID()) // Be sure to assign it an ID
		creation = true
	}

	// Retrieve our user
	u, ok := user.FromContext(ctx)
	if !ok {
		l.Print("failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var allowed bool

	// We need to check either [creation] we can create the record or [!creation]
	// we can update the record we are trying to update
	if creation {
		prop, ok := m.(access.Property)
		if !ok {
			l.Printf("tried to create record that isn't property")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		allowed, err = access.CanCreate(db, u, prop)
	} else {
		allowed, err = access.CanWrite(db, u, m)
	}

	if err != nil {
		l.Printf("access.{CanCreate | CanWrite} error: %s", err)
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

	// Now we shall write our response
	bytes, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		l.Printf("error marshalling model: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if creation {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

// --- }}}

// --- RecordDELETE {{{

// RecordDelete implements gaia's response to a DELETE request to the '/record/' endpoint.
//
// Assumptions: The user has been authenticated.
//
// Proceedings: Parses the url parameters, and retrieves the kind and id parameters (both required).
// Then checks for authorization to delete, carries it out if allowed.
//
// Success:
//		* StatusNoContent indicating a succesful deletion
//
// Errors:
//		* InternalServerError: failure to parse the parameters, database connections, json marshalling
//		* BadRequest: no kind param, unrecognized kind, no id param, invalid id param
//		* NotFound: unauthorized, record actually doesn't exist
//		* Unauthorized: not authorized to delete that record, database access denial
func RecordDELETE(ctx context.Context, w http.ResponseWriter, r *http.Request, logger services.Logger, db services.DB) {
	l := logger.WithPrefix("RecordDELETE: ")

	// Parse the form
	if err := r.ParseForm(); err != nil {
		l.Printf("error parsing form: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Retrieve the kind parameter
	k := r.FormValue(kindParam)
	if k == "" {
		l.Printf("no kind specified")
		http.Error(w, fmt.Sprintf("You must specify a %q parameter", kindParam), http.StatusBadRequest)
		return
	}
	kind := data.Kind(k)

	// Retrieve the id parameter
	i := r.FormValue(idParam)
	if i == "" {
		l.Printf("no id specified")
		http.Error(w, fmt.Sprintf("You must specify a %q parameter", idParam), http.StatusBadRequest)
		return
	}

	// Verify the kind is recognized
	_, ok := models.Kinds[kind]
	if !ok {
		l.Printf("unrecognized kind: %q", kind)
		http.Error(w, fmt.Sprintf("The kind %q is not recognized", kind), http.StatusBadRequest)
		return
	}

	// Verify the id is valid
	id, err := db.ParseID(i)
	if err != nil {
		l.Printf("invalid id: %q, error: %s", i, err)
		http.Error(w, fmt.Sprintf("The id %q is invalid", i), http.StatusBadRequest)
		return
	}

	// Get the record, so that we can decide whether we have permission to delete it
	m := models.ModelFor(kind)
	m.SetID(id)
	if err = db.PopulateByID(m); err != nil {
		l.Printf("db.PopulateByID error: %s", err)
		switch err {
		case data.ErrAccessDenial:
			fallthrough // don't leak information (were we denied access, this record doesn't exist)
		case data.ErrNotFound:
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		case data.ErrNoConnection:
			fallthrough
		case data.ErrInvalidID:
			fallthrough
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	// Retrieve the user we are authenticated as
	u, ok := user.FromContext(ctx)
	if !ok {
		l.Print("faild to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// check for authorization
	if allowed, err := access.CanDelete(db, u, m); err != nil {
		// TODO(nclandolfi) standardize this with the POST and GET where we handle the possible errors
		l.Printf("RecordDELETE Error: %s", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	} else if !allowed {
		// in order to not leak information, we treat this as a not found
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if err := db.Delete(m); err != nil {
		switch err {
		case data.ErrAccessDenial:
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound) // don't leak information
		case data.ErrNotFound:
			// this shouldn't happen unless it was deleted by another process
			// in between when we populated the record by id, in which case it was successful
			goto successfulDelete
			// all of these errors, we can't guarantee deletion
		case data.ErrNoConnection:
			fallthrough
		case data.ErrInvalidID:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		default:
			l.Printf("RecordDELETE Error: %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

successfulDelete:
	w.WriteHeader(http.StatusNoContent)
}

// --- }}}

// --- RecordOPTIONS {{{

// RecordOPTIONS implements gaia's response to a OPTIONS request to the /record/ endpoint
//
// Assumptions: The user has been authenticated [not used, but verification to talk to gaia].
//
// Proceedings: Write a success, the CORS header should already have been applied by middleware.
//
// Success:
//		* StatusOK
func RecordOPTIONS(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// The CORS header has already been applied by our middleware,
	// therefore we need only indicate a successful response.
	w.WriteHeader(http.StatusOK)
}

// --- }}}

// --- RecordQueryPOST {{{

// RecordQueryPOST implements gaia's response to a POST request to the '/record/query/' endpoint.
//
// Assumptions: The user has been authenticated.
//
// Proceedings:
//
// Success:
//		* StatusOK
//
// Error:
//		* InternalServerError: parsing url params,
//		* BadRequest: no kind parameter, unrecognized kind
func RecordQueryPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, logger services.Logger, db data.DB) {
	l := logger.WithPrefix("RecordQueryPOST: ")

	// Parse the form
	if err := r.ParseForm(); err != nil {
		l.Printf("error parsing form: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Retrieve the kind parameter
	k := r.FormValue(kindParam)
	if k == "" {
		l.Printf("no kind parameter")
		http.Error(w, fmt.Sprintf("You must specify a %q parameter", kindParam), http.StatusBadRequest)
		return
	}
	kind := data.Kind(k)

	// Verify the kind is recognized
	if !models.Kinds[kind] {
		l.Printf("unrecognized kind %q", kind)
		http.Error(w, fmt.Sprintf("The kind %q is not recognized", kind), http.StatusBadRequest)
		return
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

	// Read the selection attrs from the body
	var requestBody []byte
	var err error
	defer r.Body.Close()
	if requestBody, err = ioutil.ReadAll(r.Body); err != nil {
		l.Printf("error while reading request body: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// These are the selectors, unmarshal the request body into them
	attrs := make(data.AttrMap)

	// only unmarshall if there is any request body
	if len(requestBody) > 0 {
		if err = json.Unmarshal(requestBody, &attrs); err != nil {
			l.Printf("info: request body:\n%s", string(requestBody))
			l.Printf("error: while unmarshalling request body, %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// Retrieve the user we are acting on behalf
	u, ok := user.FromContext(ctx)
	if !ok {
		l.Print("failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Load our actual query
	var iter data.Iterator
	if iter, err = db.Query(kind).Select(attrs).Limit(limit).Batch(batch).Skip(skip).Order(r.Form["order"]...).Execute(); err != nil {
		l.Printf("db.Query(%q).Select(%v).Limit(%d).Batch(%d).Skip(%d) error: %s", kind, attrs, limit, batch, skip, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "[")

	first := true

	// Iterator through the results and write the response
	m := models.ModelFor(kind)
	for iter.Next(m) {
		if !first {
			fmt.Fprint(w, ",")
		}
		first = false
		if ok, err := access.CanRead(db, u, m); err != nil {
			// We've hit an error and need to bail
			l.Printf("access.CanRead error: %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if ok {
			bytes, err := json.Marshal(m)
			if err != nil {
				l.Printf("error marshalling JSON: %s", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			w.Write(bytes)
		}
	}

	if err := iter.Close(); err != nil {
		l.Printf("error closing query, %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "]")
}

// --- }}}

// --- {Contextualize}RecordChangesGET {{{

func ContextualizeRecordChangesGET(ctx context.Context, db data.DB, logger services.Logger) websocket.Handler {
	return func(ws *websocket.Conn) {
		defer ws.Close()

		l := logger.WithPrefix("ContextualizeRecordChangesGET: ")

		if err := ws.Request().ParseForm(); err != nil {
			l.Printf("error parsing form: %s", err)
			return
		}

		public := ws.Request().Form.Get(publicParam)
		private := ws.Request().Form.Get(privateParam)

		if public == "" || private == "" {
			l.Print("failed to retrieve credentials")
			return
		}

		cred, err := access.Authenticate(db, public, private)
		if err != nil {
			l.Print("failed to authenticate")
			return
		}

		if u, err := cred.Owner(db); err != nil {
			l.Print("error retrieving user: %s", err)
		} else {
			RecordChangesGET(user.NewContext(ctx, u), ws, db, logger)
		}
	}
}

func RecordChangesGET(ctx context.Context, ws *websocket.Conn, db data.DB, logger services.Logger) {
	l := logger.WithPrefix("RecordChangesGet: ")

	u, ok := user.FromContext(ctx)
	if !ok {
		l.Print("failed to retrieve user from context")
		return
	}

	// Get the db's changes, then filter by updates, then
	// filter by whether this user can read the record
	changes := data.Filter(db.Changes(), func(c *data.Change) bool {
		ok, err := access.CanRead(db, u, c.Record)
		if err != nil {
			l.Printf("error checking access control: %s", err)
		}
		return ok
	})

	var kind data.Kind
	if kindParam := ws.Request().Form.Get(kindParam); kindParam != "" {
		kind = data.Kind(kindParam)

		if _, ok := models.Kinds[kind]; !ok {
			l.Printf("unrecognized kind: %q", kind)
			if err := websocket.Message.Send(ws, fmt.Sprintf("The kind %q is not recognized", kind)); err != nil {
				if err != io.EOF {
					l.Printf("error sending on websocket: %s", err)
				}
			}
			return
		}

		// If a kind was specified, filter by it
		changes = data.FilterKind(changes, kind)
	}

	for {
		select {
		case change, ok := <-*changes:
			if !ok {
				l.Printf("change channel was closed")
				return
			}

			l.Printf("recieved change: %+v", change)

			changeTransport := transfer.Change(change)

			if err := websocket.JSON.Send(ws, changeTransport); err != nil {
				if err != io.EOF {
					l.Printf("error sending to socket: %s", err)
				}

				return
			}
		case <-time.After(5 * time.Second):
			l.Printf("no change in 5 seconds, but still listening")
		case <-ctx.Done():
			l.Printf("context cancelled")
			// context was cancelled
			return
		}
	}
}

// --- }}}
