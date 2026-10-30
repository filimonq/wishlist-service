package httpadapter

import (
	"context"
	"slices"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/filimonq/wishlist-service/internal/service"
)

type Server struct {
	auth                 service.AuthService
	wishlists            service.WishlistService
	items                service.ItemService
	jwtSecret            string
	pingDB               func(context.Context) error
	publicReadMiddleware []gin.HandlerFunc
}

func NewServer(
	auth service.AuthService,
	wishlists service.WishlistService,
	items service.ItemService,
	jwtSecret string,
	pingDB func(context.Context) error,
	publicReadMiddleware ...gin.HandlerFunc,
) *Server {
	return &Server{
		auth:                 auth,
		wishlists:            wishlists,
		items:                items,
		jwtSecret:            jwtSecret,
		pingDB:               pingDB,
		publicReadMiddleware: publicReadMiddleware,
	}
}

func (s *Server) Router(middleware ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(requestLogging())
	r.Use(middleware...)
	r.Use(gin.CustomRecoveryWithWriter(nil, recoverPanic))

	r.GET("/livez", s.livez)
	r.GET("/readyz", s.readyz)
	registerFrontend(r)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", s.register)
			auth.POST("/login", s.login)
		}

		public := v1.Group("/public/:token")
		{
			public.GET("", append(slices.Clone(s.publicReadMiddleware), s.getPublicWishlist)...)
			public.POST("/items/:itemID/reserve", s.reserveItem)
		}

		authorized := v1.Group("")
		authorized.Use(s.authMiddleware())
		{
			wishlists := authorized.Group("/wishlists")
			{
				wishlists.GET("", s.listWishlists)
				wishlists.POST("", s.createWishlist)
				wishlists.GET("/:id", s.getWishlist)
				wishlists.PUT("/:id", s.updateWishlist)
				wishlists.DELETE("/:id", s.deleteWishlist)

				items := wishlists.Group("/:id/items")
				{
					items.GET("", s.listItems)
					items.POST("", s.createItem)
					items.PUT("/:itemID", s.updateItem)
					items.DELETE("/:itemID", s.deleteItem)
				}
			}
		}
	}

	return r
}
