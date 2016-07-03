package records_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/elos/gaia/routes/records"
	"github.com/elos/models"
)

func TestNewTemplate(t *testing.T) {
	nd := &records.NewData{
		Models: models.Metis,
	}
	b := new(bytes.Buffer)

	if err := records.NewTemplate.Execute(b, nd); err != nil {
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
