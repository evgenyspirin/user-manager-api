package rest

import (
	"net/http"
	"user-manager-api/internal/infrastructure/jwt"
	"user-manager-api/internal/interface/api/rest/dto/user_file"
	"user-manager-api/internal/interface/api/rest/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"user-manager-api/internal/application/ports"
	"user-manager-api/internal/interface/api/rest/validator"
)

// 10MB
const maxSize = int64(10 << 20)

type UserFileController struct {
	userFileService ports.UserFileService
	logger          *zap.Logger
}

func NewUserFileController(
	r *gin.Engine,
	userFileService ports.UserFileService,
	logger *zap.Logger,
	jwtService *jwt.Service,
) *UserFileController {
	ufc := &UserFileController{
		userFileService: userFileService,
		logger:          logger,
	}

	r.GET(RouteUserFiles, ufc.GetUserFilesHandler)
	r.POST(RouteUserFiles, middleware.AuthMiddleware(jwtService), ufc.CreateUserFileHandler)
	r.DELETE(RouteUserFiles, middleware.AuthMiddleware(jwtService), ufc.DeleteUserFilesHandler)

	return ufc
}

func (ufc *UserFileController) GetUserFilesHandler(c *gin.Context) {
	page, err := validator.ValidatePage(c.Query("page"))
	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": err.Error()},
		)
		return
	}
	ok, uuid := validator.IsUUID(c.Param("user_id"))
	if !ok {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "user_id must be a valid UUID"},
		)
		return
	}

	files, err := ufc.userFileService.FindUserFiles(c.Request.Context(), uuid, page)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "failed to get files"},
		)
		ufc.logger.Error("FindUserFiles() error", zap.Error(err))
		return
	}

	c.JSON(http.StatusOK, user_file.ResponseData{
		Data: user_file.ToResponseUserFiles(files),
	})
}

func (ufc *UserFileController) CreateUserFileHandler(c *gin.Context) {
	ok, uuid := validator.IsUUID(c.Param("user_id"))
	if !ok {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "user_id must be a valid UUID"},
		)
		return
	}

	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	if fh.Size <= 0 || fh.Size > maxSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file too large or empty"})
		return
	}

	uf, err := ufc.userFileService.CreateUserFile(c.Request.Context(), uuid, fh)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "failed to create a file"},
		)
		ufc.logger.Error("CreateUserFile() error", zap.Error(err))
		return
	}

	c.JSON(http.StatusCreated, user_file.ToResponseUserFile(*uf))
}

func (ufc *UserFileController) DeleteUserFilesHandler(c *gin.Context) {
	ok, uuid := validator.IsUUID(c.Param("user_id"))
	if !ok {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "user_id must be a valid UUID"},
		)
		return
	}

	err := ufc.userFileService.DeleteUserFiles(c.Request.Context(), uuid)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "failed to delete user files"},
		)
		ufc.logger.Error("DeleteUserFiles() error", zap.Error(err))
		return
	}

	c.Status(http.StatusNoContent)
}
