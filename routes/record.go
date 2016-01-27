package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/elos/data"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/access"
	"golang.org/x/net/context"
	"golang.org/x/net/websocket"
)

type key int

var userKey key = 0

func userFromContext(ctx context.Context) (*models.User, bool) {
	u, ok := ctx.Value(userKey).(*models.User)
	return u, ok
}

func Authenticate(ctx context.Context, w http.ResponseWriter, r *http.Request, l services.Logger, db services.DB) (context.Context, bool) {
	public, private, ok := r.BasicAuth()
	if !ok {
		l.Print("Authentication failed: couldn't retrieve basic auth")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return nil, false
	}

	cred, err := access.Authenticate(db, public, private)
	if err != nil {
		l.Print("Authentication failed: couldn't find credential")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return nil, false
	}

	u, err := cred.Owner(db)
	if err != nil {
		l.Print("Authentication failed: couldn't load user's owner")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return nil, false
	}

	return context.WithValue(ctx, userKey, u), true
}

const kindParam = "kind"
const idParam = "id"

func RecordGET(ctx context.Context, w http.ResponseWriter, r *http.Request, l services.Logger, db services.DB) {
	if err := r.ParseForm(); err != nil {
		l.Printf("RecordGET Error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	k := r.Form[kindParam]
	if len(k) == 0 {
		http.Error(w, "You must specifiy a kind", http.StatusBadRequest)
		return
	}

	kind := data.Kind(k[0])

	i := r.Form[idParam]
	if len(i) == 0 {
		http.Error(w, "You must specify an id", http.StatusBadRequest)
		return
	}

	id, err := db.ParseID(i[0])
	if err != nil {
		http.Error(w, fmt.Sprintf("The id '%s' is invalid", id), http.StatusBadRequest)
		return
	}

	if _, ok := models.Kinds[kind]; !ok {
		http.Error(w, fmt.Sprintf("The kind '%s' is not recognized", kind), http.StatusBadRequest)
		return
	}

	m := models.ModelFor(kind)
	m.SetID(id)

	if err := db.PopulateByID(m); err != nil {
		switch err {
		case data.ErrAccessDenial:
			fallthrough // don't leak information
		case data.ErrNotFound:
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		case data.ErrNoConnection:
			fallthrough
		case data.ErrInvalidID:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		default:
			l.Printf("RecordGET Error: %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	user, ok := userFromContext(ctx)
	if !ok {
		l.Print("RecordGET Error: failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if allowed, err := access.CanRead(db, user, m); err != nil {
		switch err {
		case data.ErrAccessDenial:
			fallthrough // don't leak information
		case data.ErrNotFound:
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		case data.ErrNoConnection:
			fallthrough
		case data.ErrInvalidID:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		default:
			l.Printf("RecordGET Error: %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	} else if !allowed {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	bytes, err := json.MarshalIndent(m, "", "	")
	if err != nil {
		l.Printf("RecordGET Error: while marshalling json %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}

// RecordPOST handles a 'POST' request to /records
//
// It has logging and CORS middleware applied to it
func RecordPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, l services.Logger, db services.DB) {
	if err := r.ParseForm(); err != nil {
		l.Printf("RecordPOST Error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	k := r.Form[kindParam]
	if len(k) == 0 {
		http.Error(w, "You must specifiy a kind", http.StatusBadRequest)
		return
	}

	kind := data.Kind(k[0])

	if _, ok := models.Kinds[kind]; !ok {
		http.Error(w, fmt.Sprintf("The kind '%s' is not recognized", kind), http.StatusBadRequest)
		return
	}

	defer r.Body.Close()
	m := models.ModelFor(kind)
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		l.Printf("RecordPOST Error: while reading request body, %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	l.Print("Request Body:\n%s", string(body))

	if err = json.Unmarshal(body, m); err != nil {
		l.Printf("RecordPOST Info: request body:\n%s", string(body))
		l.Printf("RecordPOST Error: while unmarshalling request body, %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	user, ok := userFromContext(ctx)
	if !ok {
		l.Print("RecordPOST Error: failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	creation := false

	if m.ID().String() == "" {
		m.SetID(db.NewID())
		creation = true
	}

	if creation {
		if allowed, err := access.CanCreate(db, user, m); err != nil {
			l.Print("CanCreate Error: %s", err)
			switch err {
			case data.ErrAccessDenial:
				fallthrough // don't leak information
			case data.ErrNotFound:
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			case data.ErrNoConnection:
				fallthrough
			case data.ErrInvalidID:
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			default:
				l.Printf("RecordPOST Error: %s", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else if !allowed {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
	} else {
		if allowed, err := access.CanWrite(db, user, m); err != nil {
			l.Print("CanWrite Error: %s", err)
			switch err {
			case data.ErrAccessDenial:
				fallthrough // don't leak information
			case data.ErrNotFound:
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			case data.ErrNoConnection:
				fallthrough
			case data.ErrInvalidID:
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			default:
				l.Printf("RecordPOST Error: %s", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else if !allowed {
			l.Print("write access denied")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
	}

	if err := db.Save(m); err != nil {
		l.Print("Save Error: %s", err)
		switch err {
		case data.ErrAccessDenial:
			fallthrough // don't leak information
		case data.ErrNotFound:
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		case data.ErrNoConnection:
			fallthrough
		case data.ErrInvalidID:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		default:
			l.Printf("RecordPOST Error: %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

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

	w.Write(bytes)
}

// RecordDELETE handles a 'DELETE' request to /records
//
// It has logging and CORS middleware applied to it.
func RecordDELETE(ctx context.Context, w http.ResponseWriter, r *http.Request, l services.Logger, db services.DB) {
	if err := r.ParseForm(); err != nil {
		l.Printf("RecordDELETE Error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	k := r.Form[kindParam]
	if len(k) == 0 {
		http.Error(w, "You must specifiy a kind", http.StatusBadRequest)
		return
	}

	kind := data.Kind(k[0])

	i := r.Form[idParam]
	if len(i) == 0 {
		http.Error(w, "You must specify an id", http.StatusBadRequest)
		return
	}

	id, err := db.ParseID(i[0])
	if err != nil {
		http.Error(w, fmt.Sprintf("The id '%s' is invalid", id), http.StatusBadRequest)
		return
	}

	_, ok := models.Kinds[kind]
	if !ok {
		http.Error(w, fmt.Sprintf("The kind '%s' is not recognized", kind), http.StatusBadRequest)
		return
	}

	m := models.ModelFor(kind)
	m.SetID(id)
	if err := db.PopulateByID(m); err != nil {
		switch err {
		case data.ErrAccessDenial:
			fallthrough // don't leak information
		case data.ErrNotFound:
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
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

	user, ok := userFromContext(ctx)
	if !ok {
		l.Print("RecordDELETE Error: failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if allowed, err := access.CanDelete(db, user, m); err != nil {
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
			fallthrough // don't leak information
		case data.ErrNotFound:
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
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

	w.WriteHeader(http.StatusNoContent)
}

// RecordOPTIONS handles an 'OPTIONS' request to /records
//
// It has logging and CORS middleware applied to it.
func RecordOPTIONS(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// The CORS header has already been applied by our middleware,
	// therefore we just need to indicate a successful response.
	w.WriteHeader(http.StatusOK)
}

func RecordQueryPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, l services.Logger, db data.DB) {
	if err := r.ParseForm(); err != nil {
		l.Printf("RecordQueryPOST Error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	k := r.Form[kindParam]
	if len(k) == 0 {
		http.Error(w, "You must specifiy a kind", http.StatusBadRequest)
		return
	}

	kind := data.Kind(k[0])

	if _, ok := models.Kinds[kind]; !ok {
		http.Error(w, fmt.Sprintf("The kind '%s' is not recognized", kind), http.StatusBadRequest)
		return
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		l.Printf("RecordQueryPOST Error: while reading request body, %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	attrs := make(data.AttrMap)

	if err = json.Unmarshal(body, &attrs); err != nil {
		l.Printf("RecordQueryPOST Info: request body:\n%s", string(body))
		l.Printf("RecordQueryPOST Error: while unmarshalling request body, %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	user, ok := userFromContext(ctx)
	if !ok {
		l.Print("RecordQueryPOST Error: failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	q := db.Query(kind)
	q.Select(attrs)
	iter, err := q.Execute()
	if err != nil {
		l.Printf("RecordQueryPOST Error: while executing query, %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	results := make([]data.Record, 0)
	m := models.ModelFor(kind)
	for iter.Next(m) {
		if ok, err := access.CanRead(db, user, m); err != nil {
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

	returnBody, err := json.Marshal(results)
	if err != nil {
		l.Printf("RecordQueryPOST Error: while loading query, %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(returnBody)
}

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
		RecordChangesGET(context.WithValue(ctx, userKey, u), ws, db, logger)
	}
}

func RecordChangesGET(ctx context.Context, ws *websocket.Conn, db data.DB, logger services.Logger) {
	user, ok := userFromContext(ctx)
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
		ok, _ := access.CanRead(db, user, c.Record)
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
