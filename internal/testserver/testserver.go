// Package testserver is a small in-memory, stateful stand-in for the
// hush-hush server's HTTP API - internal/client's actual transport
// contract (api/openapi.yaml's /objects endpoints), not hush-hush's own
// internal/api implementation, which stays in that repo. It exists so
// internal/cli and internal/cmd's tests can exercise a real
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

// ErrConsumerAlreadyExists is returned by Store.AddConsumer for a name
// already in the consumer directory.
var ErrConsumerAlreadyExists = errors.New("consumer already exists")

// ErrConsumerNotFound is returned by Store.UpdateConsumer, when renaming an
// unknown name, and by Store.DeleteConsumer for an unknown consumer.
var ErrConsumerNotFound = errors.New("consumer not found")

// ErrConsumerTokenNotFound is returned by Store.RotateConsumerToken and
// Store.PurgeConsumerToken for an unknown, already revoked, or already
// expired token id.
var ErrConsumerTokenNotFound = errors.New("consumer token not found")

// ErrConsumerTokenActive is returned by Store.PurgeConsumerToken for a
// token that's neither revoked nor expired yet.
var ErrConsumerTokenActive = errors.New("consumer token is still active")

var objectIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// Object is a stored object's current state.
type Object struct {
	Value       []byte
	UsedBy      []string
	Description string
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Store is an in-memory object store plus write-token issuance, backing a
// Server started by New. Safe for concurrent use.
type Store struct {
	mu           sync.Mutex
	objects      map[string]Object
	tokens       map[string]time.Time
	auditLog     []AuditLogEntry
	auditSeq     int64
	bootstrapped bool
	// consumers holds the consumer directory: name -> registered age public
	// key, nil if none registered. Presence as a key is what makes a name
	// a directory entry, whether it got there via AddConsumer or via
	// appearing in some object's used_by list at creation.
	consumers map[string]*string
	// consumerTokens holds every issued consumer read token's full state,
	// keyed by id (alrayyes/hush-hush#446's consumerBearerAuth - a
	// consumer token authorizes GetObject only, only for an object whose
	// used_by includes its bound consumer; alrayyes/hush-hush#467 added
	// minting/listing/rotating/revoking/purging one via a write bearer
	// token instead of a cookie session).
	consumerTokens map[string]*consumerToken
	// consumerTokenValues maps a raw token value to the id that issued
	// it - the lookup handleGetObject's own auth check uses, and the one
	// RotateConsumerToken has to keep in sync when a token's value
	// changes.
	consumerTokenValues map[string]string
	// ownerPublicKey is the owner's escrowed identity public key, served
	// by GET /auth/identity. Empty means the owner hasn't registered one.
	ownerPublicKey string
}

// consumerToken is one issued consumer read token's full state - the
// metadata every response shape (ConsumerToken, ConsumerTokenWithValue)
// is built from.
type consumerToken struct {
	ID          string
	Consumer    string
	Description string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	LastUsedAt  *time.Time
	Revoked     bool
	Value       string
}

// ConsumerToken is one issued consumer read token's metadata - the shape
// GET /consumer-tokens returns, and CreateConsumerToken/RotateConsumerToken
// embed alongside the raw value - never a raw value on its own
// (api/openapi.yaml's ConsumerTokenMetadata).
type ConsumerToken struct {
	ID          string     `json:"id"`
	Consumer    string     `json:"consumer"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	Revoked     bool       `json:"revoked"`
	// Status and AllowedActions mirror hush-hush's own server-computed
	// fields (hush-hush-go v4.2.4+).
	Status         string   `json:"status"`
	AllowedActions []string `json:"allowed_actions"`
}

// ConsumerTokenWithValue is POST /consumer-tokens and
// POST /consumer-tokens/{id}/rotate's own response shape - the same
// metadata as ConsumerToken plus the raw token, shown here once
// (api/openapi.yaml's ConsumerTokenWithValue).
type ConsumerTokenWithValue struct {
	ConsumerToken
	Value string `json:"value"`
}

func toConsumerTokenMetadata(t *consumerToken) ConsumerToken {
	status, actions := "active", []string{"rotate", "revoke"}

	switch {
	case t.Revoked:
		status, actions = "revoked", []string{"purge"}
	case !time.Now().Before(t.ExpiresAt):
		status, actions = "expired", []string{"purge"}
	}

	return ConsumerToken{
		Status:         status,
		AllowedActions: actions,
		ID:             t.ID,
		Consumer:       t.Consumer,
		Description:    t.Description,
		CreatedAt:      t.CreatedAt,
		ExpiresAt:      t.ExpiresAt,
		LastUsedAt:     t.LastUsedAt,
		Revoked:        t.Revoked,
	}
}

func toConsumerTokenWithValue(t *consumerToken) ConsumerTokenWithValue {
	return ConsumerTokenWithValue{ConsumerToken: toConsumerTokenMetadata(t), Value: t.Value}
}

// newStore defaults bootstrapped to true - the shape every other test in
// this package already assumes (a normal, already-set-up server); a test
// exercising the unbootstrapped case calls SetBootstrapped(false) itself.
func newStore() *Store {
	return &Store{
		objects:             make(map[string]Object),
		tokens:              make(map[string]time.Time),
		bootstrapped:        true,
		consumers:           make(map[string]*string),
		consumerTokens:      make(map[string]*consumerToken),
		consumerTokenValues: make(map[string]string),
	}
}

// SetBootstrapped overrides the value GET /auth/status reports.
func (s *Store) SetBootstrapped(bootstrapped bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.bootstrapped = bootstrapped
}

// AuthStatus reports whether an admin account has been created yet -
// matches hush-hush's own GET /auth/status (hush-hush-go#100).
func (s *Store) AuthStatus(_ context.Context) AuthStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	return AuthStatus{Bootstrapped: s.bootstrapped}
}

// CreateObject stores value under id, or ErrAlreadyExists if id is taken.
// Every name in usedBy that's new to the consumer directory joins it with
// no registered key, matching ConsumerEntry's own doc comment: a directory
// entry is "a distinct consumer name recorded in some object's used_by
// list", not only one added explicitly via AddConsumer.
func (s *Store) CreateObject(ctx context.Context, id string, value []byte, usedBy []string, description string) error {
	return s.createObject(ctx, id, value, usedBy, description, nil)
}

func (s *Store) createObject(_ context.Context, id string, value []byte, usedBy []string, description string, tags []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.objects[id]; ok {
		return ErrAlreadyExists
	}

	now := time.Now()
	s.objects[id] = Object{Value: value, UsedBy: usedBy, Description: description, Tags: tags, CreatedAt: now, UpdatedAt: now}

	for _, name := range usedBy {
		if _, ok := s.consumers[name]; !ok {
			s.consumers[name] = nil
		}
	}

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

// UpdateObject replaces id's stored value, leaving used_by, description
// and tags unchanged, or ErrNotFound.
func (s *Store) UpdateObject(ctx context.Context, id string, value []byte) error {
	return s.updateObject(ctx, id, value, nil, nil)
}

// updateObject also replaces id's tags when tags is non-nil (an empty
// slice clears them), and likewise its used_by when usedBy is non-nil,
// matching PUT /objects/{slug}'s own semantics.
func (s *Store) updateObject(_ context.Context, id string, value []byte, tags, usedBy *[]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, ok := s.objects[id]
	if !ok {
		return ErrNotFound
	}

	obj.Value = value
	obj.UpdatedAt = time.Now()

	if tags != nil {
		obj.Tags = *tags
	}

	if usedBy != nil {
		obj.UsedBy = *usedBy

		for _, name := range *usedBy {
			if _, ok := s.consumers[name]; !ok {
				s.consumers[name] = nil
			}
		}
	}

	s.objects[id] = obj

	return nil
}

// SetObjectTags replaces id's tags as given - a test-setup shortcut that
// skips the HTTP layer's normalization, so pass already-valid tags.
func (s *Store) SetObjectTags(_ context.Context, id string, tags []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, ok := s.objects[id]
	if !ok {
		return ErrNotFound
	}

	obj.Tags = tags
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

// ConsumerEntry is one directory entry, mirroring hush-hush-go's own type
// of the same name (GET /consumers's paginated response shape, and
// AddConsumer/UpdateConsumer's own response).
type ConsumerEntry struct {
	Name        string  `json:"name"`
	PublicKey   *string `json:"public_key,omitempty"`
	SecretCount int32   `json:"secret_count"`
}

// ConsumersPage is GET /consumers's paginated response shape, returned
// whenever the request's filter has any field set.
type ConsumersPage struct {
	Consumers []ConsumerEntry `json:"consumers"`
	Total     int32           `json:"total"`
}

// ConsumerFilter narrows Store.ListConsumers, mirroring GET /consumers's
// own query parameters.
type ConsumerFilter struct {
	Q        *string
	Page     *int32
	PageSize *int32
}

// ConsumersResult is Store.ListConsumers's return value - exactly one
// field set, matching GET /consumers's own union response shape: Names for
// an empty filter, Page for a filter with any field set.
type ConsumersResult struct {
	Names []string
	Page  *ConsumersPage
}

// AddConsumer adds name to the directory with no public key registered, or
// ErrConsumerAlreadyExists if it's already there.
func (s *Store) AddConsumer(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.consumers[name]; ok {
		return ErrConsumerAlreadyExists
	}

	s.consumers[name] = nil

	return nil
}

// UpdateConsumer renames name to newName (if non-nil) and/or registers
// publicKey on the resulting name (if non-nil), applied in that order -
// matching hush-hush-go's own UpdateConsumer doc comment. Renaming an
// unknown name is ErrConsumerNotFound; registering a key alone on an
// unknown name upserts a directory entry for it instead. A rename whose
// target already has its own recorded objects merges under it, and a
// target's own key wins the merge if it already had one.
func (s *Store) UpdateConsumer(_ context.Context, name string, newName, publicKey *string) (ConsumerEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, ok := s.consumers[name]
	if !ok {
		if newName != nil {
			return ConsumerEntry{}, ErrConsumerNotFound
		}

		s.consumers[name] = publicKey

		return s.consumerEntryLocked(name), nil
	}

	target := name

	if newName != nil {
		target = *newName

		if targetKey, targetOK := s.consumers[target]; targetOK && targetKey != nil {
			key = targetKey
		}

		delete(s.consumers, name)
		s.consumers[target] = key

		for id, obj := range s.objects {
			obj.UsedBy = renameUsedBy(obj.UsedBy, name, target)
			s.objects[id] = obj
		}
	}

	if publicKey != nil {
		s.consumers[target] = publicKey
	}

	return s.consumerEntryLocked(target), nil
}

// DeleteConsumer strips name from every object's used_by list and removes
// it from the directory, or ErrConsumerNotFound if it's not in the
// directory at all.
func (s *Store) DeleteConsumer(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.consumers[name]; !ok {
		return ErrConsumerNotFound
	}

	delete(s.consumers, name)

	for id, obj := range s.objects {
		obj.UsedBy = removeUsedBy(obj.UsedBy, name)
		s.objects[id] = obj
	}

	return nil
}

// ListConsumers returns the consumer directory, matching GET /consumers's
// own union response shape: an empty filter returns the plain, sorted name
// list; any filter field set switches to the paginated ConsumersPage shape
// (name substring match, 1-based page, page size capped at 100).
func (s *Store) ListConsumers(_ context.Context, filter ConsumerFilter) ConsumersResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	names := make([]string, 0, len(s.consumers))
	for name := range s.consumers {
		names = append(names, name)
	}

	slices.Sort(names)

	if filter.Q == nil && filter.Page == nil && filter.PageSize == nil {
		return ConsumersResult{Names: names}
	}

	matched := names
	if filter.Q != nil {
		q := strings.ToLower(*filter.Q)
		filtered := make([]string, 0, len(names))

		for _, name := range names {
			if strings.Contains(strings.ToLower(name), q) {
				filtered = append(filtered, name)
			}
		}

		matched = filtered
	}

	total := int32(len(matched)) //nolint:gosec // matched is an in-memory test fixture, never near int32 max

	page := int32(1)
	if filter.Page != nil {
		page = *filter.Page
	}

	pageSize := int32(20)
	if filter.PageSize != nil {
		pageSize = *filter.PageSize
	}

	if pageSize > 100 {
		pageSize = 100
	}

	start := min(int((page-1)*pageSize), len(matched))
	end := min(start+int(pageSize), len(matched))

	entries := make([]ConsumerEntry, 0, end-start)
	for _, name := range matched[start:end] {
		entries = append(entries, s.consumerEntryLocked(name))
	}

	return ConsumersResult{Page: &ConsumersPage{Consumers: entries, Total: total}}
}

// consumerEntryLocked builds name's directory entry, including its
// computed secret count - callers must hold s.mu.
func (s *Store) consumerEntryLocked(name string) ConsumerEntry {
	entry := ConsumerEntry{Name: name}

	for _, obj := range s.objects {
		if slices.Contains(obj.UsedBy, name) {
			entry.SecretCount++
		}
	}

	if key := s.consumers[name]; key != nil {
		entry.PublicKey = key
	}

	return entry
}

// renameUsedBy replaces every occurrence of from in usedBy with to,
// deduplicating - a used_by list already naming both collapses to one.
func renameUsedBy(usedBy []string, from, to string) []string {
	seen := make(map[string]bool, len(usedBy))
	result := make([]string, 0, len(usedBy))

	for _, name := range usedBy {
		if name == from {
			name = to
		}

		if seen[name] {
			continue
		}

		seen[name] = true

		result = append(result, name)
	}

	return result
}

// removeUsedBy returns usedBy with every occurrence of name stripped.
func removeUsedBy(usedBy []string, name string) []string {
	result := make([]string, 0, len(usedBy))

	for _, n := range usedBy {
		if n != name {
			result = append(result, n)
		}
	}

	return result
}

// ListObjects returns every stored object's slug, used_by, and
// description - never the value - sorted by slug, optionally narrowed to
// objects whose used_by includes usedByFilter ("" means no filter).
// Matches hush-hush's own GET /objects (hush-hush#188/#189).
func (s *Store) ListObjects(ctx context.Context, usedByFilter string) ([]ObjectMetadata, error) {
	return s.ListObjectsFiltered(ctx, usedByFilter, nil)
}

// ListObjectsFiltered is ListObjects narrowed further to objects carrying
// every one of tags, compared case-insensitively, matching GET /objects'
// repeated tag parameter.
func (s *Store) ListObjectsFiltered(_ context.Context, usedByFilter string, tags []string) ([]ObjectMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]ObjectMetadata, 0, len(s.objects))
	for id, obj := range s.objects {
		if usedByFilter != "" && !slices.Contains(obj.UsedBy, usedByFilter) {
			continue
		}

		if !hasAllTags(obj.Tags, tags) {
			continue
		}

		actor := &Actor{ID: "testserver", Type: "token"}
		created, updated := obj.CreatedAt, obj.UpdatedAt

		result = append(result, ObjectMetadata{
			Slug: id, UsedBy: obj.UsedBy, Description: obj.Description, Tags: tagsOrEmpty(obj.Tags),
			CreatedAt: &created, CreatedBy: actor, UpdatedAt: &updated, UpdatedBy: actor,
		})
	}

	slices.SortFunc(result, func(a, b ObjectMetadata) int { return strings.Compare(a.Slug, b.Slug) })

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
		if !auditLogEntryMatches(e, filter) {
			continue
		}

		result = append(result, e)
		if len(result) >= int(limit) {
			break
		}
	}

	return result, nil
}

// auditLogEntryMatches reports whether e satisfies every filter field
// that's set - filters combine with AND (api/openapi.yaml).
func auditLogEntryMatches(e AuditLogEntry, filter AuditLogFilter) bool {
	if filter.ObjectID != nil && e.ObjectID != *filter.ObjectID {
		return false
	}

	if filter.Caller != nil && (e.Caller == nil || *e.Caller != *filter.Caller) {
		return false
	}

	if filter.Actor != nil && (e.ActorID == nil || *e.ActorID != *filter.Actor) {
		return false
	}

	if filter.From != nil && e.Timestamp.Before(*filter.From) {
		return false
	}

	if filter.To != nil && e.Timestamp.After(*filter.To) {
		return false
	}

	if filter.After != nil && e.ID <= *filter.After {
		return false
	}

	return true
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

// CreateConsumerToken issues a fresh read token bound to consumer, valid
// for a year - a thin convenience over IssueConsumerToken for tests that
// only need a plain token value and don't care about description or ttl.
func (s *Store) CreateConsumerToken(consumer string) (token string) {
	return s.IssueConsumerToken(consumer, "", 365*24*time.Hour).Value
}

// IssueConsumerToken mints a new consumer read token, mirroring
// hush-hush's POST /consumer-tokens (alrayyes/hush-hush#467).
func (s *Store) IssueConsumerToken(consumer, description string, ttl time.Duration) ConsumerTokenWithValue {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	rec := &consumerToken{
		ID:          randomSuffix(),
		Consumer:    consumer,
		Description: description,
		CreatedAt:   now,
		ExpiresAt:   now.Add(ttl),
		Value:       "consumer-" + consumer + "-" + randomSuffix(),
	}

	s.consumerTokens[rec.ID] = rec
	s.consumerTokenValues[rec.Value] = rec.ID

	return toConsumerTokenWithValue(rec)
}

// ListConsumerTokens returns every issued consumer token's metadata,
// sorted by id, mirroring hush-hush's GET /consumer-tokens.
func (s *Store) ListConsumerTokens() []ConsumerToken {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]ConsumerToken, 0, len(s.consumerTokens))
	for _, rec := range s.consumerTokens {
		result = append(result, toConsumerTokenMetadata(rec))
	}

	slices.SortFunc(result, func(a, b ConsumerToken) int { return strings.Compare(a.ID, b.ID) })

	return result
}

// RotateConsumerToken replaces id's secret and expiry, keeping its
// consumer and description unchanged - ErrConsumerTokenNotFound for an
// unknown, already revoked, or already expired id, matching hush-hush-go's
// own RotateConsumerToken doc comment.
func (s *Store) RotateConsumerToken(id string, ttl time.Duration) (ConsumerTokenWithValue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.consumerTokens[id]
	if !ok || rec.Revoked || time.Now().After(rec.ExpiresAt) {
		return ConsumerTokenWithValue{}, ErrConsumerTokenNotFound
	}

	delete(s.consumerTokenValues, rec.Value)

	rec.Value = "consumer-" + rec.Consumer + "-" + randomSuffix()
	rec.ExpiresAt = time.Now().Add(ttl)
	s.consumerTokenValues[rec.Value] = id

	return toConsumerTokenWithValue(rec), nil
}

// RevokeConsumerToken invalidates id - revoking an unknown, already
// revoked, or already expired id isn't an error, matching hush-hush-go's
// own RevokeConsumerToken doc comment.
func (s *Store) RevokeConsumerToken(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rec, ok := s.consumerTokens[id]; ok {
		rec.Revoked = true
	}
}

// PurgeConsumerToken permanently removes id, once it's already revoked or
// past its expiry - ErrConsumerTokenNotFound for an unknown id,
// ErrConsumerTokenActive for one still active, matching hush-hush-go's own
// PurgeConsumerToken doc comment.
func (s *Store) PurgeConsumerToken(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.consumerTokens[id]
	if !ok {
		return ErrConsumerTokenNotFound
	}

	if !rec.Revoked && time.Now().Before(rec.ExpiresAt) {
		return ErrConsumerTokenActive
	}

	delete(s.consumerTokens, id)
	delete(s.consumerTokenValues, rec.Value)

	return nil
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

// consumerForToken reports the consumer name value is bound to, and
// whether it's currently a valid consumer token at all - false for an
// unknown, revoked, or expired value, the same as hush-hush's own
// consumerBearerAuth would reject each of those.
func (s *Store) consumerForToken(value string) (consumer string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, ok := s.consumerTokenValues[value]
	if !ok {
		return "", false
	}

	rec := s.consumerTokens[id]
	if rec.Revoked || time.Now().After(rec.ExpiresAt) {
		return "", false
	}

	return rec.Consumer, true
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

// AuthStatus mirrors hush-hush's GET /auth/status response shape
// (api/openapi.yaml's AuthStatus schema).
type AuthStatus struct {
	Bootstrapped bool `json:"bootstrapped"`
}

// ObjectMetadata is a stored object's slug, used_by, and description, with
// no value - the shape create/update/list all return over the wire. Slug,
// not id: the server's own internal id is a separate, opaque value never
// exposed over this API (alrayyes/Hush-Hush#404).
type ObjectMetadata struct {
	Slug        string   `json:"slug"`
	UsedBy      []string `json:"used_by,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags"`
	// CreatedAt through UpdatedBy are only sent by GET /objects, as the
	// real server does (hush-hush-go v4.2.4+). The fake has no audit log,
	// so both actors are always the same stand-in.
	CreatedAt *time.Time `json:"created_at,omitempty"`
	CreatedBy *Actor     `json:"created_by,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	UpdatedBy *Actor     `json:"updated_by,omitempty"`
}

// Actor is who performed an audited write.
type Actor struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type createObjectRequest struct {
	Slug        string   `json:"slug"`
	Value       []byte   `json:"value"`
	UsedBy      []string `json:"used_by,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type updateObjectRequest struct {
	Value  []byte    `json:"value"`
	Tags   *[]string `json:"tags,omitempty"`
	UsedBy *[]string `json:"used_by,omitempty"`
}

// tagPattern and maxTags mirror api/openapi.yaml's Tags schema: 1 to 32
// characters from a-z 0-9 . _ / -, at most 10 per object.
var tagPattern = regexp.MustCompile(`^[a-z0-9._/-]{1,32}$`)

const maxTags = 10

var errInvalidTags = errors.New("invalid tags")

// normalizeTags lower-cases and de-duplicates tags, then validates them -
// the same conversions and limits the real server applies.
func normalizeTags(in []string) ([]string, error) {
	out := make([]string, 0, len(in))

	for _, tag := range in {
		tag = strings.ToLower(tag)
		if !tagPattern.MatchString(tag) {
			return nil, fmt.Errorf("%w: tag %q must be 1-32 characters from a-z 0-9 . _ / -", errInvalidTags, tag)
		}

		if !slices.Contains(out, tag) {
			out = append(out, tag)
		}
	}

	if len(out) > maxTags {
		return nil, fmt.Errorf("%w: at most %d tags per object", errInvalidTags, maxTags)
	}

	return out, nil
}

// tagsOrEmpty makes sure a response always carries the tags array, never
// null, as the real API does.
func tagsOrEmpty(tags []string) []string {
	if tags == nil {
		return []string{}
	}

	return tags
}

type errorBody struct {
	Error string `json:"error"`
}

// newMux wires the /objects, /audit-log, and /auth/status endpoints
// internal/client actually calls (api/openapi.yaml) - not GET /healthz,
// which internal/client's Client exposes no method for.
func newMux(s *Store) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /objects", requireWriteToken(s, handleCreateObject(s)))
	mux.HandleFunc("GET /objects", requireWriteToken(s, handleListObjects(s)))
	mux.HandleFunc("GET /objects/{id}", handleGetObject(s))
	mux.HandleFunc("PUT /objects/{id}", requireWriteToken(s, handleUpdateObject(s)))
	mux.HandleFunc("DELETE /objects/{id}", requireWriteToken(s, handleDeleteObject(s)))
	mux.HandleFunc("GET /objects/{id}/used-by", handleGetObjectUsedBy(s))
	mux.HandleFunc("GET /audit-log", handleQueryAuditLog(s))
	mux.HandleFunc("GET /auth/status", handleAuthStatus(s))
	mux.HandleFunc("GET /auth/identity", requireWriteToken(s, handleAuthIdentity(s)))
	mux.HandleFunc("GET /consumers", requireWriteToken(s, handleListConsumers(s)))
	mux.HandleFunc("POST /consumers", requireWriteToken(s, handleAddConsumer(s)))
	// {name...}, not {name}: a consumer name is a repo or host slug and
	// routinely contains a slash (e.g. "homelab/vps-docker"), which a
	// single path segment can't capture.
	mux.HandleFunc("PATCH /consumers/{name...}", requireWriteToken(s, handleUpdateConsumer(s)))
	mux.HandleFunc("DELETE /consumers/{name...}", requireWriteToken(s, handleDeleteConsumer(s)))
	mux.HandleFunc("POST /consumer-tokens", requireWriteToken(s, handleCreateConsumerToken(s)))
	mux.HandleFunc("GET /consumer-tokens", requireWriteToken(s, handleListConsumerTokens(s)))
	mux.HandleFunc("POST /consumer-tokens/{id}/rotate", requireWriteToken(s, handleRotateConsumerToken(s)))
	mux.HandleFunc("DELETE /consumer-tokens/{id}/purge", requireWriteToken(s, handlePurgeConsumerToken(s)))
	mux.HandleFunc("DELETE /consumer-tokens/{id}", requireWriteToken(s, handleRevokeConsumerToken(s)))

	return mux
}

// SetOwnerPublicKey sets the escrowed identity public key GET
// /auth/identity returns. Unset, the response carries no public_key, as
// for an owner who hasn't completed a first registration.
func (s *Store) SetOwnerPublicKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ownerPublicKey = key
}

// ownerIdentityResponse matches api/openapi.yaml's OwnerIdentity schema.
type ownerIdentityResponse struct {
	PublicKey *string `json:"public_key,omitempty"`
}

func handleAuthIdentity(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		s.mu.Lock()
		key := s.ownerPublicKey
		s.mu.Unlock()

		resp := ownerIdentityResponse{}
		if key != "" {
			resp.PublicKey = &key
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

// handleAuthStatus is unauthenticated, matching hush-hush-go's own
// AuthStatus doc comment ("no credential is required to call it").
func handleAuthStatus(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.AuthStatus(r.Context()))
	}
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

		if req.Slug == "" || len(req.Value) == 0 {
			writeError(w, http.StatusBadRequest, "slug and value are required")

			return
		}

		tags, err := normalizeTags(req.Tags)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())

			return
		}

		if err := s.createObject(r.Context(), req.Slug, req.Value, req.UsedBy, req.Description, tags); err != nil {
			writeError(w, http.StatusConflict, "object already exists")

			return
		}

		writeJSON(w, http.StatusCreated, ObjectMetadata{Slug: req.Slug, UsedBy: req.UsedBy, Description: req.Description, Tags: tagsOrEmpty(tags)})
	}
}

// handleGetObject requires a write token, or a consumer token whose bound
// consumer appears in the object's used_by list - matching hush-hush's
// current contract (alrayyes/hush-hush#446), not the earlier
// unauthenticated-by-design one. A consumer token presented for an object
// outside its scope gets the same 404 an unknown slug would, never a 403,
// so it can't be used to enumerate which other slugs exist - the same
// anti-enumeration rule api/openapi.yaml documents on the real endpoint.
func handleGetObject(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")

		var consumer string

		switch {
		case ok && got != "" && s.validateToken(got):
			// A write token reads any object - no scope check.
		case ok && got != "":
			consumer, ok = s.consumerForToken(got)
			if !ok {
				writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")

				return
			}
		default:
			writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")

			return
		}

		obj, err := s.GetObject(r.Context(), r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusNotFound, "unknown object")

			return
		}

		if consumer != "" && !slices.Contains(obj.UsedBy, consumer) {
			writeError(w, http.StatusNotFound, "unknown object")

			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(obj.Value)
	}
}

// usedByResponse matches api/openapi.yaml's UsedBy schema.
type usedByResponse struct {
	UsedBy []string `json:"used_by"`
}

// handleGetObjectUsedBy is unauthenticated, matching handleGetObject: an
// object's own recorded consumers need no more than its id to fetch.
func handleGetObjectUsedBy(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		obj, err := s.GetObject(r.Context(), r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusNotFound, "unknown object")

			return
		}

		writeJSON(w, http.StatusOK, usedByResponse{UsedBy: obj.UsedBy})
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

		var tags *[]string

		if req.Tags != nil {
			normalized, err := normalizeTags(*req.Tags)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())

				return
			}

			tags = &normalized
		}

		id := r.PathValue("id")

		if err := s.updateObject(r.Context(), id, req.Value, tags, req.UsedBy); err != nil {
			writeError(w, http.StatusNotFound, "unknown object")

			return
		}

		obj, err := s.GetObject(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "unknown object")

			return
		}

		writeJSON(w, http.StatusOK, ObjectMetadata{Slug: id, UsedBy: obj.UsedBy, Description: obj.Description, Tags: tagsOrEmpty(obj.Tags)})
	}
}

// handleListObjects is gated by the same write token as create/update/
// delete, unlike handleGetObject - matching hush-hush's own design:
// enumerating every object is a capability none of the other, id-scoped
// reads grant on their own.
func handleListObjects(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		objects, err := s.ListObjectsFiltered(r.Context(), r.URL.Query().Get("used_by"), r.URL.Query()["tag"])
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

// handleListConsumers matches GET /consumers's own union response shape:
// the plain name array for an empty filter, ConsumersPage otherwise.
func handleListConsumers(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter, err := parseConsumerFilter(r.URL.Query())
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())

			return
		}

		result := s.ListConsumers(r.Context(), filter)
		if result.Page != nil {
			writeJSON(w, http.StatusOK, result.Page)

			return
		}

		writeJSON(w, http.StatusOK, result.Names)
	}
}

func parseConsumerFilter(q url.Values) (ConsumerFilter, error) {
	var filter ConsumerFilter

	if v := q.Get("q"); v != "" {
		filter.Q = &v
	}

	if v := q.Get("page"); v != "" {
		page, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return filter, fmt.Errorf("page: %w", err)
		}

		page32 := int32(page)
		filter.Page = &page32
	}

	if v := q.Get("page_size"); v != "" {
		pageSize, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return filter, fmt.Errorf("page_size: %w", err)
		}

		pageSize32 := int32(pageSize)
		filter.PageSize = &pageSize32
	}

	return filter, nil
}

type addConsumerRequest struct {
	Name string `json:"name"`
}

func handleAddConsumer(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req addConsumerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			writeError(w, http.StatusBadRequest, "name is required")

			return
		}

		if err := s.AddConsumer(r.Context(), req.Name); err != nil {
			writeError(w, http.StatusConflict, "consumer already exists")

			return
		}

		s.mu.Lock()
		entry := s.consumerEntryLocked(req.Name)
		s.mu.Unlock()

		writeJSON(w, http.StatusCreated, entry)
	}
}

type updateConsumerRequest struct {
	Name      *string `json:"name,omitempty"`
	PublicKey *string `json:"public_key,omitempty"`
}

func handleUpdateConsumer(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req updateConsumerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "malformed request body")

			return
		}

		entry, err := s.UpdateConsumer(r.Context(), r.PathValue("name"), req.Name, req.PublicKey)
		if err != nil {
			writeError(w, http.StatusNotFound, "unknown consumer")

			return
		}

		writeJSON(w, http.StatusOK, entry)
	}
}

func handleDeleteConsumer(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.DeleteConsumer(r.Context(), r.PathValue("name")); err != nil {
			writeError(w, http.StatusNotFound, "unknown consumer")

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

type createConsumerTokenRequest struct {
	Consumer    string `json:"consumer"`
	Description string `json:"description"`
	TTLSeconds  int64  `json:"ttl_seconds"`
}

func handleCreateConsumerToken(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createConsumerTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Consumer == "" || req.TTLSeconds <= 0 {
			writeError(w, http.StatusBadRequest, "consumer and a positive ttl_seconds are required")

			return
		}

		token := s.IssueConsumerToken(req.Consumer, req.Description, time.Duration(req.TTLSeconds)*time.Second)

		writeJSON(w, http.StatusCreated, token)
	}
}

func handleListConsumerTokens(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.ListConsumerTokens())
	}
}

type rotateConsumerTokenRequest struct {
	TTLSeconds int64 `json:"ttl_seconds"`
}

func handleRotateConsumerToken(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req rotateConsumerTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TTLSeconds <= 0 {
			writeError(w, http.StatusBadRequest, "a positive ttl_seconds is required")

			return
		}

		token, err := s.RotateConsumerToken(r.PathValue("id"), time.Duration(req.TTLSeconds)*time.Second)
		if err != nil {
			writeError(w, http.StatusNotFound, "unknown, revoked, or expired consumer token")

			return
		}

		writeJSON(w, http.StatusOK, token)
	}
}

func handleRevokeConsumerToken(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.RevokeConsumerToken(r.PathValue("id"))

		w.WriteHeader(http.StatusNoContent)
	}
}

func handlePurgeConsumerToken(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch err := s.PurgeConsumerToken(r.PathValue("id")); {
		case errors.Is(err, ErrConsumerTokenNotFound):
			writeError(w, http.StatusNotFound, "unknown consumer token")
		case errors.Is(err, ErrConsumerTokenActive):
			writeError(w, http.StatusConflict, "consumer token is still active")
		case err != nil:
			writeError(w, http.StatusInternalServerError, err.Error())
		default:
			w.WriteHeader(http.StatusNoContent)
		}
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

// hasAllTags reports whether have carries every one of want, ignoring case.
func hasAllTags(have, want []string) bool {
	for _, tag := range want {
		if !slices.ContainsFunc(have, func(h string) bool { return strings.EqualFold(h, tag) }) {
			return false
		}
	}

	return true
}
