package testserver_test

import (
	"io"
	"net/http"
	"testing"

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
		`[{"id":"apple"},{"id":"zebra","used_by":["homelab/vps-docker"],"description":"z desc"}]`,
		getObjects(t, srv.URL, token, ""),
	)
}

func TestListObjectsFiltersByUsedBy(t *testing.T) {
	t.Parallel()

	srv, store, token := testserver.New(t)

	require.NoError(t, store.CreateObject(t.Context(), "matched", []byte("v"), []string{"homelab/vps-docker"}, ""))
	require.NoError(t, store.CreateObject(t.Context(), "unmatched", []byte("v"), []string{"other/repo"}, ""))

	require.JSONEq(t, `[{"id":"matched","used_by":["homelab/vps-docker"]}]`, getObjects(t, srv.URL, token, "homelab/vps-docker"))
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
