package http

import (
	"net/http/httptest"
	"testing"
	"testing/fstest"

	stdhttp "net/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedFrontendRootRedirectsToDashboard(t *testing.T) {
	files := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}

	srv := New(Config{Address: "127.0.0.1", Port: "0"}, false, WithEmbeddedFrontend(files, files))

	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.Echo().ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusTemporaryRedirect, rec.Code)
	assert.Equal(t, "/dashboard/", rec.Header().Get("Location"))
}

func TestEmbeddedFrontendDoesNotFallbackMissingAssetsToIndex(t *testing.T) {
	files := fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<html></html>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('ok')")},
	}

	srv := New(Config{Address: "127.0.0.1", Port: "0"}, false, WithEmbeddedFrontend(files, files))

	req := httptest.NewRequest(stdhttp.MethodGet, "/p/assets/missing.js", nil)
	rec := httptest.NewRecorder()

	srv.Echo().ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusNotFound, rec.Code)
	assert.NotEqual(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestEmbeddedFrontendServesExistingAssets(t *testing.T) {
	files := fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<html></html>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('ok')")},
	}

	srv := New(Config{Address: "127.0.0.1", Port: "0"}, false, WithEmbeddedFrontend(files, files))

	req := httptest.NewRequest(stdhttp.MethodGet, "/p/assets/app.js", nil)
	rec := httptest.NewRecorder()

	srv.Echo().ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	assert.Equal(t, "console.log('ok')", rec.Body.String())
}
