package records_test

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia/routes/records"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

func TestQueryTemplate(t *testing.T) {
	b := new(bytes.Buffer)

	if err := records.QueryTemplate.Execute(b, records.QueryData{
		Flash:         "this is the flash",
		QueryFormHTML: template.HTML(`<label for="attr"><input name="attr" type="text" value="val">`),
		Model:         models.Metis[models.EventKind],
		Records: []map[string]interface{}{
			{
				"attr": 1,
				"whatever": map[string]int{
					"one": 1,
				},
			},
		},
	}); err != nil {
		t.Fatalf("records.QueryTemplate.Execute error: %v", err)
	}

	t.Logf("Out:\n%s", b)
}

func TestQueryGET(t *testing.T) {
	db := mem.WithData(map[data.Kind][]data.Record{
		models.UserKind: []data.Record{
			&models.User{
				Id:             "1",
				CredentialsIds: []string{"2"},
			},
			&models.User{
				Id: "2",
			},
		},
		models.CredentialKind: []data.Record{
			&models.Credential{
				Id:        "3",
				CreatedAt: time.Now(),
				Spec:      "password",
				Public:    "username",
				Private:   "password",
			},
		},
		models.EventKind: []data.Record{
			&models.Event{
				Id:      "4",
				OwnerId: "1",
				Name:    "event 1",
			},
			&models.Event{
				Id:      "5",
				OwnerId: "1",
				Name:    "event 2",
			},
			&models.Event{
				Id:      "6",
				OwnerId: "1",
				Name:    "event 3",
			},
			&models.Event{
				Id:      "7",
				OwnerId: "1",
				Name:    "generic event",
			},
			&models.Event{
				Id:      "8",
				OwnerId: "1",
				Name:    "generic event",
			},

			// Not owned by User 1.
			&models.Event{
				Id:      "9",
				OwnerId: "2",
				Name:    "not owned",
			},
		},
	})

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		records.QueryGET(user.NewContext(context.Background(), &models.User{Id: "1"}), w, r, db, services.NewTestLogger(t))
	}))

	p := s.URL + "?" + url.Values{
		"query/Kind": []string{"event"},
	}.Encode()

	resp, err := http.Get(p)
	if err != nil {
		t.Fatalf("http.Get error: %v", err)
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	body := string(b)
	if err != nil {
		t.Fatalf("ioutil.ReadAll error: %v", err)
	}
	t.Logf("resp.Body:\n%s", body)

	contents := map[string]bool{
		"Query":         true,
		"5 results.":    true,
		"event 1":       true,
		"event 2":       true,
		"event 3":       true,
		"generic event": true,
		"Edit":          true,
		"/records/edit/?kind=event&id=8": true,
		"/records/edit/?kind=event&id=9": false,
		"not owned":                      false,
	}

	for content, want := range contents {
		if got := strings.Contains(body, content); got != want {
			t.Errorf("bytes.Contains(body, %q): got %t, want %t", content, got, want)
		}
	}
}
