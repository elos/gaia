package records

import (
	"net/http"

	"github.com/elos/gaia/services"
	"github.com/elos/x/auth"
	"github.com/elos/x/models"
	"github.com/elos/x/records"
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
func DeletePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, webui services.WebUIClient) {
	pu, pr := auth.CredentialsFromRequest(r)

	resp, err := webui.DeletePOST(ctx, &records.DeletePOSTRequest{
		Public:  pu,
		Private: pr,
		Kind:    models.Kind(models.Kind_value[r.FormValue("kind")]),
		Id:      r.FormValue("id"),
	})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp.ServeHTTP(w, r)
}
