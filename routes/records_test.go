package routes

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/data/transfer"
	"github.com/elos/gaia/services"
	"github.com/elos/metis"
	"github.com/elos/models"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

// --- Test /records/ {{{

// --- TestRecordsTemplate {{{

func TestRecordsTemplate(t *testing.T) {
	db := mem.NewDB()

	creds := make([]map[string]interface{}, 5)
	for i := 0; i < 5; i++ {
		_, c, err := user.Create(db, "username", "password")
		if err != nil {
			t.Fatal("user.Create error: %s", err)
		}

		m := make(map[string]interface{})
		transfer.TransferAttrs(c, &m)
		creds[i] = m
	}
	s := &records{
		Kind:    models.CredentialKind,
		Model:   models.Metis[models.CredentialKind],
		Records: creds,
	}

	var b bytes.Buffer
	if err := recordsTemplate.Execute(&b, s); err != nil {
		t.Fatalf("template.Execute error: %s", err)
	}

	o := b.String()
	t.Logf("TemplateOutput:\n%s", o)

	if got, want := strings.Contains(o, "username"), true; got != want {
		t.Errorf("strings.Contains(\"username\"): got %t, want %t", got, want)
	}
	if got, want := strings.Contains(o, "password"), true; got != want {
		t.Errorf("strings.Contains(\"password\"): got %t, want %t", got, want)
	}
}

// --- }}}

// --- TestRecordsGET {{{

func TestRecordsGET(t *testing.T) {
	ctx := context.Background()
	db := mem.NewDB()
	logger := services.NewTestLogger(t)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := Authenticate(ctx, w, r, logger, db)
		if !ok {
			t.Fatal("authentication failed")
		}

		RecordsGET(ctx, w, r, db, logger)
	}))
	defer s.Close()

	_, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatal(err)
	}

	c := new(http.Client)
	req, err := http.NewRequest("GET", s.URL+"?"+url.Values{
		"kind": []string{"credential"},
	}.Encode(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("username", "password")

	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Body:\n%s", body)
	doc := string(body)

	if got, want := strings.Contains(doc, "username"), true; got != want {
		t.Fatalf("strings.Contains(doc, \"username\"): got %t, want %t", got, want)
	}
	if got, want := strings.Contains(doc, "password"), true; got != want {
		t.Fatalf("strings.Contains(doc, \"password\"): got %t, want %t", got, want)
	}
}

// --- }}}

// --- }}}

// --- Test /records/new/ {{{

// --- TestRecordsNewTemplate {{{

func TestRecordsNewTemplate(t *testing.T) {
	rn := &recordsNew{
		Flash: "this is the flash",
		Models: map[data.Kind]*metis.Model{
			models.UserKind:       models.Metis[models.UserKind],
			models.CredentialKind: models.Metis[models.CredentialKind],
			models.EventKind:      models.Metis[models.EventKind],
		},
	}

	b := new(bytes.Buffer)
	if err := recordsNewTemplate.Execute(b, rn); err != nil {
		t.Fatalf("recordsNewTemplate.Execute error: %s", err)
	}

	o := b.String()
	t.Logf("Output:\n%s", o)

	contains := []string{
		"this is the flash",
		"user",
		"credential",
		"event",
		"New",
	}

	for _, c := range contains {
		if got, want := strings.Contains(o, c), true; got != want {
			t.Errorf("strings.Contains(%q): got %b, want %b", c, got, want)
		}

	}
}

// --- }}}

// --- TestRecordsNewGET {{{

func TestRecordsNewGET(t *testing.T) {
	ctx := context.Background()
	db := mem.NewDB()
	logger := services.NewTestLogger(t)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := Authenticate(ctx, w, r, logger, db)
		if !ok {
			t.Fatal("authentication failed")
		}

		RecordsNewGET(ctx, w, r, db, logger)
	}))
	defer s.Close()

	_, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatal(err)
	}

	c := new(http.Client)
	req, err := http.NewRequest("GET", s.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("username", "password")

	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Body:\n%s", body)
	doc := string(body)

	contains := map[string]bool{
		"user":  false,
		"event": true,
		"New":   true,
	}

	for c, want := range contains {
		if got := strings.Contains(doc, c); got != want {
			t.Fatalf("strings.Contains(%q): got %t, want %t", c, got, want)
		}
	}
}

// --- }}}

// --- }}}

// --- Test /records/edit/ {{{

// --- TestRecordsEditTemplate {{{

func TestRecordsEditTemplate(t *testing.T) {
	re := &recordsEdit{
		Flash: "this is the flash",
		Model: models.Metis[models.EventKind],
		Record: map[string]interface{}{
			"name": "this is the name",
		},
	}

	b := new(bytes.Buffer)
	if err := recordsEditTemplate.Execute(b, re); err != nil {
		t.Fatalf("recordsEditTemplate.Execute error: %s", err)
	}

	o := b.String()
	t.Logf("Output:\n%s", o)

	contains := map[string]bool{
		"this is the flash": true,
		"name":              true,
		"this is the name":  true,
		"credential":        false,
	}

	for c, want := range contains {
		if got := strings.Contains(o, c); got != want {
			t.Errorf("strings.Contains(%q): got %b, want %b", c, got, want)
		}

	}
}

// --- }}}

// --- TestRecordsEditGET {{{

func TestRecordsEditGET(t *testing.T) {
	ctx := context.Background()
	db := mem.NewDB()
	logger := services.NewTestLogger(t)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := Authenticate(ctx, w, r, logger, db)
		if !ok {
			t.Fatal("authentication failed")
		}

		RecordsEditGET(ctx, w, r, db, logger)
	}))
	defer s.Close()

	u, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatal(err)
	}

	e := new(models.Event)
	e.SetID(db.NewID())
	e.SetOwner(u)
	e.Name = "eventname"
	if err := db.Save(e); err != nil {
		t.Fatal("db.Save(e) error: %s", err)
	}

	c := new(http.Client)
	req, err := http.NewRequest("GET", s.URL+"?"+url.Values{
		"kind": []string{string(models.EventKind)},
		"id":   []string{e.Id},
	}.Encode(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("username", "password")

	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Body:\n%s", body)
	doc := string(body)

	contains := map[string]bool{
		"eventname": true,
		"name":      true,
	}

	for c, want := range contains {
		if got := strings.Contains(doc, c); got != want {
			t.Fatalf("strings.Contains(%q): got %t, want %t", c, got, want)
		}
	}
}

// --- }}}

// --- }}}
