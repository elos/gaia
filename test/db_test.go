package test

import (
	"net/http"
	"testing"
	"time"

	"github.com/elos/data"
	"github.com/elos/gaia"
	"github.com/elos/models"
)

func TestDB(t *testing.T) {
	db, _, s := testInstance()
	defer s.Close()

	user, cred := testUser(t, db)

	gdb := &gaia.DB{
		URL:      s.URL,
		Username: cred.Public,
		Password: cred.Private,
		Client:   http.DefaultClient,
	}

	t.Log("Creating a test task")

	taskName := "task name"
	task := models.NewTask()
	task.SetID(db.NewID())
	task.CreatedAt = time.Now()
	task.OwnerId = user.Id
	task.Name = taskName
	task.UpdatedAt = time.Now()
	if err := db.Save(task); err != nil {
		t.Fatal(err)
	}

	t.Log("Test task created")

	t.Log("Creating a task to retrieve using the gaia db")

	getTask := models.NewTask()
	getTask.SetID(task.ID()) // use task's id

	t.Logf("Retrieving task using ID: %s and the gaia db", task.ID().String())
	if err := gdb.PopulateByID(getTask); err != nil {
		t.Fatal(err)
	}
	t.Logf("Task retrieved: %+v", getTask)

	if getTask.Name != taskName {
		t.Fatal("Task names should match")
	}

	t.Log("Deleting task using the gaia db")
	if err := gdb.Delete(getTask); err != nil {
		t.Fatal(err)
	}
	t.Log("Task deleted")

	t.Log("Checking that task is no longer in real database")
	if err := db.PopulateByID(task); err != data.ErrNotFound {
		t.Fatal("Task should be gone")
	}
	t.Log("Checked")

	t.Log("Using gaia db to recreate that task")
	if err := gdb.Save(task); err != nil {
		t.Fatal(err)
	}
	t.Log("Recreated")

	t.Log("Checking that task back in real database")
	if err := db.PopulateByID(task); err != nil {
		t.Fatal("Task should be back")
	}

	if task.Name != taskName {
		t.Fatalf("Task name should be %s", taskName)
	}

	t.Log("Checked")

	t.Log("Changing task name")
	newName := "new name"
	getTask.Name = newName
	t.Log("Changed")

	t.Log("Using gaia db to update the task")
	if err := gdb.Save(getTask); err != nil {
		t.Fatal(err)
	}
	t.Log("Updated")

	t.Log("Re-retrieving that task from the real db")
	if err := db.PopulateByID(task); err != nil {
		t.Fatal(err)
	}
	t.Log("Retrieved")

	t.Log("Checking that the task was updated")
	if task.Name != newName {
		t.Fatalf("Save should have updated the task")
	}
	t.Log("Checked")

	t.Log("Querying for the task using gaia db")
	q := gdb.Query(models.TaskKind)
	q.Select(data.AttrMap{
		"name": newName,
	})
	iter, err := q.Execute()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Query executed")

	t.Log("Examining quert results")

	result := models.NewTask()
	iter.Next(result)
	if iter.Next(result) != false {
		t.Fatal("Should only be one result")
	}

	if err := iter.Close(); err != nil {
		t.Fatal(err)
	}

	if result.ID().String() != task.ID().String() {
		t.Fatal("Ids should match")
	}

	if result.Name != newName {
		t.Fatal("should ahve retrievied the task with the new name")
	}

	t.Log("Examined")
}
