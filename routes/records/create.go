package records

import (
	"io/ioutil"
	"net/http"

	"github.com/elos/data"
	"github.com/elos/gaia/services"
	"github.com/elos/x/auth"
	"github.com/elos/x/models"
	"github.com/elos/x/records"
	"golang.org/x/net/context"
)

// CreateGET handles a `GET` request to the `/records/create/` route of the records web UI.
//
// Parameters:
//		{
//			kind string
//		}
//
// CreateGET provides a web frame to create a new record of the provided `kind`.
//
// Success:
//		* StatusOK
//			- html page with form to create the record
//
// Errors:
//		* StatusBadRequest:
//			- missing kind
//			- unrecognized kind
//		* StatusUnauthorized
//			- model.(access.Property) false
//		* StatusInternalServerError
//			- error parsing form
//			- error marshalling model into form
//			- EditTemplate.Execute error
func CreateGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, l services.Logger, webui services.WebUIClient) {
	pu, pr := auth.CredentialsFromRequest(r)

	resp, err := webui.CreateGET(ctx, &records.CreateGETRequest{
		Public:  pu,
		Private: pr,
		Kind:    models.Kind(models.Kind_value[r.FormValue("kind")]),
	})

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp.ServeHTTP(w, r)
}

// CreatePOST handles a `POST` request to the `/records/create/` route of the records web UI.
//
// Parameters:
//		{
//			kind string
//			<arbitrary-fields> {string|[]string}
//		}
//
// CreatePOST creates the record by matching the `kind`, and dynamically pulls the appropriate attributes
// from the form parameters.
//
// Success:
//		* StatusFound
//			- record created, redirect to /records/view/
//
// Errors:
//		* StatusBadRequest
//			- missing kind
//			- unrecognized kind
//		* StatusUnauthorized
//			- model not property
//			- access.CanCreate false
//		* StatusInternalServerError
//			- r.ParseForm error
//			- unmarshalling model
//			- ctx missing user
//			- access.CanCreate error
//			- db.Save error
func CreatePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, l services.Logger, webui services.WebUIClient) {
	pu, pr := auth.CredentialsFromRequest(r)

	defer r.Body.Close()
	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp, err := webui.CreatePOST(ctx, &records.CreatePOSTRequest{
		Public:  pu,
		Private: pr,
		Url:     r.URL.String(),
		Body:    bytes,
	})

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp.ServeHTTP(w, r)
}
