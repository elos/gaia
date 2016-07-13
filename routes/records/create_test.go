package records_test

import (
	"bytes"
	"html/template"
	"testing"

	"github.com/elos/gaia/routes/records"
	"github.com/elos/gaia/routes/records/form"
	"github.com/elos/models"
)

func TestCreateTemplate(t *testing.T) {
	task := new(models.Task)
	bs, err := form.Marshal(task, string(models.TaskKind))
	if err != nil {
		t.Fatalf("form.Marshal(task, string(models.TaskKind)) error: %v", err)
	}

	cd := &records.CreateData{
		Flash:    "flash message",
		FormHTML: template.HTML(string(bs)),
	}

	b := new(bytes.Buffer)
	if err := records.CreateTemplate.Execute(b, cd); err != nil {
		t.Fatalf("records.CreateTemplate.Execute error: %v", err)
	}

	out := b.String()
	t.Logf("Out:\n%s", out)
}
