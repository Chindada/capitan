package v1

import (
	"net/http"

	"github.com/chindada/capitan/internal/controller/http/resp"
	"github.com/chindada/capitan/internal/usecases"
	"github.com/chindada/panther/golang/pb"
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
		h.GET("/futures", r.getFutures)
		h.GET("/options", r.getOptions)

		h.POST("/future/kbar", r.getFutureKbar)
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
//	@Failure	500	{object}	pb.APIResponse
//	@Router		/api/capitan/v1/basic/stocks [get]
func (r *basicRoutes) getStocks(c *gin.Context) {
	stocks, err := r.t.GetAllStockDetail(c)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &pb.StockDetailList{
		List: stocks,
	})
}

// getFutures -.
//
//	@Tags		Basic V1
//	@Summary	Get futures
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Success	200	{object}	pb.FutureDetailList
//	@Failure	500	{object}	pb.APIResponse
//	@Router		/api/capitan/v1/basic/futures [get]
func (r *basicRoutes) getFutures(c *gin.Context) {
	data, err := r.t.GetAllFutureDetail(c)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &pb.FutureDetailList{
		List: data,
	})
}

// getOptions -.
//
//	@Tags		Basic V1
//	@Summary	Get options
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Success	200	{object}	pb.OptionDetailList
//	@Failure	500	{object}	pb.APIResponse
//	@Router		/api/capitan/v1/basic/options [get]
func (r *basicRoutes) getOptions(c *gin.Context) {
	data, err := r.t.GetAllOptionDetail(c)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &pb.OptionDetailList{
		List: data,
	})
}

// getFutureKbar -.
//
//	@Tags		Basic V1
//	@Summary	Get futures
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Success	200	{object}	pb.HistoryKbarList
//	@Failure	400	{object}	pb.APIResponse
//	@Failure	500	{object}	pb.APIResponse
//	@Router		/api/capitan/v1/basic/future/kbar [post]
func (r *basicRoutes) getFutureKbar(c *gin.Context) {
	req := &pb.HistoryKbarRequest{}
	err := c.Bind(req)
	if err != nil {
		resp.Fail(c, http.StatusBadRequest, err)
		return
	}

	data, err := r.t.GetFutureKbar(c, req)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, data)
}
