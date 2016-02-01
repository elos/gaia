package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/elos/data"
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
	var (
		c   *models.Credential
		u   *models.User
		err error
	)

	public, private, ok := r.BasicAuth()
	if !ok {
		// assume std lib didn't make a mistake, and the BasicAuth simply wasn't given
		goto unauthorized
	}

	if c, err = access.Authenticate(db, public, private); err != nil {
		l.Printf("routes.Authenticate Error: authentication failed: couldn't find credential: %s", err)
		goto unauthorized // this error is on us, but it manifests as a failure to authenticate
	}

	if u, err = c.Owner(db); err != nil {
		l.Printf("routes.Authenticate Error: authentication failed: couldn't load user's owner: %s", err)
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

// RecordGET implements gaia's response to a GET request to the /record/ endpoint
//
// Assumes: The user has been authenticated
//
// Proceedings: First parses the url parameters, and retrieves the kind and id parameters; both of which are required
//	Then it loads that record, checks if the user is allowed to access it, and then returns the model as JSON
//
// Success:
//		StatusOK with the record as JSON
//
// Errors:
//		* InternalServerError: failure to parse the parameters, database connections, json marshalling
//		* BadRequest: no kind param, no id param, unrecognized kind, invalid id
//		* NotFound: unauthorized, record actually doesn't exist
func RecordGET(ctx context.Context, w http.ResponseWriter, r *http.Request, l services.Logger, db services.DB) {
	// Parse the form value
	if err := r.ParseForm(); err != nil {
		l.Printf("RecordGET Error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Secure the kind parameter's existence, and superficial validity (i.e., non-empty)
	k := r.FormValue(kindParam)
	if k == "" {
		http.Error(w, "You must specifiy a kind", http.StatusBadRequest)
		return
	}
	kind := data.Kind(k)

	// Secure the id parameter's existence, and superficial validity (i.e., non-empty)
	i := r.FormValue(idParam)
	if i == "" {
		http.Error(w, "You must specify an id", http.StatusBadRequest)
		return
	}

	// Ensure the kind is recognized
	if _, ok := models.Kinds[kind]; !ok {
		http.Error(w, fmt.Sprintf("The kind '%s' is not recognized", kind), http.StatusBadRequest)
		return
	}

	// Ensure the id is valid
	id, err := db.ParseID(i)
	if err != nil {
		http.Error(w, fmt.Sprintf("The id '%s' is invalid", id), http.StatusBadRequest)
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
			// This is, by fay, the most common case here.
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		// ErrNoConnection, ErrInvalidID and the undetermined errors are all non-normal cases
		case data.ErrNoConnection:
			fallthrough
		case data.ErrInvalidID:
			fallthrough
		default:
			l.Printf("RecordGET Error: %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return // regardless of what the error was, we are bailing
	}

	// Retrieve the user this request was authenticated as
	u, ok := user.FromContext(ctx)
	if !ok { // This is certainly an issue, and should _never_ happen
		l.Print("RecordGET Error: failed to retrieve user from context")
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
			l.Printf("RecordGET Error: %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	} else if !allowed {
		// If you can't read the record you are asking for,
		// it does not exist (as far as you are concerned"
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	bytes, err := json.MarshalIndent(m, "", "	")
	if err != nil {
		l.Printf("RecordGET Error: while marshalling json %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

// --- }}}

// --- RecordPOST {{{

// RecordPOST implements gaia's response to a POST request to the /record/ endpoint
//
// Assumes: The user has been authenticated
//
// Proceedings: First parses the url parameters, and retrieves the kind parameter, which is required.
//	Then it checks whether the record's id is declared, if not it generates one. And proceeds to
//	save/update the record.
//
// Success:
//		StatusOK with the record as JSON, meaning the record was _updated_
//		StatusCreated with the record as JSON, meaning the record was _created_
//
// Errors:
//		* InternalServerError: failure to parse the parameters, database connections, json marshalling
//		* BadRequest: no kind param, unrecognized kind
//		* NotFound: unauthorized, record actually doesn't exist
//		* Unauthorized: not authorized to create/update that record, database access denial
func RecordPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, l services.Logger, db services.DB) {
	var err error

	// Parse the form
	if err = r.ParseForm(); err != nil {
		l.Printf("RecordPOST Error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Retrieve the kind parameter
	k := r.FormValue(kindParam)
	if k == "" {
		http.Error(w, "You must specifiy a kind", http.StatusBadRequest)
		return
	}
	kind := data.Kind(k)

	// Verify it is a recognized kind
	if _, ok := models.Kinds[kind]; !ok {
		http.Error(w, fmt.Sprintf("The kind '%s' is not recognized", kind), http.StatusBadRequest)
		return
	}

	m := models.ModelFor(kind)

	var requestBody []byte

	// Now we must read the body of the request
	defer r.Body.Close() // don't forget to close it
	if requestBody, err = ioutil.ReadAll(r.Body); err != nil {
		l.Printf("RecordPOST Error: while reading request body: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Now we unmarshal that into the record
	if err = json.Unmarshal(requestBody, m); err != nil {
		l.Printf("RecordPOST Info: request body:\n%s", string(requestBody))
		l.Printf("RecordPOST Error: while unmarshalling request body, %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Now we can determine whether we are creating a new record
	// or updating an existing one. We expect to be updating...
	creation := false

	// Unless the id is empty, in which case we are creating
	if m.ID().String() == "" {
		m.SetID(db.NewID()) // Be sure to assign it an ID
		creation = true
	}

	// Retrieve our user
	u, ok := user.FromContext(ctx)
	if !ok {
		l.Print("RecordPOST Error: failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var allowed bool

	// We need to check either [creation] we can create the record or [!creation]
	// we can update the record we are trying to update
	if creation {
		allowed, err = access.CanCreate(db, u, m)
	} else {
		allowed, err = access.CanWrite(db, u, m)
	}

	if err != nil {
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
			l.Printf("RecordPOST Error: %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	} else if !allowed {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// If we have made it this far, it only remains to commit the record
	if err = db.Save(m); err != nil {
		switch err {
		case data.ErrAccessDenial:
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		// These are all equally distressing
		case data.ErrNotFound: // TODO shouldn't a not found not be fing possible for a Save?
			fallthrough
		case data.ErrNoConnection:
			fallthrough
		case data.ErrInvalidID:
		default:
			l.Printf("RecordPOST Error: %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	// Now we shall write our response
	bytes, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		l.Printf("RecordPOST Error: %s", err)
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

// RecordDelete implements gaia's response to a DELETE request to the /record/ endpoint
//
// Assumes: The user has been authenticated
//
// Proceedings: First parses the url parameters, and retrieves the kind and id parameters,
//	which are required. Then it checks for authorization to delete, and proceeds with the
//	action if so.
//
// Success:
//		StatusNoContent indicating a succesful deletion
//
// Errors:
//		* InternalServerError: failure to parse the parameters, database connections, json marshalling
//		* BadRequest: no kind param, unrecognized kind, no id param, invalid id param
//		* NotFound: unauthorized, record actually doesn't exist
//		* Unauthorized: not authorized to delete that record, database access denial
func RecordDELETE(ctx context.Context, w http.ResponseWriter, r *http.Request, l services.Logger, db services.DB) {
	// Parse the form
	if err := r.ParseForm(); err != nil {
		l.Printf("RecordDELETE Error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Retrieve the kind parameter
	k := r.FormValue(kindParam)
	if k == "" {
		http.Error(w, "You must specifiy a kind", http.StatusBadRequest)
		return
	}
	kind := data.Kind(k)

	// Retrieve the id parameter
	i := r.FormValue(idParam)
	if i == "" {
		http.Error(w, "You must specify an id", http.StatusBadRequest)
		return
	}

	// Verify the kind is recognized
	_, ok := models.Kinds[kind]
	if !ok {
		http.Error(w, fmt.Sprintf("The kind '%s' is not recognized", kind), http.StatusBadRequest)
		return
	}

	// Verify the id is valid
	id, err := db.ParseID(i)
	if err != nil {
		http.Error(w, fmt.Sprintf("The id '%s' is invalid", id), http.StatusBadRequest)
		return
	}

	// Get the record, so that we can decide whether we have permission to delete it
	m := models.ModelFor(kind)
	m.SetID(id)
	if err = db.PopulateByID(m); err != nil {
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
			l.Printf("RecordDELETE Error: %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	// Retrieve the user we are authenticated as
	u, ok := user.FromContext(ctx)
	if !ok {
		l.Print("RecordDELETE Error: failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// check for authorization
	if allowed, err := access.CanDelete(db, u, m); err != nil {
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
// Assumes: The user has been authenticated
//
// Proceedings: Write a success, the CORS header should already have been applied by middleware
//
// Success:
//		StatusOK
func RecordOPTIONS(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// The CORS header has already been applied by our middleware,
	// therefore we need only indicate a successful response.
	w.WriteHeader(http.StatusOK)
}

// --- }}}

// --- RecordQueryPOST {{{

// RecordQueryPOST implements gaia's response to a POST request to the /record/query/ endpoint
//
// Assumes: The user has been authenticated
//
// Proceedings:
//
// Success:
//		StatusOK
//
// Error:
//		* InternalServerError: parsing url params,
//		* BadRequest: no kind parameter, unrecognized kind
func RecordQueryPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, l services.Logger, db data.DB) {
	var err error

	// Parse the form
	if err = r.ParseForm(); err != nil {
		l.Printf("RecordQueryPOST Error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Retrieve the kind parameter
	k := r.FormValue(kindParam)
	if k == "" {
		http.Error(w, "You must specifiy a kind", http.StatusBadRequest)
		return
	}
	kind := data.Kind(k)

	// Verify the kind is recognized
	if _, ok := models.Kinds[kind]; !ok {
		http.Error(w, fmt.Sprintf("The kind '%s' is not recognized", kind), http.StatusBadRequest)
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
	defer r.Body.Close()
	if requestBody, err = ioutil.ReadAll(r.Body); err != nil {
		l.Printf("RecordQueryPOST Error: while reading request body: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// These are the selectors, unmarshal the request body into them
	attrs := make(data.AttrMap)

	// only unmarshall if there is any request body
	if len(requestBody) > 0 {
		if err = json.Unmarshal(requestBody, &attrs); err != nil {
			l.Printf("RecordQueryPOST Info: request body:\n%s", string(requestBody))
			l.Printf("RecordQueryPOST Error: while unmarshalling request body, %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// Retrieve the user we are acting on behalf
	u, ok := user.FromContext(ctx)
	if !ok {
		l.Print("RecordQueryPOST Error: failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Load our actual query
	var iter data.Iterator
	query := db.Query(kind).Select(attrs)

	if limit > 0 {
		query.Limit(limit)
	}
	if batch > 0 {
		query.Batch(batch)
	}
	if skip > 0 {
		query.Skip(0)
	}

	if iter, err = query.Execute(); err != nil {
		l.Printf("RecordQueryPOST Error: while executing query, %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Iterator through the results and write the response
	results := make([]data.Record, 0)
	m := models.ModelFor(kind)
	for iter.Next(m) {
		if ok, err := access.CanRead(db, u, m); err != nil {
			// We've hit an error and need to bail
			l.Printf("RecordQueryPOST Error: while processing query, %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if ok {
			results = append(results, m)
			m = models.ModelFor(kind)
		}
	}

	if err := iter.Close(); err != nil {
		l.Printf("RecordQueryPOST Error: while loading query, %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	returnBody, err := json.MarshalIndent(results, "", "    ")
	if err != nil {
		l.Printf("RecordQueryPOST Error: while loading query, %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(returnBody)
}

// --- }}}

func ContextualizeRecordChangesGET(ctx context.Context, db data.DB, logger services.Logger) websocket.Handler {
	return func(ws *websocket.Conn) {
		if err := ws.Request().ParseForm(); err != nil {
			logger.Print("Failure parsing form")
			return
		}
		public := ws.Request().Form.Get("public")
		private := ws.Request().Form.Get("private")

		if public == "" || private == "" {
			logger.Print("failed to retrieve credentials")
			return
		}

		cred, err := access.Authenticate(db, public, private)
		if err != nil {
			logger.Print("failed to auth")
			return
		}

		u, _ := cred.Owner(db)
		RecordChangesGET(user.NewContext(ctx, u), ws, db, logger)
	}
}

func RecordChangesGET(ctx context.Context, ws *websocket.Conn, db data.DB, logger services.Logger) {
	u, ok := user.FromContext(ctx)
	if !ok {
		logger.Print("RecordChangesGET Error: failed to retrieve user from context")
		return
	}

	var kind data.Kind
	kindParam := ws.Request().Form.Get("kind")

	if kindParam != "" {
		kind = data.Kind(kindParam)
		if _, ok := models.Kinds[kind]; !ok {
			logger.Printf("RecordChangesGET Error: unrecognized kind: '%s':", kind)
			websocket.Message.Send(ws, fmt.Sprintf("The kind '%s' is not recognized", kind))
			return
		}
	}

	audit := db.Changes()
	go func() {
		for c := range *audit {
			log.Printf("[AUDIT]: \n %+v", c)
		}
	}()

	// Get the db's changes, then filter by updates, then
	// filter by whether this user can read the record
	changes := data.Filter(db.Changes(), func(c *data.Change) bool {
		ok, _ := access.CanRead(db, u, c.Record)
		return ok
	})

	// If a kind was specified, filter by than
	if kind != data.Kind("") {
		changes = data.FilterKind(changes, kind)
	}

	for {
		select {
		case change, ok := <-*changes:
			log.Printf("Recieved Change: %+v", change)
			// channels was closed
			if !ok {
				return
			}

			if err := websocket.JSON.Send(ws, change); err != nil {
				if err != io.EOF {
					logger.Printf("Error reading from socket: %s", err)
				}

				return
			}
		case <-ctx.Done():
			// context was cancelled
			return
		}
	}
}
