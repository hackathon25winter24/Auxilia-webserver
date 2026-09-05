package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"auxilia-webserver/internal/game"
	"auxilia-webserver/internal/store"
)

type guestResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Token     string   `json:"token,omitempty"`
	Selection []string `json:"selection"`
	MatchID   string   `json:"matchId,omitempty"`
	Queued    bool     `json:"queued"`
}

type characterResponse struct {
	game.CharacterDefinition
	UsageCount     uint64 `json:"usageCount"`
	TotalPickCount uint64 `json:"totalPickCount"`
}

func (s *service) characters(w http.ResponseWriter, r *http.Request) {
	counts, totalPickCount, err := s.store.CharacterUsageCounts()
	if err != nil {
		serverError(w, err)
		return
	}
	write(w, 200, characterResponses(counts, totalPickCount))
}

func characterResponses(counts map[string]uint64, totalPickCount uint64) []characterResponse {
	response := make([]characterResponse, 0, len(game.Definitions))
	for _, definition := range game.Definitions {
		count := counts[definition.ID]
		response = append(response, characterResponse{CharacterDefinition: definition, UsageCount: count, TotalPickCount: totalPickCount})
	}
	return response
}
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
func (s *service) readyMatch(w http.ResponseWriter, r *http.Request, g *store.Guest) {
	state, err := s.store.ReadyMatch(r.PathValue("id"), g.ID)
	if err != nil {
		storeError(w, err)
		return
	}
	write(w, 200, state)
}
func (s *service) cancelReadyMatch(w http.ResponseWriter, r *http.Request, g *store.Guest) {
	state, err := s.store.CancelReadyMatch(r.PathValue("id"), g.ID)
	if err != nil {
		if errors.Is(err, game.ErrInvalidAction) {
			problem(w, 409, "開始済みの試合はキャンセルできません")
			return
		}
		storeError(w, err)
		return
	}
	write(w, 200, state)
}
func (s *service) leaveMatch(w http.ResponseWriter, r *http.Request, g *store.Guest) {
	updated, err := s.store.LeaveFinishedMatch(r.PathValue("id"), g.ID)
	if err != nil {
		if errors.Is(err, game.ErrInvalidAction) {
			problem(w, 409, "終了していない試合からは退出できません")
			return
		}
		storeError(w, err)
		return
	}
	write(w, 200, guestDTO(updated))
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
func (s *service) surrender(w http.ResponseWriter, r *http.Request, g *store.Guest) {
	var c game.Command
	if decode(w, r, &c) != nil {
		return
	}
	s.apply(w, r, g, c, func(st *game.State) error { return st.Surrender(g.ID, c.ExpectedRevision) })
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
