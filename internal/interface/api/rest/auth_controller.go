package rest

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"user-manager-api/internal/application/ports"
	userDB "user-manager-api/internal/infrastructure/db/postgres/user"
	"user-manager-api/internal/interface/api/rest/dto/auth"
	"user-manager-api/internal/interface/api/rest/validator"
)

type AuthController struct {
	logger      *zap.Logger
	userService ports.UserService
	authService ports.Auth
}

func NewAuthController(
	r *gin.Engine,
	logger *zap.Logger,
	userService ports.UserService,
	authService ports.Auth,
) *AuthController {
	ac := &AuthController{
		logger:      logger,
		userService: userService,
		authService: authService,
	}

	r.POST(RouteLogin, ac.LoginHandler)

	return ac
}

func (ac *AuthController) LoginHandler(c *gin.Context) {
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "invalid json"},
		)
		return
	}

	if errs := validator.ValidateLogin(req); errs != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": errs,
		})
		return
	}

	u, err := ac.userService.FindByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "failed to get a user"},
		)
		ac.logger.Error("FindUserByID() error", zap.Error(err))
		return
	}
	if u == nil {
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "user not found"},
		)
		return
	}

	token, err := ac.authService.GenerateToken(u, req.Password)
	if err != nil {
		if errors.Is(err, userDB.ErrEmailAlreadyExists) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		}
		if errors.Is(err, userDB.ErrEmailAlreadyExists) {
			ac.logger.Error("GenerateToken() error", zap.Error(err), zap.Stringer("user_uuid", u.UUID))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": token,
		"token_type":   "Bearer",
	})
}
