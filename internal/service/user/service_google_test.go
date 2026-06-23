package user_test

import (
	"testing"

	"github.com/oxygenpay/oxygen/internal/auth"
	"github.com/oxygenpay/oxygen/internal/service/user"
	"github.com/oxygenpay/oxygen/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_ResolveWithGoogle(t *testing.T) {
	tc := test.NewIntegrationTest(t)

	t.Run("Handles whitelist access", func(t *testing.T) {
		t.Run("whitelist disabled", func(t *testing.T) {
			// ARRANGE
			g := &auth.GoogleUser{
				Name:          "u1",
				Sub:           "google-u1",
				Email:         "user@gmail.com",
				EmailVerified: true,
			}

			// ACT
			u, err := tc.Services.Users.ResolveWithGoogle(tc.Context, g)

			// ASSERT
			assert.NoError(t, err)
			assert.Equal(t, g.Email, u.Email)
		})

		t.Run("whitelist enabled", func(t *testing.T) {
			// ARRANGE
			// Given existing user
			existingUser, _ := tc.Must.CreateUser(t, auth.GoogleUser{Name: "u1", Email: "user@gmail.com"})

			// Given whitelist enabled
			_, err := tc.Services.Registry.Set(tc.Context, "registration.is_whitelist_only", "true")
			require.NoError(t, err)

			g := &auth.GoogleUser{
				Name:          "u2",
				Sub:           "google-u2",
				Email:         "user2@gmail.com",
				EmailVerified: true,
			}

			// ACT
			_, err = tc.Services.Users.ResolveWithGoogle(tc.Context, g)

			// ASSERT
			assert.ErrorIs(t, user.ErrRestricted, err)

			t.Run("existing user work", func(t *testing.T) {
				// ACT
				u, err := tc.Services.Users.ResolveWithGoogle(tc.Context, &auth.GoogleUser{
					Name:          existingUser.Name,
					Sub:           *existingUser.GoogleID,
					Email:         existingUser.Email,
					EmailVerified: false,
				})

				// ASSERT
				assert.NoError(t, err)
				assert.Equal(t, existingUser.Email, u.Email)
			})

			t.Run("access allowed", func(t *testing.T) {
				// ARRANGE
				// Given a whitelist
				_, err := tc.Services.Registry.Set(tc.Context, "registration.whitelist", "hey@o2pay.co,test@me.com")
				require.NoError(t, err)

				// ACT
				_, err = tc.Services.Users.ResolveWithGoogle(tc.Context, &auth.GoogleUser{
					Name:          "John",
					Sub:           "abc123",
					Email:         "hey@o2pay.co",
					EmailVerified: true,
				})

				// ASSERT
				assert.NoError(t, err)
			})
		})

		t.Run("rejects unverified email for account linking", func(t *testing.T) {
			// ARRANGE
			existingUser, _ := tc.Must.CreateUser(t, auth.GoogleUser{
				Name:          "victim",
				Sub:           "victim-google-sub",
				Email:         "victim@gmail.com",
				EmailVerified: true,
			})

			// ACT
			_, err := tc.Services.Users.ResolveWithGoogle(tc.Context, &auth.GoogleUser{
				Name:          "attacker",
				Sub:           "attacker-google-sub",
				Email:         existingUser.Email,
				EmailVerified: false,
			})

			// ASSERT
			assert.ErrorIs(t, err, user.ErrEmailNotVerified)

			unchangedUser, err := tc.Services.Users.GetByID(tc.Context, existingUser.ID)
			require.NoError(t, err)
			require.NotNil(t, unchangedUser.GoogleID)
			assert.Equal(t, *existingUser.GoogleID, *unchangedUser.GoogleID)
		})
	})
}
