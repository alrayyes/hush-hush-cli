// Package testserver is a small in-memory, stateful stand-in for the
// hush-hush server's HTTP API - internal/client's actual transport
// contract (api/openapi.yaml's /objects endpoints), not hush-hush's own
// internal/api implementation, which stays in that repo. It exists so
// internal/cli and cmd/hush-hush-cli's tests can exercise a real
// create/get/update/delete round trip - including auth, 404, and 409
// semantics against live state - without a second repo's server in the
// loop (openspec/changes/split-cli-into-own-repo/design.md: a Prism
// spec-mock can't reproduce that state, so this fake carries it itself).
package testserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ErrNotFound is returned by Store.GetObject, Store.UpdateObject, and
// Store.DeleteObject for an unknown id.
var ErrNotFound = errors.New("object not found")

// ErrAlreadyExists is returned by Store.CreateObject for an id already
// stored.
var ErrAlreadyExists = errors.New("object already exists")

// Object is a stored object's current state.
type Object struct {
	Value       []byte
	UsedBy      []string
	Description string
}

// Store is an in-memory object store plus write-token issuance, backing a
// Server started by New. Safe for concurrent use.
type Store struct {
	mu      sync.Mutex
	objects map[string]Object
	tokens  map[string]time.Time
}

func newStore() *Store {
	return &Store{objects: make(map[string]Object), tokens: make(map[string]time.Time)}
}

// CreateObject stores value under id, or ErrAlreadyExists if id is taken.
func (s *Store) CreateObject(_ context.Context, id string, value []byte, usedBy []string, description string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.objects[id]; ok {
		return ErrAlreadyExists
	}

	s.objects[id] = Object{Value: value, UsedBy: usedBy, Description: description}

	return nil
}

// GetObject returns id's current stored state, or ErrNotFound.
func (s *Store) GetObject(_ context.Context, id string) (Object, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, ok := s.objects[id]
	if !ok {
		return Object{}, ErrNotFound
	}

	return obj, nil
}

// UpdateObject replaces id's stored value, leaving used_by and
// description unchanged, or ErrNotFound.
func (s *Store) UpdateObject(_ context.Context, id string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, ok := s.objects[id]
	if !ok {
		return ErrNotFound
	}

	obj.Value = value
	s.objects[id] = obj

	return nil
}

// DeleteObject removes id, or ErrNotFound.
func (s *Store) DeleteObject(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.objects[id]; !ok {
		return ErrNotFound
	}

	delete(s.objects, id)

	return nil
}

// CreateWriteToken issues a fresh write token valid for ttl. The first
// return value mirrors hush-hush's own store.CreateWriteToken (an issued
// token's id); this fake has no separate use for it.
func (s *Store) CreateWriteToken(_ context.Context, name string, ttl time.Duration) (id, token string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token = name + "-" + randomSuffix()
	s.tokens[token] = time.Now().Add(ttl)

	return "", token, nil
}

// randomSuffix never errors in practice - crypto/rand.Read only fails if
// the OS entropy source itself is broken, not a condition a test fake
// needs to handle - so a read failure falls back to a fixed suffix rather
// than adding an error return every caller would have to check.
func randomSuffix() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "fallback"
	}

	return hex.EncodeToString(b)
}

func (s *Store) validateToken(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	expiry, ok := s.tokens[token]

	return ok && time.Now().Before(expiry)
}

// New starts an httptest.Server backed by a fresh Store and issues a
// write token valid against it - none of this repo's tests need a
// distinct one, matching hush-hush's own newTestServer convention.
func New(t *testing.T) (srv *httptest.Server, s *Store, token string) {
	t.Helper()

	s = newStore()

	_, token, err := s.CreateWriteToken(t.Context(), "test", time.Hour)
	if err != nil {
		t.Fatalf("create write token: %v", err)
	}

	srv = httptest.NewServer(newMux(s))
	t.Cleanup(srv.Close)

	return srv, s, token
}

type objectMetadata struct {
	ID          string   `json:"id"`
	UsedBy      []string `json:"used_by,omitempty"`
	Description string   `json:"description,omitempty"`
}

type createObjectRequest struct {
	ID          string   `json:"id"`
	Value       []byte   `json:"value"`
	UsedBy      []string `json:"used_by,omitempty"`
	Description string   `json:"description,omitempty"`
}

type updateObjectRequest struct {
	Value []byte `json:"value"`
}

type errorBody struct {
	Error string `json:"error"`
}

// newMux wires the same four /objects endpoints internal/client actually
// calls (api/openapi.yaml) - not GET /objects/{id}/used-by, GET
// /audit-log, or GET /healthz, none of which internal/client's Client
// exposes a method for.
func newMux(s *Store) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /objects", requireWriteToken(s, handleCreateObject(s)))
	mux.HandleFunc("GET /objects/{id}", handleGetObject(s))
	mux.HandleFunc("PUT /objects/{id}", requireWriteToken(s, handleUpdateObject(s)))
	mux.HandleFunc("DELETE /objects/{id}", requireWriteToken(s, handleDeleteObject(s)))

	return mux
}

// requireWriteToken matches hush-hush's own handler: an unknown,
// malformed, or expired token are all the same 401 to the caller.
func requireWriteToken(s *Store, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || got == "" || !s.validateToken(got) {
			writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")

			return
		}

		next(w, r)
	}
}

func handleCreateObject(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createObjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "malformed request body")

			return
		}

		if req.ID == "" || len(req.Value) == 0 {
			writeError(w, http.StatusBadRequest, "id and value are required")

			return
		}

		if err := s.CreateObject(r.Context(), req.ID, req.Value, req.UsedBy, req.Description); err != nil {
			writeError(w, http.StatusConflict, "object already exists")

			return
		}

		writeJSON(w, http.StatusCreated, objectMetadata{ID: req.ID, UsedBy: req.UsedBy, Description: req.Description})
	}
}

// handleGetObject is unauthenticated by design, matching hush-hush's own
// v1 confidentiality boundary: "who holds a matching private key".
func handleGetObject(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		obj, err := s.GetObject(r.Context(), r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusNotFound, "unknown object")

			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(obj.Value)
	}
}

func handleUpdateObject(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req updateObjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "malformed request body")

			return
		}

		if len(req.Value) == 0 {
			writeError(w, http.StatusBadRequest, "value is required")

			return
		}

		id := r.PathValue("id")

		if err := s.UpdateObject(r.Context(), id, req.Value); err != nil {
			writeError(w, http.StatusNotFound, "unknown object")

			return
		}

		obj, err := s.GetObject(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "unknown object")

			return
		}

		writeJSON(w, http.StatusOK, objectMetadata{ID: id, UsedBy: obj.UsedBy, Description: obj.Description})
	}
}

func handleDeleteObject(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.DeleteObject(r.Context(), r.PathValue("id")); err != nil {
			writeError(w, http.StatusNotFound, "unknown object")

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorBody{Error: msg})
}
