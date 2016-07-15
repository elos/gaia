package records

import (
	"fmt"
	"net/http"

	"github.com/elos/data"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/access"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

// DeletePOST handles a `POST` request to the `/records/delete/` route of the records web UI.
//
// Parameters:
//	{
//		kind string
//		id string
//	}
//
// DeletePOST removes the record by matching the kind and id parameters.
//
// Success:
//		* StatusFound
//			- record deleted, redirect to /records/query/
//
// Errors:
//		* StatusBadRequest
//			- missing kind
//			- unrecognized kind
//			- missing id
//			- invalid id
//		* StatusUnauthorized
//			- access.CanDelete false
//		* StatusNotFound
//			- db.PopulateByID data.ErrNotFound
//		* StatusInternalServerError
//			- r.ParseForm error
//			- db.PopulateByID error
//			- user.FromContext false
//			- access.CanDelete error
func DeletePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	k := r.FormValue("kind")
	if k == "" {
		http.Error(w, "Missing parameter \"kind\"", http.StatusBadRequest)
		return
	}
	kind := data.Kind(k)

	if !models.Kinds[kind] {
		http.Error(w, fmt.Sprintf("Unrecognized kind: %q", k), http.StatusBadRequest)
		return
	}

	i := r.FormValue("id")
	if i == "" {
		http.Error(w, "Missing parameter \"id\"", http.StatusBadRequest)
		return
	}

	id, err := db.ParseID(i)
	if err != nil {
		http.Error(w, "Invalid \"id\"", http.StatusBadRequest)
		return
	}

	m := models.ModelFor(kind)
	m.SetID(id)
	switch err := db.PopulateByID(m); err {
	case data.ErrNotFound:
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	default:
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return

	case nil:
	}

	u, ok := user.FromContext(ctx)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ok, err = access.CanDelete(db, u, m)

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if err := db.Delete(m); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(
		w, r,
		"/records/query/",
		http.StatusFound,
	)
}
