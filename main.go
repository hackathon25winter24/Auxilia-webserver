package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"auxilia-webserver/internal/game"
)

type guest struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Token     string   `json:"token,omitempty"`
	Selection []string `json:"selection"`
	MatchID   string   `json:"matchId,omitempty"`
	Queued    bool     `json:"queued"`
}
type service struct {
	mu       sync.RWMutex
	guests   map[string]*guest
	tokens   map[string]string
	matches  map[string]*game.State
	queue    []string
	commands map[string]bool
}

func main() {
	s := &service{guests: map[string]*guest{}, tokens: map[string]string{}, matches: map[string]*game.State{}, commands: map[string]bool{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) { write(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /api/characters", characters)
	mux.HandleFunc("POST /api/guests", s.join)
	mux.HandleFunc("GET /api/me", s.auth(s.me))
	mux.HandleFunc("PUT /api/me/selection", s.auth(s.selection))
	mux.HandleFunc("POST /api/matchmaking", s.auth(s.matchmaking))
	mux.HandleFunc("DELETE /api/matchmaking", s.auth(s.cancel))
	mux.HandleFunc("GET /api/matches/{id}", s.auth(s.matchState))
	mux.HandleFunc("POST /api/matches/{id}/move", s.auth(s.move))
	mux.HandleFunc("POST /api/matches/{id}/attack", s.auth(s.attack))
	mux.HandleFunc("POST /api/matches/{id}/end-turn", s.auth(s.endTurn))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Printf("Auxilia webserver listening on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, cors(mux)))
}
func characters(w http.ResponseWriter, r *http.Request) { write(w, 200, game.Definitions) }
func (s *service) join(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if decode(w, r, &in) != nil {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if len([]rune(in.Name)) < 1 || len([]rune(in.Name)) > 20 {
		problem(w, 400, "名前は1〜20文字で入力してください")
		return
	}
	g := &guest{ID: id("guest"), Name: in.Name, Token: id("token"), Selection: []string{}}
	s.mu.Lock()
	s.guests[g.ID] = g
	s.tokens[g.Token] = g.ID
	s.mu.Unlock()
	write(w, 201, g)
}
func (s *service) me(w http.ResponseWriter, r *http.Request, g *guest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := *g
	copy.Token = ""
	write(w, 200, copy)
}
func (s *service) selection(w http.ResponseWriter, r *http.Request, g *guest) {
	var in struct {
		CharacterIDs []string `json:"characterIds"`
	}
	if decode(w, r, &in) != nil {
		return
	}
	if len(in.CharacterIDs) != 3 || hasDuplicates(in.CharacterIDs) {
		problem(w, 400, "異なるキャラクターを3体選択してください")
		return
	}
	for _, v := range in.CharacterIDs {
		if _, ok := game.Definition(v); !ok {
			problem(w, 400, "不明なキャラクターです")
			return
		}
	}
	s.mu.Lock()
	if g.Queued || g.MatchID != "" {
		s.mu.Unlock()
		problem(w, 409, "マッチング中は変更できません")
		return
	}
	g.Selection = append([]string(nil), in.CharacterIDs...)
	s.mu.Unlock()
	write(w, 200, g)
}
func (s *service) matchmaking(w http.ResponseWriter, r *http.Request, g *guest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if g.MatchID != "" {
		write(w, 200, g)
		return
	}
	if !g.Queued {
		if len(g.Selection) != 3 {
			problem(w, 400, "キャラクターを3体選択してください")
			return
		}
		s.queue = append(s.queue, g.ID)
		g.Queued = true
	}
	if len(s.queue) >= 2 {
		a, b := s.guests[s.queue[0]], s.guests[s.queue[1]]
		s.queue = s.queue[2:]
		mid := id("match")
		st := game.NewState(mid, [2]game.Player{{ID: a.ID, Name: a.Name}, {ID: b.ID, Name: b.Name}}, [2][]string{a.Selection, b.Selection})
		s.matches[mid] = st
		a.MatchID = mid
		b.MatchID = mid
		a.Queued = false
		b.Queued = false
	}
	write(w, 200, g)
}
func (s *service) cancel(w http.ResponseWriter, r *http.Request, g *guest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if g.MatchID != "" {
		problem(w, 409, "開始済みの試合は解除できません")
		return
	}
	next := s.queue[:0]
	for _, v := range s.queue {
		if v != g.ID {
			next = append(next, v)
		}
	}
	s.queue = next
	g.Queued = false
	write(w, 200, g)
}
func (s *service) matchState(w http.ResponseWriter, r *http.Request, g *guest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.matches[r.PathValue("id")]
	if st == nil || g.MatchID != st.MatchID {
		problem(w, 404, "試合が見つかりません")
		return
	}
	st.ExpireTurn(time.Now())
	write(w, 200, st)
}
func (s *service) move(w http.ResponseWriter, r *http.Request, g *guest) {
	var c game.Command
	if decode(w, r, &c) != nil {
		return
	}
	s.apply(w, r, g, c, func(st *game.State) error { return st.ApplyMove(g.ID, c) })
}
func (s *service) attack(w http.ResponseWriter, r *http.Request, g *guest) {
	var c game.Command
	if decode(w, r, &c) != nil {
		return
	}
	s.apply(w, r, g, c, func(st *game.State) error { return st.ApplyAttack(g.ID, c) })
}
func (s *service) endTurn(w http.ResponseWriter, r *http.Request, g *guest) {
	var c game.Command
	if decode(w, r, &c) != nil {
		return
	}
	s.apply(w, r, g, c, func(st *game.State) error { return st.EndTurn(g.ID, c.ExpectedRevision) })
}
func (s *service) apply(w http.ResponseWriter, r *http.Request, g *guest, c game.Command, fn func(*game.State) error) {
	if c.ID == "" {
		problem(w, 400, "commandIdが必要です")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.matches[r.PathValue("id")]
	if st == nil || g.MatchID != st.MatchID {
		problem(w, 404, "試合が見つかりません")
		return
	}
	st.ExpireTurn(time.Now())
	key := st.MatchID + ":" + g.ID + ":" + c.ID
	if s.commands[key] {
		write(w, 200, st)
		return
	}
	if err := fn(st); err != nil {
		code := 422
		if errors.Is(err, game.ErrStaleRevision) {
			code = 409
		}
		problem(w, code, err.Error())
		return
	}
	s.commands[key] = true
	write(w, 200, st)
}
func (s *service) auth(next func(http.ResponseWriter, *http.Request, *guest)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		s.mu.RLock()
		g := s.guests[s.tokens[token]]
		s.mu.RUnlock()
		if g == nil {
			problem(w, 401, "セッションが無効です")
			return
		}
		next(w, r, g)
	}
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		problem(w, 400, "リクエスト形式が不正です")
		return err
	}
	return nil
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, message string) {
	write(w, status, map[string]string{"error": message})
}
func id(prefix string) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}
func hasDuplicates(values []string) bool {
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		if seen[value] {
			return true
		}
		seen[value] = true
	}
	return false
}
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:3000" || origin == "http://localhost:5173" || origin == "http://127.0.0.1:3000" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
