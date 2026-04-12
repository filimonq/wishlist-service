package httpadapter

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// @Summary   Создать вишлист
// @Tags      wishlists
// @Security  BearerAuth
// @Accept    json
// @Produce   json
// @Param     body body     CreateWishlistRequest true "Данные вишлиста"
// @Success   201  {object} WishlistResponse
// @Failure   400  {object} map[string]string
// @Failure   401  {object} map[string]string
// @Router    /wishlists [post]
func (s *Server) createWishlist(c *gin.Context) {
	var req CreateWishlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wishlist, err := s.wishlists.Create(c.Request.Context(), getUserID(c), req.Title, req.Description, req.EventDate)
	if err != nil {
		s.domainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toWishlistResponse(wishlist))
}

// @Summary   Список вишлистов
// @Tags      wishlists
// @Security  BearerAuth
// @Produce   json
// @Success   200  {array}  WishlistResponse
// @Failure   401  {object} map[string]string
// @Router    /wishlists [get]
func (s *Server) listWishlists(c *gin.Context) {
	wishlists, err := s.wishlists.List(c.Request.Context(), getUserID(c))
	if err != nil {
		s.domainError(c, err)
		return
	}

	result := make([]WishlistResponse, len(wishlists))
	for i, wl := range wishlists {
		result[i] = toWishlistResponse(wl)
	}

	c.JSON(http.StatusOK, result)
}

// @Summary   Получить вишлист
// @Tags      wishlists
// @Security  BearerAuth
// @Produce   json
// @Param     id  path     string true "ID вишлиста"
// @Success   200 {object} WishlistResponse
// @Failure   400 {object} map[string]string
// @Failure   401 {object} map[string]string
// @Failure   403 {object} map[string]string
// @Failure   404 {object} map[string]string
// @Router    /wishlists/{id} [get]
func (s *Server) getWishlist(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wishlist id"})
		return
	}

	wishlist, err := s.wishlists.Get(c.Request.Context(), getUserID(c), id)
	if err != nil {
		s.domainError(c, err)
		return
	}

	c.JSON(http.StatusOK, toWishlistResponse(wishlist))
}

// @Summary   Обновить вишлист
// @Tags      wishlists
// @Security  BearerAuth
// @Accept    json
// @Produce   json
// @Param     id   path     string               true "ID вишлиста"
// @Param     body body     UpdateWishlistRequest true "Данные для обновления"
// @Success   200  {object} WishlistResponse
// @Failure   400  {object} map[string]string
// @Failure   401  {object} map[string]string
// @Failure   403  {object} map[string]string
// @Failure   404  {object} map[string]string
// @Router    /wishlists/{id} [put]
func (s *Server) updateWishlist(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wishlist id"})
		return
	}

	var req UpdateWishlistRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wishlist, err := s.wishlists.Update(c.Request.Context(), getUserID(c), id, req.Title, req.Description, req.EventDate)
	if err != nil {
		s.domainError(c, err)
		return
	}

	c.JSON(http.StatusOK, toWishlistResponse(wishlist))
}

// @Summary   Удалить вишлист
// @Tags      wishlists
// @Security  BearerAuth
// @Param     id  path string true "ID вишлиста"
// @Success   204
// @Failure   400 {object} map[string]string
// @Failure   401 {object} map[string]string
// @Failure   403 {object} map[string]string
// @Failure   404 {object} map[string]string
// @Router    /wishlists/{id} [delete]
func (s *Server) deleteWishlist(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wishlist id"})
		return
	}

	if err = s.wishlists.Delete(c.Request.Context(), getUserID(c), id); err != nil {
		s.domainError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
