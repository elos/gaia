package routes_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia/routes"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

// --- TestAuthenticateBasicAuth {{{

func TestAuthenticateBasicAuth(t *testing.T) {
	ctx := context.Background()
	db := mem.NewDB()
	logger := services.NewTestLogger(t)

	var userContext context.Context
	var authed bool
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userContext, authed = routes.Authenticate(ctx, w, r, logger, db)
		if authed {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer s.Close()

	u, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatalf("user.Create(db, \"username\", \"password\") error: %s", err)
	}
	client := new(http.Client)

	req, err := http.NewRequest("GET", s.URL, new(bytes.Buffer))
	if err != nil {
		t.Fatalf("http.NewRequest error: %s", err)
	}
	if _, err = client.Do(req); err != nil {
		t.Fatalf("client.Do(req) error: %s", err)
	}
	if got, want := authed, false; got != want {
		t.Errorf("authed: got %t, want %t", got, want)
	}

	req, err = http.NewRequest("GET", s.URL, new(bytes.Buffer))
	if err != nil {
		t.Fatalf("http.NewRequest error: %s", err)
	}
	req.SetBasicAuth("badusername", "password")
	if _, err = client.Do(req); err != nil {
		t.Fatalf("client.Do(req) error: %s", err)
	}
	if got, want := authed, false; got != want {
		t.Errorf("authed: got %t, want %t", got, want)
	}

	req, err = http.NewRequest("GET", s.URL, new(bytes.Buffer))
	if err != nil {
		t.Fatalf("http.NewRequest error: %s", err)
	}
	req.SetBasicAuth("badusername", "badpassword")
	if _, err = client.Do(req); err != nil {
		t.Fatalf("client.Do(req) error: %s", err)
	}
	if got, want := authed, false; got != want {
		t.Errorf("authed: got %t, want %t", got, want)
	}

	req, err = http.NewRequest("GET", s.URL, new(bytes.Buffer))
	if err != nil {
		t.Fatalf("http.NewRequest error: %s", err)
	}
	req.SetBasicAuth("username", "password")
	if _, err = client.Do(req); err != nil {
		t.Fatalf("client.Do(req) error: %s", err)
	}
	if got, want := authed, true; got != want {
		t.Fatalf("authed: got %t, want %t", got, want)
	}
	authedU, ok := user.FromContext(userContext)
	if got, want := ok, true; got != want {
		t.Fatalf("_, ok := user.FromContext: got %t, want %t", got, want)
	}
	if got, want := data.Equivalent(authedU, u), true; got != want {
		t.Errorf("data.Equivalent(authedU, u): got %t, want %t", got, want)
	}
}

// --- }}}

// --- TestAuthenticateSession {{{
func TestAuthenticateSession(t *testing.T) {
	ctx := context.Background()
	db := mem.NewDB()
	logger := services.NewTestLogger(t)

	var userContext context.Context
	var authed bool
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userContext, authed = routes.Authenticate(ctx, w, r, logger, db)
		if authed {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer s.Close()

	u, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatalf("user.Create(db, \"username\", \"password\") error: %s", err)
	}
	client := new(http.Client)

	req, err := http.NewRequest("GET", s.URL, new(bytes.Buffer))
	if err != nil {
		t.Fatalf("http.NewRequest error: %s", err)
	}
	req.AddCookie(&http.Cookie{
		Name:  "elos-session-token",
		Value: "garbage",
	})
	client.Do(req)

	if got, want := authed, false; got != want {
		t.Errorf("authed: got %t, want %t", got, want)
	}

	sesh := models.NewSessionForUser(u)
	if err := db.Save(sesh); err != nil {
		t.Fatalf("db.Save(sesh) error: %s", err)
	}
	req, err = http.NewRequest("GET", s.URL, new(bytes.Buffer))
	if err != nil {
		t.Fatalf("http.NewRequest error: %s", err)
	}
	req.AddCookie(&http.Cookie{
		Name:  "elos-session-token",
		Value: sesh.Token,
	})
	client.Do(req)

	if got, want := authed, true; got != want {
		t.Fatalf("authed: got %t, want %t", got, want)
	}
	authedU, ok := user.FromContext(userContext)
	if got, want := ok, true; got != want {
		t.Fatalf("_, ok := user.FromContext: got %t, want %t", got, want)
	}
	if got, want := data.Equivalent(authedU, u), true; got != want {
		t.Errorf("data.Equivalent(authedU, u): got %t, want %t", got, want)
	}
}

// --- }}}

// --- TestAuthenticateQueryParams {{{

func TestAuthenticateQueryParams(t *testing.T) {
	ctx := context.Background()
	db := mem.NewDB()
	logger := services.NewTestLogger(t)

	var userContext context.Context
	var authed bool
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userContext, authed = routes.Authenticate(ctx, w, r, logger, db)
		if authed {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer s.Close()

	u, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatalf("user.Create(db, \"username\", \"password\") error: %s", err)
	}
	client := new(http.Client)

	req, err := http.NewRequest("GET", s.URL, new(bytes.Buffer))
	if err != nil {
		t.Fatalf("http.NewRequest error: %s", err)
	}
	client.Do(req)
	if got, want := authed, false; got != want {
		t.Fatalf("authed: got %t, want %t", got, want)
	}

	req, err = http.NewRequest("GET", s.URL+"?public=username&private=password", new(bytes.Buffer))
	if err != nil {
		t.Fatalf("http.NewRequest error: %s", err)
	}
	client.Do(req)
	if got, want := authed, true; got != want {
		t.Fatalf("authed: got %t, want %t")
	}
	authedU, ok := user.FromContext(userContext)
	if got, want := ok, true; got != want {
		t.Fatalf("_, ok := user.FromContext: got %t, want %t")
	}
	if got, want := data.Equivalent(authedU, u), true; got != want {
		t.Errorf("data.Equivalent(authedU, u): got %t, want %t", got, want)
	}
}

// --- }}}

// --- TestAuthenticatePostForm {{{

func TestAuthenticatePostForm(t *testing.T) {
	ctx := context.Background()
	db := mem.NewDB()
	logger := services.NewTestLogger(t)

	var userContext context.Context
	var authed bool
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userContext, authed = routes.Authenticate(ctx, w, r, logger, db)
		if authed {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer s.Close()

	u, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatalf("user.Create(db, \"username\", \"password\") error: %s", err)
	}
	client := new(http.Client)
	client.PostForm(s.URL, url.Values{
		"public":  []string{"username"},
		"private": []string{"password"},
	})

	if got, want := authed, true; got != want {
		t.Fatalf("authed: got %t, want %t", got, want)
	}
	authedU, ok := user.FromContext(userContext)
	if got, want := ok, true; got != want {
		t.Fatalf("_, ok := user.FromContext: got %t, want %t", got, want)
	}
	if got, want := data.Equivalent(authedU, u), true; got != want {
		t.Fatal("data.Equivalent(authedU, u): got %t ,want %t", got, want)
	}
}

// --- }}}
