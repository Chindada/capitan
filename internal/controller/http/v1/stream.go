package v1

import (
	"github.com/chindada/capitan/internal/usecases"
	"github.com/gin-gonic/gin"
)

type streamRoutes struct {
	t usecases.Stream
}

func NewStreamRoutes(ws *gin.RouterGroup, t usecases.Stream) {
	r := &streamRoutes{t}
	w := ws.Group("/stream")
	{
		w.GET("/futures", r.streamFutrues)
	}
}

func (r *streamRoutes) streamFutrues(*gin.Context) {
}
