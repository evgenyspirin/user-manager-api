package rest

import (
	"errors"
	"net/http"
	"user-manager-api/internal/infrastructure/jwt"
	"user-manager-api/internal/interface/api/rest/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"user-manager-api/internal/application/ports"
	userDB "user-manager-api/internal/infrastructure/db/postgres/user"
	"user-manager-api/internal/interface/api/rest/dto/user"
	"user-manager-api/internal/interface/api/rest/validator"
)

type UserController struct {
	userService ports.UserService
	logger      *zap.Logger
}

func NewUserController(
	r *gin.Engine,
	userService ports.UserService,
	logger *zap.Logger,
	jwtService *jwt.Service,
) *UserController {
	uc := &UserController{
		userService: userService,
		logger:      logger,
	}

	r.GET(RouteUsers, uc.GetUsersHandler)
	r.GET(RouteUser, uc.GetUserHandler)
	r.POST(RouteUsers, middleware.AuthMiddleware(jwtService), uc.CreateUserHandler)
	r.PUT(RouteUser, middleware.AuthMiddleware(jwtService), uc.UpdateUserHandler)
	r.DELETE(RouteUser, middleware.AuthMiddleware(jwtService), uc.DeleteUserHandler)

	return uc
}

func (uc *UserController) GetUsersHandler(c *gin.Context) {
	page, err := validator.ValidatePage(c.Query("page"))
	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": err.Error()},
		)
		return
	}

	users, err := uc.userService.FindUsers(c.Request.Context(), page)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "failed to get users"},
		)
		uc.logger.Error("FindUsers() error", zap.Error(err))
		return
	}

	c.JSON(http.StatusOK, user.ResponseData{
		Data: user.ToResponseUsers(users),
	})
}

func (uc *UserController) GetUserHandler(c *gin.Context) {
	ok, uuid := validator.IsUUID(c.Param("user_id"))
	if !ok {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "user_id must be a valid UUID"},
		)
		return
	}

	u, err := uc.userService.FindUserByID(c.Request.Context(), uuid)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "failed to get a user"},
		)
		uc.logger.Error("FindUserByID() error", zap.Error(err))
		return
	}

	if u == nil {
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "user not found"},
		)
		return
	}

	c.JSON(http.StatusOK, user.ToResponseUser(*u))
}

func (uc *UserController) CreateUserHandler(c *gin.Context) {
	var req user.Request
	// for a good boost of performance(x3 minimum) and to avoid reflection under the hood
	// better to use codegen for marshal/unmarshal for example:
	// https://github.com/mailru/easyjson
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}
	if errs := validator.ValidateUser(req); errs != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": errs,
		})
		return
	}

	uDomain, err := user.ToDomainUser(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
	}

	u, err := uc.userService.CreateUser(c.Request.Context(), uDomain)
	if err != nil {
		if errors.Is(err, userDB.ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "failed to create a user"},
		)
		uc.logger.Error("CreateUser() error", zap.Error(err))
		return
	}

	c.JSON(http.StatusCreated, user.ToResponseUser(*u))
}

func (uc *UserController) UpdateUserHandler(c *gin.Context) {
	ok, uuid := validator.IsUUID(c.Param("user_id"))
	if !ok {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "user_id must be a valid UUID"},
		)
		return
	}

	var req user.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}
	if errs := validator.ValidateUser(req); errs != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": errs,
		})
		return
	}

	uDomain, err := user.ToDomainUser(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
	}
	uDomain.UUID = uuid

	u, err := uc.userService.UpdateUser(c.Request.Context(), uDomain)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "failed to update a user"},
		)
		uc.logger.Error("UpdateUser() error", zap.Error(err))
		return
	}

	if u == nil {
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "user not found"},
		)
		return
	}

	c.JSON(http.StatusOK, user.ToResponseUser(*u))
}

func (uc *UserController) DeleteUserHandler(c *gin.Context) {
	ok, uuid := validator.IsUUID(c.Param("user_id"))
	if !ok {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "user_id must be a valid UUID"},
		)
		return
	}

	err := uc.userService.DeleteUser(c.Request.Context(), uuid)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "failed to delete user"},
		)
		uc.logger.Error("DeleteUser() error", zap.Error(err))
		return
	}

	c.Status(http.StatusNoContent)
}
