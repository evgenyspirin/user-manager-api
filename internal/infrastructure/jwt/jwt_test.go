package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndValidate_Success(t *testing.T) {
	s := New("super-secret")
	userID := "u-123"
	role := "admin"

	tok, err := s.GenerateJWT(userID, role, time.Hour)
	require.NoError(t, err, "GenerateJWT should not error")
	require.NotEmpty(t, tok, "token must not be empty")

	claims, err := s.ValidateToken(tok)
	require.NoError(t, err, "ValidateToken should not error for fresh token")
	require.NotNil(t, claims)

	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, role, claims.Role)
	require.NotNil(t, claims.ExpiresAt)
	assert.True(t, claims.ExpiresAt.Time.After(time.Now().Add(-1*time.Second)))
}

func TestValidateToken_Table(t *testing.T) {
	type fields struct {
		secret string
	}
	type want struct {
		ok    bool
		err   string
		check func(t *testing.T, c *Claims)
	}

	makeToken := func(secret string, exp time.Duration) string {
		s := New(secret)
		tok, err := s.GenerateJWT("user-42", "worker", exp)
		require.NoError(t, err)
		return tok
	}

	tests := []struct {
		name   string
		fields fields
		token  string
		want   want
	}{
		{
			name:   "valid token",
			fields: fields{secret: "k1"},
			token:  makeToken("k1", 5*time.Minute),
			want: want{
				ok:  true,
				err: "",
				check: func(t *testing.T, c *Claims) {
					assert.Equal(t, "user-42", c.UserID)
					assert.Equal(t, "worker", c.Role)
					require.NotNil(t, c.ExpiresAt)
					assert.True(t, c.ExpiresAt.Time.After(time.Now().Add(-1*time.Second)))
				},
			},
		},
		{
			name:   "invalid secret (signature mismatch)",
			fields: fields{secret: "k2"},
			token:  makeToken("k1", 5*time.Minute),
			want: want{
				ok:  false,
				err: "invalid token",
			},
		},
		{
			name:   "expired token",
			fields: fields{secret: "k1"},
			token:  makeToken("k1", -1*time.Minute),
			want: want{
				ok:  false,
				err: "invalid token",
			},
		},
		{
			name:   "malformed token string",
			fields: fields{secret: "k1"},
			token:  "not-a-jwt",
			want: want{
				ok:  false,
				err: "invalid token",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.fields.secret)

			claims, err := s.ValidateToken(tt.token)
			if tt.want.ok {
				require.NoError(t, err)
				require.NotNil(t, claims)
				if tt.want.check != nil {
					tt.want.check(t, claims)
				}
			} else {
				require.Error(t, err)
				assert.EqualError(t, err, tt.want.err)
				assert.Nil(t, claims)
			}
		})
	}
}
