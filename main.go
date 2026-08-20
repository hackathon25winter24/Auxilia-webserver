package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"auxilia-webserver/internal/game"
	"auxilia-webserver/internal/store"
	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type guestResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Token     string   `json:"token,omitempty"`
	Selection []string `json:"selection"`
	MatchID   string   `json:"matchId,omitempty"`
	Queued    bool     `json:"queued"`
}
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

func openDatabase() (*gorm.DB, error) {
	user := env("NS_MARIADB_USER", "auxilia_user")
	password := env("NS_MARIADB_PASSWORD", "auxilia_password")
	host := env("NS_MARIADB_HOSTNAME", "127.0.0.1")
	port := env("NS_MARIADB_PORT", "3306")
	database := env("NS_MARIADB_DATABASE", "auxilia_web")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC", user, password, host, port, database)
	db, err := gorm.Open(gormMysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), TranslateError: true})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	maxOpen := envInt("DB_MAX_OPEN_CONNS", 5)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(min(maxOpen, 2))
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}
	return db, nil
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
	token := id("token")
	g, err := s.store.CreateGuest(id("guest"), in.Name, token)
	if err != nil {
		serverError(w, err)
		return
	}
	response := guestDTO(g)
	response.Token = token
	write(w, 201, response)
}
func (s *service) me(w http.ResponseWriter, r *http.Request, g *store.Guest) {
	latest, err := s.store.GuestByID(g.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	write(w, 200, guestDTO(latest))
}
func (s *service) selection(w http.ResponseWriter, r *http.Request, g *store.Guest) {
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
	updated, err := s.store.SetSelection(g.ID, in.CharacterIDs)
	if err != nil {
		problem(w, 409, err.Error())
		return
	}
	write(w, 200, guestDTO(updated))
}
func (s *service) matchmaking(w http.ResponseWriter, r *http.Request, g *store.Guest) {
	updated, err := s.store.EnqueueAndMatch(g.ID, id("match"))
	if err != nil {
		problem(w, 409, err.Error())
		return
	}
	write(w, 200, guestDTO(updated))
}
func (s *service) cancel(w http.ResponseWriter, r *http.Request, g *store.Guest) {
	updated, err := s.store.CancelQueue(g.ID)
	if err != nil {
		problem(w, 409, err.Error())
		return
	}
	write(w, 200, guestDTO(updated))
}
func (s *service) matchState(w http.ResponseWriter, r *http.Request, g *store.Guest) {
	state, err := s.store.LoadState(r.PathValue("id"), g.ID)
	if err != nil {
		storeError(w, err)
		return
	}
	write(w, 200, state)
}
func (s *service) move(w http.ResponseWriter, r *http.Request, g *store.Guest) {
	var c game.Command
	if decode(w, r, &c) != nil {
		return
	}
	s.apply(w, r, g, c, func(st *game.State) error { return st.ApplyMove(g.ID, c) })
}
func (s *service) attack(w http.ResponseWriter, r *http.Request, g *store.Guest) {
	var c game.Command
	if decode(w, r, &c) != nil {
		return
	}
	s.apply(w, r, g, c, func(st *game.State) error { return st.ApplyAttack(g.ID, c) })
}
func (s *service) endTurn(w http.ResponseWriter, r *http.Request, g *store.Guest) {
	var c game.Command
	if decode(w, r, &c) != nil {
		return
	}
	s.apply(w, r, g, c, func(st *game.State) error { return st.EndTurn(g.ID, c.ExpectedRevision) })
}
func (s *service) apply(w http.ResponseWriter, r *http.Request, g *store.Guest, c game.Command, fn func(*game.State) error) {
	if c.ID == "" {
		problem(w, 400, "commandIdが必要です")
		return
	}
	state, err := s.store.Apply(r.PathValue("id"), g.ID, c.ID, fn)
	if err != nil {
		code := 422
		if errors.Is(err, game.ErrStaleRevision) {
			code = 409
		}
		if errors.Is(err, store.ErrNotFound) {
			code = 404
		}
		problem(w, code, err.Error())
		return
	}
	write(w, 200, state)
}
func (s *service) auth(next func(http.ResponseWriter, *http.Request, *store.Guest)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			problem(w, 401, "セッションが無効です")
			return
		}
		g, err := s.store.GuestByToken(token)
		if err != nil {
			problem(w, 401, "セッションが無効です")
			return
		}
		next(w, r, g)
	}
}

func guestDTO(g *store.Guest) guestResponse {
	return guestResponse{ID: g.ID, Name: g.Name, Selection: g.Selection(), MatchID: g.MatchID, Queued: g.Queued}
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
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
func serverError(w http.ResponseWriter, err error) {
	log.Printf("server error: %v", err)
	problem(w, 500, "サーバー処理に失敗しました")
}
func storeError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		problem(w, 404, "データが見つかりません")
		return
	}
	serverError(w, err)
}
func id(prefix string) string {
	bytes := make([]byte, 12)
	_, _ = rand.Read(bytes)
	return prefix + "-" + hex.EncodeToString(bytes)
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
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
func cors(next http.Handler) http.Handler {
	allowed := map[string]bool{
		"http://localhost:3000":                 true,
		"http://localhost:5173":                 true,
		"http://127.0.0.1:3000":                 true,
		"https://hackathon25winter24.github.io": true,
	}
	for _, origin := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			allowed[origin] = true
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		w.Header().Add("Vary", "Origin")
		if allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
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
