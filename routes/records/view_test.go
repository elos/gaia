package records_test

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
	"github.com/elos/gaia/routes/records"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

func TestViewTemplate(t *testing.T) {
	b := new(bytes.Buffer)

	if err := records.ViewTemplate.Execute(b, records.ViewData{
		Flash: "this is the flash",
		Record: map[string]interface{}{
			"attr1": 1,
			"attr2": "string",
			"attr3": []string{"one", "two"},
			"attr4": map[string]int{
				"one": 1,
				"two": 2,
			},
		},
	}); err != nil {
		t.Fatalf("records.ViewTemplate.Execute error: %v", err)
	}

	t.Logf("Out:\n%s", b)
}

func TestViewGET(t *testing.T) {
	db := mem.WithData(map[data.Kind][]data.Record{
		models.UserKind: {
			&models.User{
				Id:             "1",
				CredentialsIds: []string{"2"},
			},
		},
		models.CredentialKind: {
			&models.Credential{
				Id:      "2",
				OwnerId: "1",
				Spec:    "password",
				Public:  "public",
				Private: "private",
			},
		},
	})

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		records.ViewGET(user.NewContext(context.Background(), &models.User{Id: "1"}), w, r, db, services.NewTestLogger(t))
	}))

	p := s.URL + "?" + url.Values{
		"kind": []string{"credential"},
		"id":   []string{"2"},
	}.Encode()

	resp, err := http.Get(p)
	if err != nil {
		t.Fatalf("http.Get(%q) error: %v", err)
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ioutil.ReadAll error: %v", err)
	}
	body := string(b)
	t.Logf("resp.Body:\n%s", body)

	contents := map[string]bool{
		"credential": true,
		"public":     true,
		"password":   true,
		"owner":      true,

		"user": false,
	}

	for content, want := range contents {
		if got := strings.Contains(body, content); got != want {
			t.Fatalf("strings.Contains(body, %q): got %t, want %t", content, got, want)
		}
	}
}
