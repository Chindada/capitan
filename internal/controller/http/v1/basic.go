package v1

import (
	"net/http"

	"github.com/chindada/capitan/internal/controller/http/resp"
	"github.com/chindada/capitan/internal/usecases"
	"github.com/gin-gonic/gin"
)

type basicRoutes struct {
	t usecases.Basic
}

func NewBasicRoutes(handler *gin.RouterGroup, t usecases.Basic) {
	r := &basicRoutes{t}

	h := handler.Group("/basic")
	{
		h.GET("/stocks", r.getStocks)
	}
}

// getStocks -.
//
//	@Tags		Basic V1
//	@Summary	Get stocks
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Success	200	{object}	pb.StockDetailList
//	@Failure	400	{object}	pb.APIResponse
//	@Failure	500	{object}	pb.APIResponse
//	@Router		/api/capitan/v1/basic/stocks [get]
func (r *basicRoutes) getStocks(c *gin.Context) {
	stocks, err := r.t.GetAllStockDetail(c)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, stocks)
}
