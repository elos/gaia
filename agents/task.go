package agents

import (
	"log"

	"github.com/elos/data"
	"github.com/elos/models"
	"github.com/elos/models/access"
	"github.com/elos/models/tag"
	"golang.org/x/net/context"
)

const (
	TaskMakeGoal = "TASK_MAKE_GOAL"
	TaskDropGoal = "TASK_DROP_GOAL"
)

func TaskAgent(ctx context.Context, db data.DB, u *models.User) {

	changes := data.Filter(data.FilterKind(db.Changes(), models.EventKind), func(c *data.Change) bool {
		ok, _ := access.CanRead(db, u, c.Record)
		return ok
	})

Run:
	for {
		select {
		case c, ok := <-*changes:
			if !ok {
				break Run
			}

			switch c.Record.(*models.Event).Name {
			case TaskMakeGoal:
				taskMakeGoal(db, u, c.Record.(*models.Event).Data)
			case TaskDropGoal:
				taskDropGoal(db, u, c.Record.(*models.Event).Data)
			}
		case <-ctx.Done():
			break Run
		}
	}

}

func taskMakeGoal(db data.DB, u *models.User, eventData map[string]interface{}) {
	g, err := tag.ForName(db, u, tag.Goal)
	if err != nil {
		log.Printf("agents.taskMakeGoal Error: %s", err)
		return
	}

	id, err := db.ParseID(eventData["task_id"].(string))
	if err != nil {
		log.Printf("agents.taskMakeGoal Error: %s", err)
		return
	}

	t, err := models.FindTask(db, id)
	if err != nil {
		log.Printf("agents.taskMakeGoal Error: %s", err)
		return
	}

	t.IncludeTag(g)

	if err := db.Save(t); err != nil {
		log.Printf("agents.taskMakeGoal Error: %s", err)
		return
	}
}

func taskDropGoal(db data.DB, u *models.User, eventData map[string]interface{}) {
	g, err := tag.ForName(db, u, tag.Goal)
	if err != nil {
		log.Printf("agents.taskDropGoal Error: %s", err)
		return
	}

	id, err := db.ParseID(eventData["task_id"].(string))
	if err != nil {
		log.Printf("agents.taskDropGoal Error: %s", err)
		return
	}

	t, err := models.FindTask(db, id)
	if err != nil {
		log.Printf("agents.taskMakeGoal Error: %s", err)
		return
	}

	t.ExcludeTag(g)

	if err := db.Save(t); err != nil {
		log.Printf("agents.taskMakeGoal Error: %s", err)
		return
	}
}
