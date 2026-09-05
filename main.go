package main

import (
	"log"
	"net/http"
	"time"

	"auxilia-webserver/internal/store"
)

type service struct{ store *store.Store }

func main() {
	db, err := openDatabase()
	if err != nil {
		log.Fatalf("database startup failed: %v", err)
	}
	repository, err := store.New(db)
	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	s := &service{store: repository}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		write(w, 200, map[string]string{"status": "ok", "database": "mariadb"})
	})
	mux.HandleFunc("GET /api/characters", s.characters)
	mux.HandleFunc("POST /api/guests", s.join)
	mux.HandleFunc("GET /api/me", s.auth(s.me))
	mux.HandleFunc("PUT /api/me/selection", s.auth(s.selection))
	mux.HandleFunc("POST /api/matchmaking", s.auth(s.matchmaking))
	mux.HandleFunc("DELETE /api/matchmaking", s.auth(s.cancel))
	mux.HandleFunc("GET /api/matches/{id}", s.auth(s.matchState))
	mux.HandleFunc("POST /api/matches/{id}/ready", s.auth(s.readyMatch))
	mux.HandleFunc("DELETE /api/matches/{id}/ready", s.auth(s.cancelReadyMatch))
	mux.HandleFunc("POST /api/matches/{id}/leave", s.auth(s.leaveMatch))
	mux.HandleFunc("POST /api/matches/{id}/move", s.auth(s.move))
	mux.HandleFunc("POST /api/matches/{id}/attack", s.auth(s.attack))
	mux.HandleFunc("POST /api/matches/{id}/end-turn", s.auth(s.endTurn))
	mux.HandleFunc("POST /api/matches/{id}/surrender", s.auth(s.surrender))
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for now := range ticker.C {
			if err := repository.Cleanup(now); err != nil {
				log.Printf("cleanup failed: %v", err)
			}
		}
	}()
	appPort := env("PORT", "8080")
	log.Printf("Auxilia webserver listening on :%s", appPort)
	log.Fatal(http.ListenAndServe(":"+appPort, cors(mux)))
}
