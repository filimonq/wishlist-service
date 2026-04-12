package httpadapter

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/filimonq/wishlist-service/internal/domain"
)

// @Summary  Регистрация
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body     RegisterRequest true "Данные для регистрации"
// @Success  201  {object} UserResponse
// @Failure  400  {object} map[string]string
// @Failure  409  {object} map[string]string
// @Router   /auth/register [post]
func (s *Server) register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := s.auth.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		s.domainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, UserResponse{ID: user.ID, Email: user.Email})
}

// @Summary  Вход
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body     LoginRequest true "Данные для входа"
// @Success  200  {object} AuthResponse
// @Failure  400  {object} map[string]string
// @Failure  401  {object} map[string]string
// @Router   /auth/login [post]
func (s *Server) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := s.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		s.domainError(c, err)
		return
	}

	c.JSON(http.StatusOK, AuthResponse{Token: token})
}

func (s *Server) domainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrUserAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrWishlistNotFound),
		errors.Is(err, domain.ErrItemNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrWishlistAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrItemAlreadyReserved):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrItemNotInWishlist):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
