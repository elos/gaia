package agents_test

import (
	"log"
	"testing"
	"time"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia/agents"
	"github.com/elos/models"
	"github.com/elos/models/event"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

func TestTaskAgent(t *testing.T) {
	db := mem.NewDB()
	u, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatal(err)
	}
	tsk := models.NewTask()
	tsk.SetID(db.NewID())
	tsk.SetOwner(u)
	if err := db.Save(tsk); err != nil {
		t.Fatal(err)
	}

	changes := data.FilterKind(db.Changes(), models.TaskKind)

	ctx, stop := context.WithCancel(context.Background())
	go agents.TaskAgent(ctx, db, u)
	defer stop()

	// give control to agent thread
	time.Sleep(10 * time.Millisecond)

	_, err = event.TaskMakeGoal(db, tsk)
	if err != nil {
		log.Fatal(err)
	}

	select {
	case taskChange := <-*changes:
		changedTask := taskChange.Record.(*models.Task)

		t.Logf("Changed task:\n%+v", changedTask)

		if !data.Equivalent(tsk, changedTask) {
			t.Fatal("Expected the changed task to be the same one")
		}

		if len(changedTask.TagsIds) != 1 {
			t.Fatal("Expected changed task to have 1 tag")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for task update")
	}
}
