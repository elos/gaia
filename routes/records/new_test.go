package records_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia/routes/records"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"golang.org/x/net/context"
)

func TestNewTemplate(t *testing.T) {
	b := new(bytes.Buffer)

	if err := records.NewTemplate.Execute(b, &records.NewData{
		Models: models.Metis,
	}); err != nil {
		t.Fatalf("records.NewTemplate.Execute error: %v", err)
	}

	out := b.String()
	t.Logf("Out:\n%s", out)

	for name := range models.Metis {
		if got, want := strings.Contains(out, string(name)), true; got != want {
			t.Errorf("strings.Contains(out, %q): got %t, want %t", name, got, want)
		}
	}
}

func TestNewGET(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		records.NewGET(context.Background(), w, r, mem.NewDB(), services.NewTestLogger(t))
	}))

	resp, err := http.Get(s.URL)
	if err != nil {
		t.Fatalf("http.Get error: %v", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ioutil.ReadAll error: %v", err)
	}
	t.Logf("resp.Body:\n%s", body)

	for name := range models.Metis {
		if name == "user" {
			continue
		}

		if got, want := bytes.Contains(body, []byte(name)), true; got != want {
			t.Errorf("strings.Contains(out, %q): got %t, want %t", name, got, want)
		}
	}
}
