package main

import (
	"auxilia-webserver/internal/store"
	"net/http"
)

func (s *service) heartbeat(w http.ResponseWriter, r *http.Request, g *store.Guest) {
	if err := s.store.Heartbeat(g.ID); err != nil {
		serverError(w, err)
		return
	}
	s.activeCount(w, r)
}
func (s *service) activeCount(w http.ResponseWriter, r *http.Request) {
	count, err := s.store.ActiveCount()
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	write(w, 200, map[string]int64{"count": count})
}
func (s *service) leavePresence(w http.ResponseWriter, r *http.Request, g *store.Guest) {
	if err := s.store.LeavePresence(g.ID); err != nil {
		serverError(w, err)
		return
	}
	write(w, 200, map[string]bool{"ok": true})
}
