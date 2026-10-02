package testserver_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/stretchr/testify/require"
)

func TestListObjectsRequiresToken(t *testing.T) {
	t.Parallel()

	srv, _, _ := testserver.New(t)

	resp, err := http.Get(srv.URL + "/objects") //nolint:noctx // test-only, no deadline needed
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestListObjectsEmptyStoreReturnsEmptyArray(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	body := getObjects(t, srv.URL, token, "")

	require.JSONEq(t, "[]", body)
}

func TestListObjectsReturnsStoredMetadataSortedByID(t *testing.T) {
	t.Parallel()

	srv, store, token := testserver.New(t)

	require.NoError(t, store.CreateObject(t.Context(), "zebra", []byte("v1"), []string{"homelab/vps-docker"}, "z desc"))
	require.NoError(t, store.CreateObject(t.Context(), "apple", []byte("v2"), nil, ""))

	require.JSONEq(t,
		`[{"slug":"apple","tags":[]},{"slug":"zebra","used_by":["homelab/vps-docker"],"description":"z desc","tags":[]}]`,
		getObjects(t, srv.URL, token, ""),
	)
}

func TestListObjectsFiltersByUsedBy(t *testing.T) {
	t.Parallel()

	srv, store, token := testserver.New(t)

	require.NoError(t, store.CreateObject(t.Context(), "matched", []byte("v"), []string{"homelab/vps-docker"}, ""))
	require.NoError(t, store.CreateObject(t.Context(), "unmatched", []byte("v"), []string{"other/repo"}, ""))

	require.JSONEq(t, `[{"slug":"matched","used_by":["homelab/vps-docker"],"tags":[]}]`, getObjects(t, srv.URL, token, "homelab/vps-docker"))
}

func TestQueryAuditLogRequiresNoToken(t *testing.T) {
	t.Parallel()

	srv, _, _ := testserver.New(t)

	resp, err := http.Get(srv.URL + "/audit-log") //nolint:noctx // test-only, no deadline needed
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestQueryAuditLogEmptyLogReturnsEmptyArray(t *testing.T) {
	t.Parallel()

	srv, _, _ := testserver.New(t)

	require.JSONEq(t, "[]", getAuditLog(t, srv.URL, ""))
}

func TestQueryAuditLogRejectsAnInvalidObjectID(t *testing.T) {
	t.Parallel()

	srv, _, _ := testserver.New(t)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/audit-log?object_id=Not_Valid!", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAuthStatusDefaultsToBootstrapped(t *testing.T) {
	t.Parallel()

	srv, _, _ := testserver.New(t)

	resp, err := http.Get(srv.URL + "/auth/status") //nolint:noctx // test-only, no deadline needed
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"bootstrapped":true}`, string(body))
}

func TestAuthStatusReflectsSetBootstrapped(t *testing.T) {
	t.Parallel()

	srv, store, _ := testserver.New(t)
	store.SetBootstrapped(false)

	resp, err := http.Get(srv.URL + "/auth/status") //nolint:noctx // test-only, no deadline needed
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"bootstrapped":false}`, string(body))
}

func TestAuthStatusRequiresNoToken(t *testing.T) {
	t.Parallel()

	srv, _, _ := testserver.New(t)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/auth/status", nil)
	require.NoError(t, err)
	// Deliberately no Authorization header - unlike list/inject/update/delete.

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListConsumersRequiresToken(t *testing.T) {
	t.Parallel()

	srv, _, _ := testserver.New(t)

	resp, err := http.Get(srv.URL + "/consumers") //nolint:noctx // test-only, no deadline needed
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestListConsumersEmptyDirectoryReturnsEmptyArray(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	require.JSONEq(t, "[]", getConsumers(t, srv.URL, token, ""))
}

func TestListConsumersIncludesNamesImplicitlyRecordedByUsedBy(t *testing.T) {
	t.Parallel()

	srv, store, token := testserver.New(t)

	require.NoError(t, store.CreateObject(t.Context(), "secret-1", []byte("v"), []string{"homelab/vps-docker"}, ""))

	require.JSONEq(t, `["homelab/vps-docker"]`, getConsumers(t, srv.URL, token, ""))
}

func TestListConsumersWithAFilterSwitchesToThePaginatedShape(t *testing.T) {
	t.Parallel()

	srv, store, token := testserver.New(t)

	require.NoError(t, store.CreateObject(t.Context(), "secret-1", []byte("v"), []string{"homelab/vps-docker"}, ""))

	require.JSONEq(t,
		`{"consumers":[{"name":"homelab/vps-docker","secret_count":1}],"total":1}`,
		getConsumers(t, srv.URL, token, "q=vps"),
	)
}

func TestAddConsumerCreatesAnEntryWithNoKeyOrSecrets(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	resp := postConsumer(t, srv.URL, token, "ci/pipeline")
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"ci/pipeline","secret_count":0}`, string(body))
}

func TestAddConsumerRejectsADuplicateName(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	first := postConsumer(t, srv.URL, token, "homelab/vps-docker")
	_ = first.Body.Close()

	resp := postConsumer(t, srv.URL, token, "homelab/vps-docker")
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestUpdateConsumerRegistersAPublicKey(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	add := postConsumer(t, srv.URL, token, "homelab/vps-docker")
	_ = add.Body.Close()

	resp := patchConsumer(t, srv.URL, token, "homelab/vps-docker", `{"public_key":"age1abc"}`)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"homelab/vps-docker","public_key":"age1abc","secret_count":0}`, string(body))
}

func TestUpdateConsumerRegisteringAKeyOnAnUnknownNameUpsertsIt(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	resp := patchConsumer(t, srv.URL, token, "new-consumer", `{"public_key":"age1abc"}`)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUpdateConsumerRenamingAnUnknownNameIs404(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	resp := patchConsumer(t, srv.URL, token, "unknown", `{"name":"renamed"}`)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeleteConsumerStripsItFromEveryObjectsUsedBy(t *testing.T) {
	t.Parallel()

	srv, store, token := testserver.New(t)

	require.NoError(t, store.CreateObject(t.Context(), "secret-1", []byte("v"), []string{"homelab/vps-docker"}, ""))

	resp := deleteConsumer(t, srv.URL, token, "homelab/vps-docker")
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	obj, err := store.GetObject(t.Context(), "secret-1")
	require.NoError(t, err)
	require.Empty(t, obj.UsedBy)
}

func TestDeleteConsumerUnknownNameIs404(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	resp := deleteConsumer(t, srv.URL, token, "unknown")
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestCreateConsumerTokenRequiresToken(t *testing.T) {
	t.Parallel()

	srv, _, _ := testserver.New(t)

	resp, err := http.Post(srv.URL+"/consumer-tokens", "application/json", //nolint:noctx // test-only, no deadline needed
		strings.NewReader(`{"consumer":"homelab/vps-docker","ttl_seconds":3600}`))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestCreateConsumerTokenRejectsAMissingConsumer(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	resp := postConsumerToken(t, srv.URL, token, `{"ttl_seconds":3600}`)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateConsumerTokenReturnsTheRawValueOnceAndListDoesNot(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	created := postConsumerToken(t, srv.URL, token, `{"consumer":"homelab/vps-docker","description":"ci reader","ttl_seconds":3600}`)
	defer func() { _ = created.Body.Close() }()

	require.Equal(t, http.StatusCreated, created.StatusCode)

	var withValue testserver.ConsumerTokenWithValue
	require.NoError(t, json.NewDecoder(created.Body).Decode(&withValue))
	require.NotEmpty(t, withValue.Value)
	require.Equal(t, "homelab/vps-docker", withValue.Consumer)

	resp := getConsumerTokens(t, srv.URL, token)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotContains(t, string(body), withValue.Value, "the raw value must never appear in the list response")
}

func TestRotateConsumerTokenUnknownIDIs404(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	resp := rotateConsumerToken(t, srv.URL, token, "unknown", `{"ttl_seconds":3600}`)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRotateConsumerTokenIssuesANewValue(t *testing.T) {
	t.Parallel()

	srv, store, token := testserver.New(t)

	minted := store.IssueConsumerToken("homelab/vps-docker", "", time.Hour)

	resp := rotateConsumerToken(t, srv.URL, token, minted.ID, `{"ttl_seconds":3600}`)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var rotated testserver.ConsumerTokenWithValue
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&rotated))
	require.Equal(t, minted.ID, rotated.ID)
	require.NotEqual(t, minted.Value, rotated.Value)
}

func TestRevokeConsumerTokenUnknownIDStillReturns204(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	resp := deleteConsumerToken(t, srv.URL, token, "unknown")
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestPurgeConsumerTokenStillActiveIs409(t *testing.T) {
	t.Parallel()

	srv, store, token := testserver.New(t)

	minted := store.IssueConsumerToken("homelab/vps-docker", "", time.Hour)

	resp := purgeConsumerToken(t, srv.URL, token, minted.ID)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestPurgeConsumerTokenUnknownIDIs404(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	resp := purgeConsumerToken(t, srv.URL, token, "unknown")
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestPurgeConsumerTokenRevokedSucceeds(t *testing.T) {
	t.Parallel()

	srv, store, token := testserver.New(t)

	minted := store.IssueConsumerToken("homelab/vps-docker", "", time.Hour)
	store.RevokeConsumerToken(minted.ID)

	resp := purgeConsumerToken(t, srv.URL, token, minted.ID)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func postConsumerToken(t *testing.T, baseURL, token, body string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, baseURL+"/consumer-tokens", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

func getConsumerTokens(t *testing.T, baseURL, token string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, baseURL+"/consumer-tokens", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

func rotateConsumerToken(t *testing.T, baseURL, token, id, body string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, baseURL+"/consumer-tokens/"+id+"/rotate", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

func deleteConsumerToken(t *testing.T, baseURL, token, id string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, baseURL+"/consumer-tokens/"+id, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

func purgeConsumerToken(t *testing.T, baseURL, token, id string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, baseURL+"/consumer-tokens/"+id+"/purge", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

func getConsumers(t *testing.T, baseURL, token, query string) string {
	t.Helper()

	url := baseURL + "/consumers"
	if query != "" {
		url += "?" + query
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return string(body)
}

func postConsumer(t *testing.T, baseURL, token, name string) *http.Response {
	t.Helper()

	body := fmt.Sprintf(`{"name":%q}`, name)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, baseURL+"/consumers", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

func patchConsumer(t *testing.T, baseURL, token, name, body string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPatch, baseURL+"/consumers/"+name, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

func deleteConsumer(t *testing.T, baseURL, token, name string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, baseURL+"/consumers/"+name, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

func getAuditLog(t *testing.T, baseURL, query string) string {
	t.Helper()

	url := baseURL + "/audit-log"
	if query != "" {
		url += "?" + query
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return string(body)
}

func getObjects(t *testing.T, baseURL, token, usedBy string) string {
	t.Helper()

	url := baseURL + "/objects"
	if usedBy != "" {
		url += "?used_by=" + usedBy
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return string(body)
}
