package routes_test

import (
	"bytes"
	"net/http"
	"net/http/cookiejar"
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

func TestLoginPOST(t *testing.T) {
	ctx := context.Background()
	db := mem.NewDB()
	logger := services.NewTestLogger(t)
	m := http.NewServeMux()
	m.Handle("/login/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := routes.Authenticate(ctx, w, r, logger, db)
		if !ok {
			t.Fatal("bad authentication")
		}
		routes.LoginPOST(ctx, w, r, db, logger)
	}))
	m.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	s := httptest.NewServer(http.HandlerFunc(m.ServeHTTP))
	defer s.Close()

	_, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatalf("user.Create(db, \"username\", \"password\") error: %s", err)
	}

	req, err := http.NewRequest("GET", s.URL+"/login/", new(bytes.Buffer))
	if err != nil {
		t.Fatalf("http.NewRequest error: %s", err)
	}
	req.SetBasicAuth("username", "password")

	jar, err := cookiejar.New(new(cookiejar.Options))
	if err != nil {
		t.Fatalf("cookiejar.New error: %s", err)
	}
	client := &http.Client{
		Jar: jar,
	}

	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("error posting to LoginPOST: %s", err)
	}

	t.Logf("Response:\n\t%v", resp)

	if got, want := resp.StatusCode, http.StatusAccepted; got != want {
		t.Errorf("resp.StatusCode: got %d, want %d", got, want)
	}

	iter, err := db.Query(models.SessionKind).Execute()
	if err != nil {
		t.Fatalf("db.Query(models.SessionKind).Execute(): %s", err)
	}

	seshes := mem.Slice(iter, func() data.Record { return new(models.Session) })

	if got, want := len(seshes), 1; got != want {
		t.Fatalf("len(seshes): got %d, want %d", got, want)
	}

	uri, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("url.Prase(s.URL) error: %s", err)
	}

	t.Logf("URL: %s", uri)

	cookies := jar.Cookies(uri)
	if got, want := len(cookies), 1; got != want {
		t.Fatalf("len(cookies): got %d, want %d", got, want)
	}

	if got, want := cookies[0].Value, seshes[0].(*models.Session).Token; got != want {
		t.Errorf("cookies[0].Value: got %s, want %s", got, want)
	}
}
