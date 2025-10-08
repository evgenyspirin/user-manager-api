// user_file_controller_test.go
package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"user-manager-api/internal/application/ports"
	domainUser "user-manager-api/internal/domain/user"
	domainFile "user-manager-api/internal/domain/user_file"
	jwtSvc "user-manager-api/internal/infrastructure/jwt"
	"user-manager-api/internal/interface/api/rest/middleware"
)

type FakeUserFileService struct {
	FindUserFilesFunc   func(ctx context.Context, userUUID domainUser.UUID, page int) (domainFile.UserFiles, error)
	CreateUserFileFunc  func(ctx context.Context, userUUID domainUser.UUID, fh *multipart.FileHeader) (*domainFile.UserFile, error)
	DeleteUserFilesFunc func(ctx context.Context, userUUID domainUser.UUID) error
}

func (f *FakeUserFileService) FindUserFiles(ctx context.Context, userUUID domainUser.UUID, page int) (domainFile.UserFiles, error) {
	if f.FindUserFilesFunc == nil {
		return nil, errors.New("not used")
	}
	return f.FindUserFilesFunc(ctx, userUUID, page)
}
func (f *FakeUserFileService) CreateUserFile(ctx context.Context, userUUID domainUser.UUID, fh *multipart.FileHeader) (*domainFile.UserFile, error) {
	if f.CreateUserFileFunc == nil {
		return nil, errors.New("not used")
	}
	return f.CreateUserFileFunc(ctx, userUUID, fh)
}
func (f *FakeUserFileService) DeleteUserFiles(ctx context.Context, userUUID domainUser.UUID) error {
	if f.DeleteUserFilesFunc == nil {
		return errors.New("not used")
	}
	return f.DeleteUserFilesFunc(ctx, userUUID)
}

func setupRouterUFC(t *testing.T, ufs ports.UserFileService, withJWT bool) (*gin.Engine, *UserFileController, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	logger := zap.NewNop()
	secret := "test-secret"
	j := jwtSvc.New(secret)

	ufc := &UserFileController{
		userFileService: ufs,
		logger:          logger,
	}

	r.GET("/users/:user_id/files", ufc.GetUserFilesHandler)
	if withJWT {
		r.POST("/users/:user_id/files", middleware.AuthMiddleware(j), ufc.CreateUserFileHandler)
		r.DELETE("/users/:user_id/files", middleware.AuthMiddleware(j), ufc.DeleteUserFilesHandler)
	} else {
		r.POST("/users/:user_id/files", ufc.CreateUserFileHandler)
		r.DELETE("/users/:user_id/files", ufc.DeleteUserFilesHandler)
	}

	return r, ufc, secret
}

func doFileReq(t *testing.T, r *gin.Engine, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	switch v := body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case string:
		reader = bytes.NewReader([]byte(v))
	case []byte:
		reader = bytes.NewReader(v)
	default:
		b, err := json.Marshal(v)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, path, reader)
	require.NoError(t, err)
	if body != nil {
		if _, isStr := body.(string); !isStr {
			if _, isBytes := body.([]byte); !isBytes {
				req.Header.Set("Content-Type", "application/json")
			}
		}
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func doMultipartReq(t *testing.T, r *gin.Engine, method, path string, fields map[string]string, fileField, fileName string, fileContent []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	for k, v := range fields {
		require.NoError(t, w.WriteField(k, v))
	}

	if fileField != "" && fileName != "" && fileContent != nil {
		fw, err := w.CreateFormFile(fileField, fileName)
		require.NoError(t, err)
		_, _ = fw.Write(fileContent)
	}

	require.NoError(t, w.Close())

	req, err := http.NewRequest(method, path, &b)
	require.NoError(t, err)
	req.Header.Set("Content-Type", w.FormDataContentType())
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestUserFileController_GetUserFilesHandler(t *testing.T) {
	okID := uuid.New()

	tests := []struct {
		name       string
		userID     string
		page       string
		mockUFS    func() ports.UserFileService
		wantStatus int
		wantErr    string
	}{
		{
			name:   "400 invalid uuid",
			userID: "not-uuid",
			page:   "1",
			mockUFS: func() ports.UserFileService {
				return &FakeUserFileService{}
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    "user_id must be a valid UUID",
		},
		{
			name:   "500 service error",
			userID: okID.String(),
			page:   "2",
			mockUFS: func() ports.UserFileService {
				return &FakeUserFileService{
					FindUserFilesFunc: func(ctx context.Context, userUUID domainUser.UUID, page int) (domainFile.UserFiles, error) {
						return nil, errors.New("db error")
					},
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to get files",
		},
		{
			name:   "200 success (empty list ok)",
			userID: okID.String(),
			page:   "3",
			mockUFS: func() ports.UserFileService {
				return &FakeUserFileService{
					FindUserFilesFunc: func(ctx context.Context, userUUID domainUser.UUID, page int) (domainFile.UserFiles, error) {
						var files domainFile.UserFiles
						return files, nil
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
			r, _, _ := setupRouterUFC(t, tt.mockUFS(), false)
			rr := doFileReq(t, r, http.MethodGet, "/users/"+tt.userID+"/files?page="+tt.page, nil, nil)
			require.Equal(t, tt.wantStatus, rr.Code)

			if tt.wantErr != "" {
				var resp map[string]any
				_ = json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.Equal(t, tt.wantErr, resp["error"])
			}
		})
	}
}

func TestUserFileController_CreateUserFileHandler(t *testing.T) {
	okID := uuid.New()

	withAuth := func(secret string) map[string]string {
		tok, _ := SignJWT(secret, "u1", "admin", time.Hour)
		return map[string]string{"Authorization": "Bearer " + tok}
	}

	tests := []struct {
		name       string
		userID     string
		headers    map[string]string
		fileField  string
		fileName   string
		fileBytes  []byte
		mockUFS    func() ports.UserFileService
		wantStatus int
		wantErr    string
	}{
		{
			name:       "401 missing Authorization",
			userID:     okID.String(),
			headers:    nil,
			fileField:  "file",
			fileName:   "doc.pdf",
			fileBytes:  []byte("pdf-bytes"),
			mockUFS:    func() ports.UserFileService { return &FakeUserFileService{} },
			wantStatus: http.StatusUnauthorized,
			wantErr:    "missing Authorization header",
		},
		{
			name:       "401 invalid format",
			userID:     okID.String(),
			headers:    map[string]string{"Authorization": "Token abc"},
			fileField:  "file",
			fileName:   "doc.pdf",
			fileBytes:  []byte("pdf-bytes"),
			mockUFS:    func() ports.UserFileService { return &FakeUserFileService{} },
			wantStatus: http.StatusUnauthorized,
			wantErr:    "invalid token format",
		},
		{
			name:       "401 bad signature",
			userID:     okID.String(),
			headers:    withAuth("other-secret"),
			fileField:  "file",
			fileName:   "doc.pdf",
			fileBytes:  []byte("pdf-bytes"),
			mockUFS:    func() ports.UserFileService { return &FakeUserFileService{} },
			wantStatus: http.StatusUnauthorized,
			wantErr:    "invalid token",
		},
		{
			name:       "400 invalid uuid",
			userID:     "not-uuid",
			headers:    withAuth("test-secret"),
			fileField:  "file",
			fileName:   "doc.pdf",
			fileBytes:  []byte("bytes"),
			mockUFS:    func() ports.UserFileService { return &FakeUserFileService{} },
			wantStatus: http.StatusBadRequest,
			wantErr:    "user_id must be a valid UUID",
		},
		{
			name:       "400 file is required",
			userID:     okID.String(),
			headers:    withAuth("test-secret"),
			fileField:  "", // no file part
			fileName:   "",
			fileBytes:  nil,
			mockUFS:    func() ports.UserFileService { return &FakeUserFileService{} },
			wantStatus: http.StatusBadRequest,
			wantErr:    "file is required",
		},
		{
			name:       "413 empty file",
			userID:     okID.String(),
			headers:    withAuth("test-secret"),
			fileField:  "file",
			fileName:   "empty.txt",
			fileBytes:  []byte{}, // size == 0 triggers "too large or empty"
			mockUFS:    func() ports.UserFileService { return &FakeUserFileService{} },
			wantStatus: http.StatusRequestEntityTooLarge,
			wantErr:    "file too large or empty",
		},
		{
			name:      "500 service error",
			userID:    okID.String(),
			headers:   withAuth("test-secret"),
			fileField: "file",
			fileName:  "doc.pdf",
			fileBytes: []byte("content"),
			mockUFS: func() ports.UserFileService {
				return &FakeUserFileService{
					CreateUserFileFunc: func(ctx context.Context, userUUID domainUser.UUID, fh *multipart.FileHeader) (*domainFile.UserFile, error) {
						return nil, errors.New("db error")
					},
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to create a file",
		},
		{
			name:      "201 success",
			userID:    okID.String(),
			headers:   withAuth("test-secret"),
			fileField: "file",
			fileName:  "doc.pdf",
			fileBytes: []byte("%PDF..."),
			mockUFS: func() ports.UserFileService {
				return &FakeUserFileService{
					CreateUserFileFunc: func(ctx context.Context, userUUID domainUser.UUID, fh *multipart.FileHeader) (*domainFile.UserFile, error) {
						return &domainFile.UserFile{}, nil
					},
				}
			},
			wantStatus: http.StatusCreated,
			wantErr:    "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r, _, secret := setupRouterUFC(t, tt.mockUFS(), true)
			_ = secret // secret is "test-secret" used in withAuth

			rr := doMultipartReq(t, r, http.MethodPost, "/users/"+tt.userID+"/files",
				nil, tt.fileField, tt.fileName, tt.fileBytes, tt.headers)

			require.Equal(t, tt.wantStatus, rr.Code)

			if tt.wantErr != "" {
				var resp map[string]any
				_ = json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.Equal(t, tt.wantErr, resp["error"])
			}
		})
	}
}

func TestUserFileController_DeleteUserFilesHandler(t *testing.T) {
	okID := uuid.New()

	authHeader := func() map[string]string {
		tok, _ := SignJWT("test-secret", "u1", "admin", time.Hour)
		return map[string]string{"Authorization": "Bearer " + tok}
	}

	tests := []struct {
		name       string
		userID     string
		headers    map[string]string
		mockUFS    func() ports.UserFileService
		wantStatus int
		wantErr    string
	}{
		{
			name:       "401 missing Authorization",
			userID:     okID.String(),
			headers:    nil,
			mockUFS:    func() ports.UserFileService { return &FakeUserFileService{} },
			wantStatus: http.StatusUnauthorized,
			wantErr:    "missing Authorization header",
		},
		{
			name:       "401 invalid format",
			userID:     okID.String(),
			headers:    map[string]string{"Authorization": "Token X"},
			mockUFS:    func() ports.UserFileService { return &FakeUserFileService{} },
			wantStatus: http.StatusUnauthorized,
			wantErr:    "invalid token format",
		},
		{
			name:   "401 bad signature",
			userID: okID.String(),
			headers: func() map[string]string {
				tok, _ := SignJWT("other-secret", "u1", "admin", time.Hour)
				return map[string]string{"Authorization": "Bearer " + tok}
			}(),
			mockUFS:    func() ports.UserFileService { return &FakeUserFileService{} },
			wantStatus: http.StatusUnauthorized,
			wantErr:    "invalid token",
		},
		{
			name:       "400 invalid uuid",
			userID:     "not-uuid",
			headers:    authHeader(),
			mockUFS:    func() ports.UserFileService { return &FakeUserFileService{} },
			wantStatus: http.StatusBadRequest,
			wantErr:    "user_id must be a valid UUID",
		},
		{
			name:    "500 service error",
			userID:  okID.String(),
			headers: authHeader(),
			mockUFS: func() ports.UserFileService {
				return &FakeUserFileService{
					DeleteUserFilesFunc: func(ctx context.Context, userUUID domainUser.UUID) error {
						return errors.New("db error")
					},
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to delete user files",
		},
		{
			name:    "204 success",
			userID:  okID.String(),
			headers: authHeader(),
			mockUFS: func() ports.UserFileService {
				return &FakeUserFileService{
					DeleteUserFilesFunc: func(ctx context.Context, userUUID domainUser.UUID) error { return nil },
				}
			},
			wantStatus: http.StatusNoContent,
			wantErr:    "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r, _, _ := setupRouterUFC(t, tt.mockUFS(), true)
			rr := doFileReq(t, r, http.MethodDelete, "/users/"+tt.userID+"/files", nil, tt.headers)
			require.Equal(t, tt.wantStatus, rr.Code)

			if tt.wantErr != "" {
				var resp map[string]any
				_ = json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.Equal(t, tt.wantErr, resp["error"])
			}
		})
	}
}
