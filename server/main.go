package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"ex8ed/pullreq-assigner/internal/handler"
	"ex8ed/pullreq-assigner/internal/service"
	"ex8ed/pullreq-assigner/internal/storage"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")

	var db *sqlx.DB
	var err error
	for i := 0; i < 5; i++ {
		db, err = sqlx.Connect("postgres", dbURL)
		if err == nil {
			break
		}
		log.Printf("Waiting for DB... (%d/5)", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatal("Could not connect to DB:", err)
	}

	repo := storage.New(db)
	svc := service.New(repo)
	h := handler.New(svc)

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("/health", h.Health)

	// Teams
	mux.HandleFunc("/team/add", h.CreateTeam)
	mux.HandleFunc("/team/get", h.GetTeam)

	// Users
	mux.HandleFunc("/users/setIsActive", h.SetUserActive)
	mux.HandleFunc("/users/getReview", h.GetUserReviews)

	// Pull Requests
	mux.HandleFunc("/pullRequest/create", h.CreatePR)
	mux.HandleFunc("/pullRequest/merge", h.MergePR)
	mux.HandleFunc("/pullRequest/reassign", h.ReassignReviewer)

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}