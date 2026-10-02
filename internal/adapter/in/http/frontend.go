package httpadapter

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/filimonq/wishlist-service/frontend"
)

func registerFrontend(r *gin.Engine) {
	files := http.FileServer(http.FS(frontend.Files))
	r.GET("/", gin.WrapH(files))
	r.GET("/assets/*filepath", gin.WrapH(http.StripPrefix("/assets/", files)))
}
