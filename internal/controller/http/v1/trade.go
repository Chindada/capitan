package v1

import (
	"github.com/chindada/capitan/internal/usecases"
	"github.com/gin-gonic/gin"
)

type tradeRoutes struct {
	t usecases.Trade
}

func NewTradeRoutes(ws *gin.RouterGroup, t usecases.Trade) {
	r := &tradeRoutes{t}
	w := ws.Group("/trade")
	{
		w.GET("/future", r.streamTrade)
	}
}

func (r *tradeRoutes) streamTrade(*gin.Context) {
}
