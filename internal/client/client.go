// Package client is the CLI's transport to the hush-hush server - a thin
// adapter over the generated hush-hush-go SDK, kept so internal/cli depends
// on this package's own sentinel errors and ObjectMetadata shape rather than
// the SDK's directly (design.md).
package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	hushhush "github.com/alrayyes/hush-hush-go/v4"
)

// auditLogPageMax is the server's own page-size cap (api/openapi.yaml's
// GET /audit-log limit parameter, max 500).
const auditLogPageMax = 500

// ErrNoOwnerKey is returned by OwnerPublicKey for an owner who hasn't
// completed a first registration, so has no escrowed identity key yet.
var ErrNoOwnerKey = errors.New("owner has no escrowed identity public key yet")

// Sentinel errors mapped from the server's documented status codes -
// callers match on these rather than inspecting a status code themselves.
var (
	ErrUnauthorized  = errors.New("missing or invalid bearer token")
	ErrNotFound      = errors.New("object not found")
	ErrAlreadyExists = errors.New("object already exists")
)

// ErrUnexpectedStatus anchors an unmapped status's error to something
// errors.Is can match, since the status code and server message vary per
// call and can't be a fixed sentinel on their own.
var ErrUnexpectedStatus = errors.New("unexpected status")

// ErrConsumerNoPublicKey is returned by ConsumerPublicKey for a name not in
// the consumer directory at all, or in it but with no key registered -
// either way there's nothing to seal to.
var ErrConsumerNoPublicKey = errors.New("consumer has no registered public key")

// ObjectMetadata is what a successful create, update, or list returns.
// Matches components.schemas.ObjectMetadata in api/openapi.yaml. JSON tags
// exist for List's --json output, the first place this type is ever
// marshaled. Slug, not ID: the server's own internal id is a separate,
// opaque value this API never exposes (alrayyes/Hush-Hush#404).
type ObjectMetadata struct {
	Slug        string   `json:"slug"`
	UsedBy      []string `json:"used_by,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags"`
	// CreatedAt through UpdatedBy are only set by List: the server leaves
	// them out of a create or update response, and CreatedBy/UpdatedBy
	// are also nil for an object that predates auditing.
	CreatedAt *time.Time `json:"created_at,omitempty"`
	CreatedBy *Actor     `json:"created_by,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	UpdatedBy *Actor     `json:"updated_by,omitempty"`
}

// Actor is who performed an audited write: Type is "session", "token" or
// "consumer_token", and ID that actor's own id.
type Actor struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// Client is a hush-hush API client. Token is the write-path bearer token;
// it's sent on every request, and reads simply ignore it server-side.
type Client struct {
	Caller string

	sdk *hushhush.Client
}

// New returns a Client backed by the hush-hush-go SDK.
func New(baseURL, token string) (*Client, error) {
	sdk, err := hushhush.NewClient(baseURL, hushhush.WithAPIKey(token))
	if err != nil {
		return nil, fmt.Errorf("build hush-hush-go client: %w", err)
	}

	return &Client{sdk: sdk}, nil
}

// Create stores value under id, sealed to usedBy's recipients before this
// is ever called - the client itself does no sealing. description is fixed
// at creation, the same as usedBy. tags are optional labels; the server
// lower-cases and de-duplicates them.
func (c *Client) Create(ctx context.Context, id string, value []byte, usedBy []string, description string, tags []string, keepReadableCopy bool) (ObjectMetadata, error) {
	req := hushhush.CreateObjectRequest{Slug: id, Value: value}
	if keepReadableCopy {
		req.KeepReadableCopy = &keepReadableCopy
	}

	if len(usedBy) > 0 {
		req.UsedBy = &usedBy
	}

	if description != "" {
		req.Description = &description
	}

	if len(tags) > 0 {
		req.Tags = &tags
	}

	meta, err := c.sdk.CreateObject(ctx, req, c.Caller)
	if err != nil {
		return ObjectMetadata{}, mapError(err)
	}

	return toObjectMetadata(meta), nil
}

// Get fetches an object's stored ciphertext exactly as sealed.
func (c *Client) Get(ctx context.Context, id string) ([]byte, error) {
	value, err := c.sdk.GetObject(ctx, id, c.Caller)
	if err != nil {
		return nil, mapError(err)
	}

	return value, nil
}

// Update replaces id's stored value, leaving its used_by metadata
// unchanged. A nil tags leaves the object's tags alone; a non-nil one
// replaces them, and an empty one clears them.
func (c *Client) Update(ctx context.Context, id string, value []byte, tags, usedBy *[]string, keepReadableCopy bool) (ObjectMetadata, error) {
	req := hushhush.UpdateObjectRequest{Value: value, Tags: tags, UsedBy: usedBy}
	if keepReadableCopy {
		req.KeepReadableCopy = &keepReadableCopy
	}

	meta, err := c.sdk.UpdateObject(ctx, id, req, c.Caller)
	if err != nil {
		return ObjectMetadata{}, mapError(err)
	}

	return toObjectMetadata(meta), nil
}

// OwnerPublicKey returns the owner's escrowed identity public key, the age
// recipient to add before sealing when the owner wants a readable copy. It
// needs a token, and returns ErrNoOwnerKey when none is registered yet.
func (c *Client) OwnerPublicKey(ctx context.Context) (string, error) {
	identity, err := c.sdk.GetOwnerIdentity(ctx)
	if err != nil {
		return "", mapError(err)
	}

	if identity.PublicKey == nil || *identity.PublicKey == "" {
		return "", ErrNoOwnerKey
	}

	return *identity.PublicKey, nil
}

// Delete permanently removes id.
func (c *Client) Delete(ctx context.Context, id string) error {
	if err := c.sdk.DeleteObject(ctx, id, c.Caller); err != nil {
		return mapError(err)
	}

	return nil
}

// List returns every stored object's metadata, sorted by id - never the
// value. Requires a credential, unlike Get: enumerating every object is a
// capability none of the other, id-scoped reads grant on their own.
func (c *Client) List(ctx context.Context, usedBy string) ([]ObjectMetadata, error) {
	metas, err := c.sdk.ListObjects(ctx, usedBy)
	if err != nil {
		return nil, mapError(err)
	}

	result := make([]ObjectMetadata, len(metas))
	for i, m := range metas {
		result[i] = toObjectMetadata(&m)
	}

	return result, nil
}

// UsedBy returns id's recorded consumers - unauthenticated, matching Get's
// auth boundary rather than List's, which needs a credential.
func (c *Client) UsedBy(ctx context.Context, id string) ([]string, error) {
	usedBy, err := c.sdk.GetObjectUsedBy(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}

	return usedBy.UsedBy, nil
}

// ConsumerPublicKey resolves name's registered age public key from the
// server's consumer directory - the lookup inject uses to seal to a
// --used-by consumer with no explicit --recipients override.
func (c *Client) ConsumerPublicKey(ctx context.Context, name string) (string, error) {
	key, err := c.sdk.GetConsumerPublicKey(ctx, name)
	if err != nil {
		return "", mapError(err)
	}

	if key == nil {
		return "", fmt.Errorf("%w: %s", ErrConsumerNoPublicKey, name)
	}

	return *key, nil
}

// AuthStatus reports whether hush-hush has an admin account bootstrapped
// yet. Matches hushhush.AuthStatus.
type AuthStatus struct {
	Bootstrapped bool `json:"bootstrapped"`
}

// AuthStatus asks whether the target server has an admin account
// bootstrapped yet - unauthenticated, matching the SDK's own AuthStatus
// doc comment: no credential is required to call it.
func (c *Client) AuthStatus(ctx context.Context) (AuthStatus, error) {
	status, err := c.sdk.AuthStatus(ctx)
	if err != nil {
		return AuthStatus{}, mapError(err)
	}

	return AuthStatus{Bootstrapped: status.Bootstrapped}, nil
}

// AuditLogEntry is one row of hush-hush's audit trail. JSON tags match
// api/openapi.yaml's AuditLogEntry shape exactly, since audit-log's
// --format json output encodes this type directly (design.md).
type AuditLogEntry struct {
	ID        int64     `json:"id"`
	Action    string    `json:"action"`
	ObjectID  string    `json:"object_id"`
	Timestamp time.Time `json:"timestamp"`
	Caller    *string   `json:"caller,omitempty"`
	IP        string    `json:"ip"`
	ActorType *string   `json:"actor_type,omitempty"`
	ActorID   *string   `json:"actor_id,omitempty"`
}

// AuditLogFilter narrows QueryAuditLog; filters combine with AND. Token
// maps onto the server's verified actor filter (a token id, or the admin
// account's own actor id) - named for the CLI's own --token flag rather
// than the SDK's "actor" vocabulary. Limit caps how many entries in total
// QueryAuditLog returns across every page it requests, not a single
// page's size; nil means fetch everything the server has.
type AuditLogFilter struct {
	ObjectID *string
	Caller   *string
	Token    *string
	Since    *time.Time
	Until    *time.Time
	Limit    *int
}

// QueryAuditLog returns every entry matching filter, oldest first. It
// pages through the server's own limit/after cursor rather than
// truncating one unpaginated response (design.md's paging decision): the
// first request's limit comes from filter.Limit capped at 500, and
// further pages are requested with after set to the previous page's last
// entry's id until filter.Limit entries have been collected or a short
// page (fewer than requested) signals nothing is left.
func (c *Client) QueryAuditLog(ctx context.Context, filter AuditLogFilter) ([]AuditLogEntry, error) {
	pageLimit := int32(auditLogPageMax)
	if filter.Limit != nil && *filter.Limit > 0 && *filter.Limit < auditLogPageMax {
		pageLimit = int32(*filter.Limit)
	}

	var (
		result []AuditLogEntry
		after  *int64
	)

	for {
		page, err := c.sdk.QueryAuditLog(ctx, hushhush.AuditLogFilter{
			ObjectID: filter.ObjectID,
			Caller:   filter.Caller,
			Actor:    filter.Token,
			From:     filter.Since,
			To:       filter.Until,
			After:    after,
			Limit:    &pageLimit,
		})
		if err != nil {
			return nil, mapError(err)
		}

		for _, e := range page {
			result = append(result, toAuditLogEntry(e))
			if filter.Limit != nil && len(result) >= *filter.Limit {
				return result, nil
			}
		}

		if len(page) < int(pageLimit) {
			return result, nil
		}

		last := page[len(page)-1].Id
		after = &last
	}
}

func toAuditLogEntry(e hushhush.AuditLogEntry) AuditLogEntry {
	entry := AuditLogEntry{
		ID:        e.Id,
		Action:    string(e.Action),
		ObjectID:  e.ObjectId,
		Timestamp: e.Timestamp,
		Caller:    e.Caller,
		IP:        e.Ip,
		ActorID:   e.ActorId,
	}

	if e.ActorType != nil {
		actorType := string(*e.ActorType)
		entry.ActorType = &actorType
	}

	return entry
}

func toObjectMetadata(m *hushhush.ObjectMetadata) ObjectMetadata {
	meta := ObjectMetadata{Slug: m.Slug, Tags: m.Tags}
	if meta.Tags == nil {
		meta.Tags = []string{}
	}

	if m.UsedBy != nil {
		meta.UsedBy = *m.UsedBy
	}

	if m.Description != nil {
		meta.Description = *m.Description
	}

	meta.CreatedAt, meta.UpdatedAt = m.CreatedAt, m.UpdatedAt

	// hush-hush-go doesn't alias its Actor type, so the fields are read
	// off each pointer directly.
	if m.CreatedBy != nil {
		meta.CreatedBy = &Actor{ID: m.CreatedBy.Id, Type: m.CreatedBy.Type}
	}

	if m.UpdatedBy != nil {
		meta.UpdatedBy = &Actor{ID: m.UpdatedBy.Id, Type: m.UpdatedBy.Type}
	}

	return meta
}

// mapError translates the SDK's *hushhush.APIError into this package's
// sentinels, so callers keep matching on client.Err* regardless of which
// transport sits underneath.
func mapError(err error) error {
	var apiErr *hushhush.APIError
	if !errors.As(err, &apiErr) {
		return fmt.Errorf("call hush-hush server: %w", err)
	}

	switch apiErr.StatusCode {
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusConflict:
		return ErrAlreadyExists
	default:
		if apiErr.Message != "" {
			return fmt.Errorf("%w %d: %s", ErrUnexpectedStatus, apiErr.StatusCode, apiErr.Message)
		}

		return fmt.Errorf("%w %d", ErrUnexpectedStatus, apiErr.StatusCode)
	}
}
