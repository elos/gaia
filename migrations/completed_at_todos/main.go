package main

import (
	"flag"
	"log"

	"github.com/elos/models"
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
		t.CompletedAt = t.UpdatedAt
		count++
	}

	log.Print("Migrated %d tasks", count)

	if err := iter.Close(); err != nil {
		log.Fatal(err)
	}
}
