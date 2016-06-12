package routes

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/elos/data"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/access"
	"github.com/elos/models/tag"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

const (
	tagsParam = "tags"
)

// EventPOST implements gaia's response to a POST request to the '/event/' endpoint.
//
// Assumptions: The user has been authenticated.
//
// Proceedings: Parses the url parameters.
func EventPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	l := logger.WithPrefix("EventPOST: ")

	// Parse the form
	if err := r.ParseForm(); err != nil {
		l.Printf("error parsing form: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Retrieve the tags parameter
	tagNames, ok := r.Form[tagsParam]
	if !ok {
		l.Print("no tags param")
		tagNames = []string{}
	}

	// Retrieve our user
	u, ok := user.FromContext(ctx)
	if !ok {
		l.Print("failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	e := new(models.Event)

	tags := make([]*models.Tag, len(tagNames))
	for i, n := range tagNames {
		t, err := tag.ForName(db, u, tag.Name(n))
		if err != nil {
			l.Printf("tag.ForName(%q) error: %s", n, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		tags[i] = t
	}

	defer r.Body.Close()
	if requestBody, err := ioutil.ReadAll(r.Body); err != nil {
		l.Printf("ioutil.ReadAll(r.Body) error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	} else if err := json.Unmarshal(requestBody, e); err != nil {
		l.Printf("info: request body:\n%s", string(requestBody))
		l.Printf("error: while unmarshalling request body, %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	e.UpdatedAt = time.Now()
	e.SetOwner(u)

	for _, t := range tags {
		e.IncludeTag(t)
	}

	if allowed, err := access.CanCreate(db, u, e); err != nil {
		l.Printf("access.CanCreate(db, u, e) error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	} else if !allowed {
		l.Print("access.CanCreate(db, u, e) rejected authorization")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if err := db.Save(u); err != nil {
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
	b, err := json.MarshalIndent(e, "", "    ")
	if err != nil {
		l.Printf("json.MarshalIndent(m, \"\", \"   \") error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}
