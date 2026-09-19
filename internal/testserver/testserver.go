// Package testserver is a small in-memory, stateful stand-in for the
// hush-hush server's HTTP API - internal/client's actual transport
// contract (api/openapi.yaml's /objects endpoints), not hush-hush's own
// internal/api implementation, which stays in that repo. It exists so
// internal/cli and cmd/hush-hush-cli's tests can exercise a real
// create/get/update/delete/list round trip - including auth, 404, and 409
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
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"strconv"
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

// ErrInvalidFilter is returned by Store.QueryAuditLog for a filter value
// the real server would reject with a 400 - matching object_id's own
// pattern (api/openapi.yaml's ObjectId schema).
var ErrInvalidFilter = errors.New("invalid filter")

var objectIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// Object is a stored object's current state.
type Object struct {
	Value       []byte
	UsedBy      []string
	Description string
}

// Store is an in-memory object store plus write-token issuance, backing a
// Server started by New. Safe for concurrent use.
type Store struct {
	mu       sync.Mutex
	objects  map[string]Object
	tokens   map[string]time.Time
	auditLog []AuditLogEntry
	auditSeq int64
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

// ListObjects returns every stored object's id, used_by, and description -
// never the value - sorted by id, optionally narrowed to objects whose
// used_by includes usedByFilter ("" means no filter). Matches hush-hush's
// own GET /objects (hush-hush#188/#189).
func (s *Store) ListObjects(_ context.Context, usedByFilter string) ([]ObjectMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]ObjectMetadata, 0, len(s.objects))
	for id, obj := range s.objects {
		if usedByFilter != "" && !slices.Contains(obj.UsedBy, usedByFilter) {
			continue
		}

		result = append(result, ObjectMetadata{ID: id, UsedBy: obj.UsedBy, Description: obj.Description})
	}

	slices.SortFunc(result, func(a, b ObjectMetadata) int { return strings.Compare(a.ID, b.ID) })

	return result, nil
}

// AuditLogEntry mirrors hush-hush's GET /audit-log response shape
// (api/openapi.yaml's AuditLogEntry schema), oldest-first.
type AuditLogEntry struct {
	ID        int64     `json:"id"`
	ObjectID  string    `json:"object_id"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
	Caller    *string   `json:"caller,omitempty"`
	IP        string    `json:"ip"`
	ActorType *string   `json:"actor_type,omitempty"`
	ActorID   *string   `json:"actor_id,omitempty"`
}

// AuditLogFilter narrows Store.QueryAuditLog - mirrors GET /audit-log's own
// query parameters (api/openapi.yaml). Filters combine with AND.
type AuditLogFilter struct {
	ObjectID *string
	Caller   *string
	Actor    *string
	From     *time.Time
	To       *time.Time
	After    *int64
	Limit    *int32
}

// RecordAuditEntry appends entry to the audit log, assigning it the next
// strictly increasing id and, if Timestamp is zero, the current time.
// Tests seed entries this way rather than through create/get/update/delete
// themselves, so a filter/pagination test controls exactly what exists
// without needing a matching object round trip for every row.
func (s *Store) RecordAuditEntry(entry AuditLogEntry) AuditLogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.auditSeq++
	entry.ID = s.auditSeq

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	s.auditLog = append(s.auditLog, entry)

	return entry
}

// QueryAuditLog returns entries matching filter, oldest first, one page at
// a time - default page size 50, capped at 500, matching
// api/openapi.yaml's own defaults. Returns ErrInvalidFilter for an
// ObjectID that doesn't match the real server's id pattern.
func (s *Store) QueryAuditLog(_ context.Context, filter AuditLogFilter) ([]AuditLogEntry, error) {
	if filter.ObjectID != nil && !objectIDPattern.MatchString(*filter.ObjectID) {
		return nil, fmt.Errorf("%w: object_id %q", ErrInvalidFilter, *filter.ObjectID)
	}

	limit := int32(50)
	if filter.Limit != nil {
		limit = *filter.Limit
	}

	if limit < 1 || limit > 500 {
		return nil, fmt.Errorf("%w: limit %d out of range [1, 500]", ErrInvalidFilter, limit)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]AuditLogEntry, 0, limit)
	for _, e := range s.auditLog {
		if filter.ObjectID != nil && e.ObjectID != *filter.ObjectID {
			continue
		}

		if filter.Caller != nil && (e.Caller == nil || *e.Caller != *filter.Caller) {
			continue
		}

		if filter.Actor != nil && (e.ActorID == nil || *e.ActorID != *filter.Actor) {
			continue
		}

		if filter.From != nil && e.Timestamp.Before(*filter.From) {
			continue
		}

		if filter.To != nil && e.Timestamp.After(*filter.To) {
			continue
		}

		if filter.After != nil && e.ID <= *filter.After {
			continue
		}

		result = append(result, e)
		if int32(len(result)) >= limit {
			break
		}
	}

	return result, nil
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

// ObjectMetadata is a stored object's id, used_by, and description, with no
// value - the shape create/update/list all return over the wire.
type ObjectMetadata struct {
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

// newMux wires the six /objects and /audit-log endpoints internal/client
// actually calls (api/openapi.yaml) - not GET /objects/{id}/used-by or GET
// /healthz, neither of which internal/client's Client exposes a method
// for.
func newMux(s *Store) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /objects", requireWriteToken(s, handleCreateObject(s)))
	mux.HandleFunc("GET /objects", requireWriteToken(s, handleListObjects(s)))
	mux.HandleFunc("GET /objects/{id}", handleGetObject(s))
	mux.HandleFunc("PUT /objects/{id}", requireWriteToken(s, handleUpdateObject(s)))
	mux.HandleFunc("DELETE /objects/{id}", requireWriteToken(s, handleDeleteObject(s)))
	mux.HandleFunc("GET /audit-log", handleQueryAuditLog(s))

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

		writeJSON(w, http.StatusCreated, ObjectMetadata{ID: req.ID, UsedBy: req.UsedBy, Description: req.Description})
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

		writeJSON(w, http.StatusOK, ObjectMetadata{ID: id, UsedBy: obj.UsedBy, Description: obj.Description})
	}
}

// handleListObjects is gated by the same write token as create/update/
// delete, unlike handleGetObject - matching hush-hush's own design:
// enumerating every object is a capability none of the other, id-scoped
// reads grant on their own.
func handleListObjects(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		objects, err := s.ListObjects(r.Context(), r.URL.Query().Get("used_by"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")

			return
		}

		writeJSON(w, http.StatusOK, objects)
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

// handleQueryAuditLog is unauthenticated, matching hush-hush-go's own
// QueryAuditLog doc comment ("No credential is required").
func handleQueryAuditLog(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter, err := parseAuditLogFilter(r.URL.Query())
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())

			return
		}

		entries, err := s.QueryAuditLog(r.Context(), filter)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())

			return
		}

		writeJSON(w, http.StatusOK, entries)
	}
}

func parseAuditLogFilter(q url.Values) (AuditLogFilter, error) {
	var filter AuditLogFilter

	if v := q.Get("object_id"); v != "" {
		filter.ObjectID = &v
	}

	if v := q.Get("caller"); v != "" {
		filter.Caller = &v
	}

	if v := q.Get("actor"); v != "" {
		filter.Actor = &v
	}

	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return filter, fmt.Errorf("from: %w", err)
		}

		filter.From = &t
	}

	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return filter, fmt.Errorf("to: %w", err)
		}

		filter.To = &t
	}

	if v := q.Get("after"); v != "" {
		after, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return filter, fmt.Errorf("after: %w", err)
		}

		filter.After = &after
	}

	if v := q.Get("limit"); v != "" {
		limit, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return filter, fmt.Errorf("limit: %w", err)
		}

		limit32 := int32(limit)
		filter.Limit = &limit32
	}

	return filter, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorBody{Error: msg})
}
