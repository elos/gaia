package records

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/elos/gaia/services"
	"github.com/elos/x/auth"
	"github.com/elos/x/models"
	"github.com/elos/x/records"
	"golang.org/x/net/context"
)

const editTemplateRaw = `<html>
	<body>
		{{ with .Flash -}}
			{{ . }}
		{{- end }}

		<form method="post">
		{{ with .FormHTML }}
			{{ . }}
		{{ end }}
			{{ with .SubmitText }}
				<input type="submit" value="{{ . }}">
			{{ else }}
				<input type="submit" value="Save">
			{{ end }}
		</form>
		{{ if and .Kind .ID }}
		<a href="/records/view/?kind={{ .Kind }}&id={{ .ID }}"> View </a>
		{{ end }}
	</body>
</html>`

var EditTemplate = template.Must(
	template.New("records/edit").Parse(editTemplateRaw),
)

type EditData struct {
	Flash      string
	FormHTML   template.HTML
	SubmitText string

	Kind, ID string
}

// EditGET handles a `GET` request to the `/records/edit/` route of the records web UI.
//
// Parameters:
//		{
//			kind string
//			id string
//		}
//
// EditGET presents an HTML form containing the record under question.
//
// Success:
//		* StatusOK
//			- html page with form to edit record
// Errors:
//		* StatusBadRequest
//			- missing kind
//			- unrecognized kind
//			- missing id
//			- invalid id
//		* StatusUnauthorized
//			- access.CanWrite false
//		* StatusNotFound
//			- db.PopulateByID data.ErrNotFound
//		* StatusInternalServerError
//			- r.ParseForm error
//			- db.PopulateByID error
//			- ctx missing user
//			- access.CanWrite error
//			- form.Marshal error
//			- EditTemplate.Execute error
func EditGET(ctx context.Context, w http.ResponseWriter, r *http.Request, webui services.WebUIClient) {
	pu, pr := auth.CredentialsFromRequest(r)

	resp, err := webui.EditGET(ctx, &records.EditGETRequest{
		Public:  pu,
		Private: pr,
		Kind:    models.Kind(models.Kind_value[r.FormValue("kind")]),
		Id:      r.FormValue("id"),
	})
	if err != nil {
		log.Printf("webui.EditGET error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp.ServeHTTP(w, r)
}

// EditPOST handles a `POST` request to the `/records/edit/` routes of the records web UI.
//
// Parameters:
//		{
//			kind string
//			id string
//			<arbitrary-fields> {string|[]string}
//		}
//
// EditPOST updates the record by matching `kind` and `id` parameters, and dynamically pulls the approriate attributes
// from the form parameters.
//
// Success:
//		* StatusFound
//			- record updated, redirect to /records/view/
//
// Errors:
//		* StatusBadRequest
//			- missing kind
//			- unrecognized kind
//			- missing id
//			- invalid id
//		* StatusUnauthorized
//			- model not property
//			- access.CanWrite false
//		* StatusNotFound
//			- db.PopulateByID data.ErrNotFound
//		* StatusInternalServerError
//			- r.ParseForm error
//			- unmarshalling model
//			- db.PopulateByID error
//			- ctx missing user
//			- access.CanWrite error
//			- db.Save error
func EditPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, webui services.WebUIClient) {
	pu, pr := auth.CredentialsFromRequest(r)

	defer r.Body.Close()
	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp, err := webui.EditPOST(ctx, &records.EditPOSTRequest{
		Public:  pu,
		Private: pr,
		Url:     r.URL.String(),
		Body:    bytes,
	})

	if err != nil {
		log.Printf("webui.EditPOST error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp.ServeHTTP(w, r)
}
