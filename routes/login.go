package routes

import (
	"net/http"
	"text/template"

	"github.com/elos/data"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

const (
	sessionCookie = "elos-session-token"
)

func cookie(s *models.Session) *http.Cookie {
	return &http.Cookie{
		Name:    sessionCookie,
		Value:   s.Token,
		Expires: s.Expires(),
		Path:    "/",
	}
}

func session(r *http.Request, db data.DB) (*models.Session, error) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil, err
	}
	return models.SessionForToken(db, c.Value)
}

func LoginPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	l := logger.WithPrefix("LoginPOST: ")

	u, ok := user.FromContext(ctx)
	if !ok {
		l.Print("failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	s := models.NewSessionForUser(u)

	if err := db.Save(s); err != nil {
		l.Print("failed to save session: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, cookie(s))
	http.Redirect(w, r, "/", http.StatusFound)
}

const loginTemplateRaw = `
<html>
	<body>
		<form method="post">
			<fieldset>
				<legend>Login:</legend>
				<input type="text" name="public"  placeholder="public"  />
				<input type="text" name="private" placeholder="private" />
				<input type="submit" />
			</fieldset>
		</form>
		<a href="/register/">Register</a>
	</body>
</html>
`

var loginTemplate = template.Must(template.New("login").Parse(loginTemplateRaw))

func LoginGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	l := logger.WithPrefix("LoginGET: ")

	if err := loginTemplate.Execute(w, nil); err != nil {
		l.Fatal(err)
	}
}

func AppHomeGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	// l := logger.WithPrefix("AppHomeGET: ")
}
