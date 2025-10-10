// auth_controller_test.go
package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"user-manager-api/internal/application/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"user-manager-api/internal/application/ports"
	"user-manager-api/internal/interface/api/rest/dto/auth"

	domain "user-manager-api/internal/domain/user"
)

type fakeAuthService struct {
	GenerateTokenFunc func(u *domain.User, password string) (string, error)
}

func (f *fakeAuthService) GenerateToken(u *domain.User, password string) (string, error) {
	return f.GenerateTokenFunc(u, password)
}

func newRouterWithController(t *testing.T, us ports.UserService, as ports.Auth) (*gin.Engine, *AuthController) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	ac := &AuthController{
		logger:      zap.NewNop(),
		userService: us,
		authService: as,
	}
	r.POST("/login", ac.LoginHandler)
	return r, ac
}

func doPOST(t *testing.T, r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var b []byte
	switch v := body.(type) {
	case string:
		b = []byte(v)
	default:
		var err error
		b, err = json.Marshal(v)
		require.NoError(t, err)
	}

	req, err := http.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func validLogin() auth.LoginRequest {
	return auth.LoginRequest{
		Email:    "user@example.com",
		Password: "VeryStrongPassw0rd!",
	}
}

func TestAuthController_LoginHandler(t *testing.T) {
	type fields struct {
		findByEmail   func(ctx context.Context, email string) (*domain.User, error)
		generateToken func(u *domain.User, password string) (string, error)
	}
	type want struct {
		code        int
		oneOfCodes  []int
		jsonEq      map[string]any
		jsonHasKeys []string
	}

	tests := []struct {
		name   string
		body   any
		fields fields
		want   want
	}{
		{
			name: "invalid JSON",
			body: "{bad json",
			fields: fields{
				findByEmail:   func(ctx context.Context, email string) (*domain.User, error) { return nil, nil },
				generateToken: func(u *domain.User, password string) (string, error) { return "", nil },
			},
			want: want{
				code:        http.StatusBadRequest,
				jsonEq:      map[string]any{"error": "invalid json"},
				jsonHasKeys: []string{"error"},
			},
		},
		{
			name: "validation error",
			body: auth.LoginRequest{Email: "not-an-email", Password: ""},
			fields: fields{
				findByEmail:   func(ctx context.Context, email string) (*domain.User, error) { return nil, nil },
				generateToken: func(u *domain.User, password string) (string, error) { return "", nil },
			},
			want: want{
				code:        http.StatusBadRequest,
				jsonHasKeys: []string{"error", "details"},
			},
		},
		{
			name: "FindByEmail error -> 500",
			body: validLogin(),
			fields: fields{
				findByEmail: func(ctx context.Context, email string) (*domain.User, error) {
					return nil, errors.New("db error")
				},
				generateToken: func(u *domain.User, password string) (string, error) { return "", nil },
			},
			want: want{
				code:        http.StatusInternalServerError,
				jsonEq:      map[string]any{"error": "failed to get a user"},
				jsonHasKeys: []string{"error"},
			},
		},
		{
			name: "user not found -> 404",
			body: validLogin(),
			fields: fields{
				findByEmail:   func(ctx context.Context, email string) (*domain.User, error) { return nil, nil },
				generateToken: func(u *domain.User, password string) (string, error) { return "", nil },
			},
			want: want{
				code:        http.StatusNotFound,
				jsonEq:      map[string]any{"error": "user not found"},
				jsonHasKeys: []string{"error"},
			},
		},
		{
			name: "GenerateToken ErrInvalidCredentials -> 401",
			body: validLogin(),
			fields: fields{
				findByEmail: func(ctx context.Context, email string) (*domain.User, error) {
					return &domain.User{}, nil
				},
				generateToken: func(u *domain.User, password string) (string, error) {
					return "", services.ErrInvalidCredentials
				},
			},
			want: want{
				oneOfCodes:  []int{http.StatusUnauthorized},
				jsonHasKeys: []string{"error"},
			},
		},
		{
			name: "GenerateToken ErrFailedToGenerateToken -> 500",
			body: validLogin(),
			fields: fields{
				findByEmail: func(ctx context.Context, email string) (*domain.User, error) {
					return &domain.User{}, nil
				},
				generateToken: func(u *domain.User, password string) (string, error) {
					return "", services.ErrFailedToGenerateToken
				},
			},
			want: want{
				oneOfCodes:  []int{http.StatusInternalServerError},
				jsonHasKeys: []string{"error"},
			},
		},
		{
			name: "success",
			body: validLogin(),
			fields: fields{
				findByEmail: func(ctx context.Context, email string) (*domain.User, error) {
					return &domain.User{}, nil
				},
				generateToken: func(u *domain.User, password string) (string, error) {
					return "tok_123", nil
				},
			},
			want: want{
				code:        http.StatusOK,
				jsonEq:      map[string]any{"access_token": "tok_123", "token_type": "Bearer"},
				jsonHasKeys: []string{"access_token", "token_type"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			us := &FakeUserService{
				FindByEmailFunc:  tt.fields.findByEmail,
				FindUserByIDFunc: func(ctx context.Context, uuid domain.UUID) (*domain.User, error) { return nil, errors.New("not used") },
				FindUsersFunc:    func(ctx context.Context, page int) (domain.Users, error) { return nil, errors.New("not used") },
				CreateUserFunc:   func(ctx context.Context, u domain.User) (*domain.User, error) { return nil, errors.New("not used") },
				UpdateUserFunc:   func(ctx context.Context, u domain.User) (*domain.User, error) { return nil, errors.New("not used") },
				DeleteUserFunc:   func(ctx context.Context, userUUID domain.UUID) error { return errors.New("not used") },
			}
			as := &fakeAuthService{GenerateTokenFunc: tt.fields.generateToken}

			r, _ := newRouterWithController(t, us, as)
			rr := doPOST(t, r, "/login", tt.body)

			if len(tt.want.oneOfCodes) > 0 {
				ok := false
				for _, c := range tt.want.oneOfCodes {
					if rr.Code == c {
						ok = true
						break
					}
				}
				require.Truef(t, ok, "status %d not in %v", rr.Code, tt.want.oneOfCodes)

				assert.Contains(t, rr.Body.String(), "error")
				return
			} else {
				require.Equal(t, tt.want.code, rr.Code)
			}

			var resp map[string]any
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

			for k, v := range tt.want.jsonEq {
				assert.Equal(t, v, resp[k], "field %q mismatch", k)
			}
			for _, k := range tt.want.jsonHasKeys {
				assert.Contains(t, resp, k, "expected key %q", k)
			}
		})
	}
}
