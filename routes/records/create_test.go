package records_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/elos/gaia/routes/records"
	"github.com/elos/metis"
	"github.com/elos/models"
)

func TestCreateTemplate(t *testing.T) {
	task := models.Metis[models.TaskKind]

	cd := &records.CreateData{
		Flash: "flash message",
		Model: task,
	}

	b := new(bytes.Buffer)
	if err := records.CreateTemplate.Execute(b, cd); err != nil {
		t.Fatalf("records.CreateTemplate.Execute error: %v", err)
	}

	out := b.String()
	t.Logf("Out:\n%s", out)

	for name := range task.Traits {
		if got, want := strings.Contains(out, name), true; got != want {
			t.Errorf("strings.Contains(out, %q): got %t, want %t", name, got, want)
		}

		p := task.Traits[name].Type
		if p == metis.BooleanList || p == metis.IntegerList || p == metis.StringList ||
			p == metis.DateTimeList || p == metis.IDList {
			numVar := fmt.Sprintf("num%s", name)
			if got, want := strings.Contains(out, numVar), true; got != want {
				t.Errorf("strings.Contains(out, %q): got %t, want %t", numVar, got, want)
			}
			varAdder := fmt.Sprintf("id=\"%sAdder\"", name)
			if got, want := strings.Contains(out, varAdder), true; got != want {
				t.Errorf("strings.Contains(out, %q): got %t, want %t", varAdder, got, want)
			}
			onclick := fmt.Sprintf("onclick=\"add%s()\"", name)
			if got, want := strings.Contains(out, onclick), true; got != want {
				t.Errorf("strings.Contains(out, %q): got %t, want %t", onclick, got, want)
			}
		}
	}

	for name, r := range task.Relations {
		if got, want := strings.Contains(out, name), true; got != want {
			t.Errorf("strings.Contains(out, %q): got %t, want %t", name, got, want)
		}

		if r.Multiplicity == metis.One {
			param := fmt.Sprintf("relation-%s-id", name)
			if got, want := strings.Contains(out, param), true; got != want {
				t.Errorf("strings.Contains(out, %q): got %t, want %t", param, got, want)
			}
		}

	}

}
