package api

import (
	"crypto/subtle"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/oxygenpay/oxygen/pkg/api-kms/v1/model"
)

const kmsAuthHeader = "X-O2PAY-KMS-TOKEN"

func Authenticate(expectedToken string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			providedToken := c.Request().Header.Get(kmsAuthHeader)
			if !tokensMatch(providedToken, expectedToken) {
				return c.JSON(http.StatusUnauthorized, &model.ErrorResponse{
					Message: "Unauthorized",
					Status:  "auth_error",
				})
			}

			return next(c)
		}
	}
}

func tokensMatch(providedToken, expectedToken string) bool {
	if providedToken == "" || expectedToken == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(providedToken), []byte(expectedToken)) == 1
}
