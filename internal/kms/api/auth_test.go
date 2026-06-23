package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticate(t *testing.T) {
	const authToken = "secret-token"

	t.Run("rejects missing token", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		res := httptest.NewRecorder()
		ctx := e.NewContext(req, res)

		handler := Authenticate(authToken)(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		require.NoError(t, handler(ctx))
		assert.Equal(t, http.StatusUnauthorized, res.Code)
	})

	t.Run("rejects empty expected token", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(kmsAuthHeader, authToken)
		res := httptest.NewRecorder()
		ctx := e.NewContext(req, res)

		handler := Authenticate("")(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		require.NoError(t, handler(ctx))
		assert.Equal(t, http.StatusUnauthorized, res.Code)
	})

	t.Run("allows matching token", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(kmsAuthHeader, authToken)
		res := httptest.NewRecorder()
		ctx := e.NewContext(req, res)

		handler := Authenticate(authToken)(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		require.NoError(t, handler(ctx))
		assert.Equal(t, http.StatusOK, res.Code)
	})
}
