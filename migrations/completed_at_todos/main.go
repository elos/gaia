package main

import (
	"flag"
	"log"

	"github.com/elos/models"
	"github.com/elos/models/task"
)

func main() {
	var (
		mongo = flag.String("mongo", "localhost", "Address of mongo instance")
	)

	db, err := models.MongoDB(*mongo)
	if err != nil {
		log.Fatal(err)
	}

	iter, err := db.Query(models.TaskKind).Execute()
	if err != nil {
		log.Fatal(err)
	}

	count := 0
	t := models.NewTask()
	for iter.Next(t) {
		if task.IsComplete(t) {
			t.CompletedAt = t.UpdatedAt

			if err := db.Save(t); err != nil {
				log.Fatal(err)
			}

			count++
		}
	}

	log.Print("Migrated %d tasks", count)

	if err := iter.Close(); err != nil {
		log.Fatal(err)
	}
}
