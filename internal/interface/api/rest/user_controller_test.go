package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"user-manager-api/internal/application/ports"
	domain "user-manager-api/internal/domain/user"
	userDB "user-manager-api/internal/infrastructure/db/postgres/user"
	jwtSvc "user-manager-api/internal/infrastructure/jwt"
	"user-manager-api/internal/interface/api/rest/dto/user"
	"user-manager-api/internal/interface/api/rest/middleware"
)

type FakeUserService struct {
	FindUserByIDFunc func(ctx context.Context, id domain.UUID) (*domain.User, error)
	FindByEmailFunc  func(ctx context.Context, email string) (*domain.User, error)
	FindUsersFunc    func(ctx context.Context, page int) (domain.Users, error)
	CreateUserFunc   func(ctx context.Context, u domain.User) (*domain.User, error)
	UpdateUserFunc   func(ctx context.Context, u domain.User) (*domain.User, error)
	DeleteUserFunc   func(ctx context.Context, userUUID domain.UUID) error
}

func (f *FakeUserService) FindUserByID(ctx context.Context, id domain.UUID) (*domain.User, error) {
	if f.FindUserByIDFunc == nil {
		return nil, errors.New("not used")
	}
	return f.FindUserByIDFunc(ctx, id)
}
func (f *FakeUserService) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if f.FindByEmailFunc == nil {
		return nil, errors.New("not used")
	}
	return f.FindByEmailFunc(ctx, email)
}
func (f *FakeUserService) FindUsers(ctx context.Context, page int) (domain.Users, error) {
	if f.FindUsersFunc == nil {
		return nil, errors.New("not used")
	}
	return f.FindUsersFunc(ctx, page)
}
func (f *FakeUserService) CreateUser(ctx context.Context, u domain.User) (*domain.User, error) {
	if f.CreateUserFunc == nil {
		return nil, errors.New("not used")
	}
	return f.CreateUserFunc(ctx, u)
}
func (f *FakeUserService) UpdateUser(ctx context.Context, u domain.User) (*domain.User, error) {
	if f.UpdateUserFunc == nil {
		return nil, errors.New("not used")
	}
	return f.UpdateUserFunc(ctx, u)
}
func (f *FakeUserService) DeleteUser(ctx context.Context, userUUID domain.UUID) error {
	if f.DeleteUserFunc == nil {
		return errors.New("not used")
	}
	return f.DeleteUserFunc(ctx, userUUID)
}

func setupRouter(t *testing.T, us ports.UserService, withJWT bool) (*gin.Engine, *UserController, *jwtSvc.Service, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	logger := zap.NewNop()
	secret := "test-secret"
	j := jwtSvc.New(secret)

	uc := &UserController{
		userService: us,
		logger:      logger,
	}

	r.GET("/users", uc.GetUsersHandler)
	r.GET("/users/:user_id", uc.GetUserHandler)
	if withJWT {
		r.POST("/users", middleware.AuthMiddleware(j), uc.CreateUserHandler)
		r.PUT("/users/:user_id", middleware.AuthMiddleware(j), uc.UpdateUserHandler)
		r.DELETE("/users/:user_id", middleware.AuthMiddleware(j), uc.DeleteUserHandler)
	} else {
		r.POST("/users", uc.CreateUserHandler)
		r.PUT("/users/:user_id", uc.UpdateUserHandler)
		r.DELETE("/users/:user_id", uc.DeleteUserHandler)
	}

	return r, uc, j, secret
}

func doReq(t *testing.T, r *gin.Engine, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var buf *bytes.Reader
	switch v := body.(type) {
	case nil:
		buf = bytes.NewReader(nil)
	case string:
		buf = bytes.NewReader([]byte(v))
	default:
		b, err := json.Marshal(v)
		require.NoError(t, err)
		buf = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, path, buf)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func validUserRequest() user.Request {
	return user.Request{
		Email:     "john.doe@example.com",
		Name:      "John",
		Lastname:  "Doe",
		BirthDate: time.Now().AddDate(-25, 0, 0).Format("2006-01-02"),
		Phone:     "+33612345678",
	}
}

func someDomainUser() *domain.User {
	return &domain.User{
		UUID:      uuid.New(),
		Email:     "john.doe@example.com",
		Name:      "John",
		Lastname:  "Doe",
		BirthDate: time.Now().AddDate(-25, 0, 0),
		Phone:     "+33612345678",
		Role:      "worker",
	}
}

func SignJWT(secret, userID, role string, exp time.Duration) (string, error) {
	type Claims struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
		jwtv5.RegisteredClaims
	}
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwtv5.RegisteredClaims{
			ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(exp)),
		},
	}
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func TestUserController_GetUsersHandler(t *testing.T) {
	tests := []struct {
		name       string
		pageQuery  string
		mockUS     func() ports.UserService
		wantStatus int
		wantErr    string
	}{
		{
			name:      "500 when service fails",
			pageQuery: "1",
			mockUS: func() ports.UserService {
				return &FakeUserService{
					FindUsersFunc: func(ctx context.Context, page int) (domain.Users, error) {
						return nil, errors.New("db error")
					},
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to get users",
		},
		{
			name:      "200 success",
			pageQuery: "2",
			mockUS: func() ports.UserService {
				return &FakeUserService{
					FindUsersFunc: func(ctx context.Context, page int) (domain.Users, error) {
						return domain.Users{someDomainUser()}, nil
					},
				}
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r, _, _, _ := setupRouter(t, tt.mockUS(), false)
			rr := doReq(t, r, http.MethodGet, "/users?page="+tt.pageQuery, nil, nil)
			require.Equal(t, tt.wantStatus, rr.Code)

			if tt.wantErr != "" {
				var resp map[string]any
				_ = json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.Equal(t, tt.wantErr, resp["error"])
			}
		})
	}
}

func TestUserController_GetUserHandler(t *testing.T) {
	okID := uuid.New()

	tests := []struct {
		name       string
		userID     string
		mockUS     func() ports.UserService
		wantStatus int
		wantErr    string
	}{
		{
			name:       "400 invalid uuid",
			userID:     "not-a-uuid",
			mockUS:     func() ports.UserService { return &FakeUserService{} },
			wantStatus: http.StatusBadRequest,
			wantErr:    "user_id must be a valid UUID",
		},
		{
			name:   "500 service error",
			userID: okID.String(),
			mockUS: func() ports.UserService {
				return &FakeUserService{
					FindUserByIDFunc: func(ctx context.Context, id domain.UUID) (*domain.User, error) {
						return nil, errors.New("db error")
					},
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to get a user",
		},
		{
			name:   "404 not found",
			userID: okID.String(),
			mockUS: func() ports.UserService {
				return &FakeUserService{
					FindUserByIDFunc: func(ctx context.Context, id domain.UUID) (*domain.User, error) {
						return nil, nil
					},
				}
			},
			wantStatus: http.StatusNotFound,
			wantErr:    "user not found",
		},
		{
			name:   "200 success",
			userID: okID.String(),
			mockUS: func() ports.UserService {
				u := someDomainUser()
				u.UUID = okID
				return &FakeUserService{
					FindUserByIDFunc: func(ctx context.Context, id domain.UUID) (*domain.User, error) {
						return u, nil
					},
				}
			},
			wantStatus: http.StatusOK,
			wantErr:    "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r, _, _, _ := setupRouter(t, tt.mockUS(), false)
			rr := doReq(t, r, http.MethodGet, "/users/"+tt.userID, nil, nil)
			require.Equal(t, tt.wantStatus, rr.Code)

			if tt.wantErr != "" {
				var resp map[string]any
				_ = json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.Equal(t, tt.wantErr, resp["error"])
			}
		})
	}
}

func TestUserController_CreateUserHandler(t *testing.T) {
	validReq := validUserRequest()

	tests := []struct {
		name       string
		headers    map[string]string
		body       any
		mockUS     func() ports.UserService
		wantStatus int
		wantErr    string
	}{
		{
			name:       "401 missing auth header",
			headers:    nil,
			body:       validReq,
			mockUS:     func() ports.UserService { return &FakeUserService{} },
			wantStatus: http.StatusUnauthorized,
			wantErr:    "missing Authorization header",
		},
		{
			name: "401 invalid format",
			headers: map[string]string{
				"Authorization": "Token something",
			},
			body:       validReq,
			mockUS:     func() ports.UserService { return &FakeUserService{} },
			wantStatus: http.StatusUnauthorized,
			wantErr:    "invalid token format",
		},
		{
			name: "401 invalid token signature",
			headers: func() map[string]string {
				tok, _ := SignJWT("other-secret", "123", "admin", time.Hour)
				return map[string]string{"Authorization": "Bearer " + tok}
			}(),
			body:       validReq,
			mockUS:     func() ports.UserService { return &FakeUserService{} },
			wantStatus: http.StatusUnauthorized,
			wantErr:    "invalid token",
		},
		{
			name: "400 invalid JSON",
			headers: func() map[string]string {
				tok, _ := SignJWT("test-secret", "123", "admin", time.Hour)
				return map[string]string{"Authorization": "Bearer " + tok}
			}(),
			body:       "{bad json",
			mockUS:     func() ports.UserService { return &FakeUserService{} },
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid request body",
		},
		{
			name: "400 validation error",
			headers: func() map[string]string {
				tok, _ := SignJWT("test-secret", "123", "admin", time.Hour)
				return map[string]string{"Authorization": "Bearer " + tok}
			}(),
			body: user.Request{
				Email:     "bad",
				Name:      "",
				Lastname:  "",
				BirthDate: "2020-01-01",
				Phone:     "123",
			},
			mockUS:     func() ports.UserService { return &FakeUserService{} },
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid request body",
		},
		{
			name: "409 email already exists",
			headers: func() map[string]string {
				tok, _ := SignJWT("test-secret", "123", "admin", time.Hour)
				return map[string]string{"Authorization": "Bearer " + tok}
			}(),
			body: validReq,
			mockUS: func() ports.UserService {
				return &FakeUserService{
					CreateUserFunc: func(ctx context.Context, du domain.User) (*domain.User, error) {
						return nil, userDB.ErrEmailAlreadyExists
					},
				}
			},
			wantStatus: http.StatusConflict,
			wantErr:    "",
		},
		{
			name: "500 service error",
			headers: func() map[string]string {
				tok, _ := SignJWT("test-secret", "123", "admin", time.Hour)
				return map[string]string{"Authorization": "Bearer " + tok}
			}(),
			body: validReq,
			mockUS: func() ports.UserService {
				return &FakeUserService{
					CreateUserFunc: func(ctx context.Context, du domain.User) (*domain.User, error) {
						return nil, errors.New("db error")
					},
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "201 success",
			headers: func() map[string]string {
				tok, _ := SignJWT("test-secret", "123", "admin", time.Hour)
				return map[string]string{"Authorization": "Bearer " + tok}
			}(),
			body: validReq,
			mockUS: func() ports.UserService {
				u := someDomainUser()
				return &FakeUserService{
					CreateUserFunc: func(ctx context.Context, du domain.User) (*domain.User, error) {
						assert.Equal(t, validReq.Email, du.Email)
						return u, nil
					},
				}
			},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r, _, _, _ := setupRouter(t, tt.mockUS(), true)
			rr := doReq(t, r, http.MethodPost, "/users", tt.body, tt.headers)
			require.Equal(t, tt.wantStatus, rr.Code)

			if tt.wantErr != "" {
				var resp map[string]any
				_ = json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.Equal(t, tt.wantErr, resp["error"])
			}
		})
	}
}

func TestUserController_UpdateUserHandler(t *testing.T) {
	id := uuid.New()
	validReq := validUserRequest()

	authHeader := func() map[string]string {
		tok, _ := SignJWT("test-secret", "123", "admin", time.Hour)
		return map[string]string{"Authorization": "Bearer " + tok}
	}

	tests := []struct {
		name       string
		userID     string
		headers    map[string]string
		body       any
		mockUS     func() ports.UserService
		wantStatus int
		wantErr    string
	}{
		{
			name:       "401 missing header",
			userID:     id.String(),
			headers:    nil,
			body:       validReq,
			mockUS:     func() ports.UserService { return &FakeUserService{} },
			wantStatus: http.StatusUnauthorized,
			wantErr:    "missing Authorization header",
		},
		{
			name:       "400 invalid uuid",
			userID:     "not-uuid",
			headers:    authHeader(),
			body:       validReq,
			mockUS:     func() ports.UserService { return &FakeUserService{} },
			wantStatus: http.StatusBadRequest,
			wantErr:    "user_id must be a valid UUID",
		},
		{
			name:       "400 invalid JSON",
			userID:     id.String(),
			headers:    authHeader(),
			body:       "{bad json",
			mockUS:     func() ports.UserService { return &FakeUserService{} },
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid request body",
		},
		{
			name:    "400 validation error",
			userID:  id.String(),
			headers: authHeader(),
			body: user.Request{
				Email:     "bad",
				Name:      "",
				Lastname:  "",
				BirthDate: "2020-01-01",
				Phone:     "123",
			},
			mockUS:     func() ports.UserService { return &FakeUserService{} },
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid request body",
		},
		{
			name:    "500 service error",
			userID:  id.String(),
			headers: authHeader(),
			body:    validReq,
			mockUS: func() ports.UserService {
				return &FakeUserService{
					UpdateUserFunc: func(ctx context.Context, du domain.User) (*domain.User, error) {
						return nil, errors.New("db error")
					},
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to update a user",
		},
		{
			name:    "404 not found (nil)",
			userID:  id.String(),
			headers: authHeader(),
			body:    validReq,
			mockUS: func() ports.UserService {
				return &FakeUserService{
					UpdateUserFunc: func(ctx context.Context, du domain.User) (*domain.User, error) {
						return nil, nil
					},
				}
			},
			wantStatus: http.StatusNotFound,
			wantErr:    "user not found",
		},
		{
			name:    "200 success",
			userID:  id.String(),
			headers: authHeader(),
			body:    validReq,
			mockUS: func() ports.UserService {
				u := someDomainUser()
				u.UUID = id
				return &FakeUserService{
					UpdateUserFunc: func(ctx context.Context, du domain.User) (*domain.User, error) {
						assert.Equal(t, id, du.UUID)
						return u, nil
					},
				}
			},
			wantStatus: http.StatusOK,
			wantErr:    "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r, _, _, _ := setupRouter(t, tt.mockUS(), true)
			rr := doReq(t, r, http.MethodPut, "/users/"+tt.userID, tt.body, tt.headers)
			require.Equal(t, tt.wantStatus, rr.Code)

			if tt.wantErr != "" {
				var resp map[string]any
				_ = json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.Equal(t, tt.wantErr, resp["error"])
			}
		})
	}
}

func TestUserController_DeleteUserHandler(t *testing.T) {
	id := uuid.New()

	authHeader := func() map[string]string {
		tok, _ := SignJWT("test-secret", "123", "admin", time.Hour)
		return map[string]string{"Authorization": "Bearer " + tok}
	}

	tests := []struct {
		name       string
		userID     string
		headers    map[string]string
		mockUS     func() ports.UserService
		wantStatus int
		wantErr    string
	}{
		{
			name:       "401 missing header",
			userID:     id.String(),
			headers:    nil,
			mockUS:     func() ports.UserService { return &FakeUserService{} },
			wantStatus: http.StatusUnauthorized,
			wantErr:    "missing Authorization header",
		},
		{
			name:       "400 invalid uuid",
			userID:     "not-uuid",
			headers:    authHeader(),
			mockUS:     func() ports.UserService { return &FakeUserService{} },
			wantStatus: http.StatusBadRequest,
			wantErr:    "user_id must be a valid UUID",
		},
		{
			name:    "500 service error",
			userID:  id.String(),
			headers: authHeader(),
			mockUS: func() ports.UserService {
				return &FakeUserService{
					DeleteUserFunc: func(ctx context.Context, userUUID domain.UUID) error {
						return errors.New("db error")
					},
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to delete user",
		},
		{
			name:    "204 success",
			userID:  id.String(),
			headers: authHeader(),
			mockUS: func() ports.UserService {
				return &FakeUserService{
					DeleteUserFunc: func(ctx context.Context, userUUID domain.UUID) error { return nil },
				}
			},
			wantStatus: http.StatusNoContent,
			wantErr:    "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r, _, _, _ := setupRouter(t, tt.mockUS(), true)
			rr := doReq(t, r, http.MethodDelete, "/users/"+tt.userID, nil, tt.headers)
			require.Equal(t, tt.wantStatus, rr.Code)

			if tt.wantErr != "" {
				var resp map[string]any
				_ = json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.Equal(t, tt.wantErr, resp["error"])
			}
		})
	}
}
