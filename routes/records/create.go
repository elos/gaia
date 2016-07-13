package records

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/elos/data"
	"github.com/elos/gaia/routes/records/form"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/access"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

const createTemplateRaw = `
<html>
	<body>
		{{ with .Flash }}
			{{ . }}
		{{ end }}

		<form method="post">
		{{ with .FormHTML }}
			{{ . }}
		{{ end }}
			<input type="submit" value="Create">
		</form>
	</body>
</html>
`

var CreateTemplate = template.Must(
	template.New("records/create").Funcs(template.FuncMap{
		"castJS": func(s string) template.JS {
			return template.JS(s)
		},
	}).Parse(createTemplateRaw),
)

type CreateData struct {
	Flash    string
	FormHTML template.HTML
}

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
//		* StatusOK, return web frame
//
// Errors:
//		* StatusBadRequest, {no kind parameter found, kind not recognized}
//		* StatusInternalServerError, {error parsing, et al.}
func CreateGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, l services.Logger) {
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

	m := models.ModelFor(kind)
	b, err := form.Marshal(m, string(kind))
	if err != nil {
		l.Printf("form.Marshal error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	cd := &CreateData{
		Flash:    "The record has not yet been created, you must save",
		FormHTML: template.HTML(string(b)),
	}

	if err := CreateTemplate.Execute(w, cd); err != nil {
		l.Fatalf("CreateTemplate.Execute error: %v", err)
	}
}

// CreatePOST handles a `POST` request to the `/records/create/` route of the records web UI.
//
// Parameters:
//	{
//		kind string
//		<arbitrary-fields> {string|[]string}
//	}
//
// CreatePOST creates the record matching the `kind`, and dynamically pulls the appropriate attributes
// from the form parameters.
//
// Success:
//		* StatusFound, redirect to `/records/view/?kind=<dynamic>&id=<dynamic>` (i.e., redirects to viewing the record
//		which was just created
//
// Errors:
//		* 400, malformed parameters
//		* 404, kind parameter not recognized
//		* 500, {error parsing, et al.}
func CreatePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, l services.Logger) {
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

	m := models.ModelFor(kind)

	if err := form.Unmarshal(r.Form, m, string(kind)); err != nil {
		l.Printf("info: r.Form :\n%v", r.Form)
		l.Printf("error: while unmarshalling form, %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	m.SetID(db.NewID())

	// Retrieve our user
	u, ok := user.FromContext(ctx)
	if !ok {
		l.Print("failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	prop, ok := m.(access.Property)
	if !ok {
		l.Printf("tried to create record that isn't property")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	allowed, err := access.CanCreate(db, u, prop)

	if err != nil {
		l.Printf("access.CanCreate error: %s", err)
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
