package httpadapter

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Server) createItem(c *gin.Context) {
	wishlistID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wishlist id"})
		return
	}

	var req CreateItemRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := s.items.Create(c.Request.Context(), getUserID(c), wishlistID, req.Name, req.Description, req.URL, req.Priority)
	if err != nil {
		s.domainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toItemResponse(item))
}

func (s *Server) listItems(c *gin.Context) {
	wishlistID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wishlist id"})
		return
	}

	items, err := s.items.List(c.Request.Context(), getUserID(c), wishlistID)
	if err != nil {
		s.domainError(c, err)
		return
	}

	c.JSON(http.StatusOK, toItemResponses(items))
}

func (s *Server) updateItem(c *gin.Context) {
	wishlistID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wishlist id"})
		return
	}

	itemID, err := uuid.Parse(c.Param("itemID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	var req UpdateItemRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := s.items.Update(c.Request.Context(), getUserID(c), wishlistID, itemID, req.Name, req.Description, req.URL, req.Priority)
	if err != nil {
		s.domainError(c, err)
		return
	}

	c.JSON(http.StatusOK, toItemResponse(item))
}

func (s *Server) deleteItem(c *gin.Context) {
	wishlistID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wishlist id"})
		return
	}

	itemID, err := uuid.Parse(c.Param("itemID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	if err = s.items.Delete(c.Request.Context(), getUserID(c), wishlistID, itemID); err != nil {
		s.domainError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) getPublicWishlist(c *gin.Context) {
	token, err := uuid.Parse(c.Param("token"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token"})
		return
	}

	wishlist, err := s.wishlists.GetByPublicToken(c.Request.Context(), token)
	if err != nil {
		s.domainError(c, err)
		return
	}

	items, err := s.items.ListByWishlistID(c.Request.Context(), wishlist.ID)
	if err != nil {
		s.domainError(c, err)
		return
	}

	resp := toWishlistResponse(wishlist)
	resp.Items = toItemResponses(items)

	c.JSON(http.StatusOK, resp)
}

func (s *Server) reserveItem(c *gin.Context) {
	token, err := uuid.Parse(c.Param("token"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token"})
		return
	}

	itemID, err := uuid.Parse(c.Param("itemID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	if err = s.items.Reserve(c.Request.Context(), token, itemID); err != nil {
		s.domainError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
