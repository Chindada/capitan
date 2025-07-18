package v1

import (
	"net/http"

	"github.com/chindada/capitan/internal/controller/http/resp"
	"github.com/chindada/capitan/internal/usecases"
	"github.com/chindada/panther/golang/pb"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type basicRoutes struct {
	t usecases.Basic
}

func NewBasicRoutes(handler *gin.RouterGroup, t usecases.Basic) {
	r := &basicRoutes{t}

	h := handler.Group("/basic")
	{
		h.GET("/stocks", r.getStocks)
		h.GET("/options", r.getOptions)

		h.GET("/futures", r.getFutures)
		h.PUT("/futures", r.updateFutures)

		h.GET("/contract/future", r.getAllFutureContract)
		h.POST("/contract/future", r.createFutureContract)
		h.PUT("/contract/future", r.updateFutureContract)
		h.DELETE("/contract/future", r.deleteFutureContract)

		h.POST("/future/kbar", r.getFutureKbar)
		h.POST("/future/kbar/last", r.getFutureLastKbar)

		h.GET("/target/stock", r.getTargetStock)
		h.GET("/target/future", r.getTargetFuture)
	}
}

// getStocks -.
//
//	@Tags		Basic V1
//	@Summary	Get stocks
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Param		code	query		string	false	"code"
//	@Success	200		{object}	pb.StockDetailList
//	@Failure	500		{object}	pb.APIResponse
//	@Router		/api/capitan/v1/basic/stocks [get]
func (r *basicRoutes) getStocks(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		stocks, err := r.t.GetAllStockDetail(c)
		if err != nil {
			resp.Fail(c, http.StatusInternalServerError, err)
			return
		}
		resp.Success(c, http.StatusOK, &pb.StockDetailList{
			List: stocks,
		})
	} else {
		stock, err := r.t.GetStockDetailByCode(c, code)
		if err != nil {
			resp.Fail(c, http.StatusInternalServerError, err)
			return
		}
		resp.Success(c, http.StatusOK, &pb.StockDetailList{
			List: []*pb.StockDetail{stock},
		})
	}
}

// getFutures -.
//
//	@Tags		Basic V1
//	@Summary	Get futures
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Param		code	query		string	false	"code"
//	@Success	200		{object}	pb.FutureDetailList
//	@Failure	500		{object}	pb.APIResponse
//	@Router		/api/capitan/v1/basic/futures [get]
func (r *basicRoutes) getFutures(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		data, err := r.t.GetAllFutureDetail(c)
		if err != nil {
			resp.Fail(c, http.StatusInternalServerError, err)
			return
		}
		resp.Success(c, http.StatusOK, &pb.FutureDetailList{
			List: data,
		})
	} else {
		future, err := r.t.GetFutureDetailByCode(c, code)
		if err != nil {
			resp.Fail(c, http.StatusInternalServerError, err)
			return
		}
		resp.Success(c, http.StatusOK, &pb.FutureDetailList{
			List: []*pb.FutureDetail{future},
		})
	}
}

// updateFutures -.
//
//	@Tags		Basic V1
//	@Summary	Update futures
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@param		body	body		pb.UpdateFutureDetailRequest	true	"Body"
//	@Success	200		{object}	emptypb.Empty
//	@Failure	400		{object}	pb.APIResponse
//	@Failure	500		{object}	pb.APIResponse
//	@Router		/api/capitan/v1/basic/futures [put]
func (r *basicRoutes) updateFutures(c *gin.Context) {
	req := &pb.UpdateFutureDetailRequest{}
	err := c.Bind(req)
	if err != nil {
		resp.Fail(c, http.StatusBadRequest, err)
		return
	}
	err = r.t.SetFutureDetailContract(c, req)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &emptypb.Empty{})
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
//	@Summary	Get future kbar
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

// getFutureLastKbar -.
//
//	@Tags		Basic V1
//	@Summary	Get last futures kbar
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Success	200	{object}	pb.HistoryKbarList
//	@Failure	400	{object}	pb.APIResponse
//	@Failure	500	{object}	pb.APIResponse
//	@Router		/api/capitan/v1/basic/future/kbar/last [post]
func (r *basicRoutes) getFutureLastKbar(c *gin.Context) {
	req := &pb.HistoryKbarRequest{}
	err := c.Bind(req)
	if err != nil {
		resp.Fail(c, http.StatusBadRequest, err)
		return
	}

	data, err := r.t.GetLastFutureKbar(c, req)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, data)
}

// getTargetStock -.
//
//	@Tags		Basic V1
//	@Summary	Get target stocks
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Success	200	{object}	pb.StockDetailList
//	@Router		/api/capitan/v1/basic/target/stock [get]
func (r *basicRoutes) getTargetStock(c *gin.Context) {
	targets := r.t.GetTargetStock()
	resp.Success(c, http.StatusOK, &pb.StockDetailList{
		List: targets,
	})
}

// getTargetFuture -.
//
//	@Tags		Basic V1
//	@Summary	Get target futures
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Success	200	{object}	pb.FutureDetailList
//	@Router		/api/capitan/v1/basic/target/future [get]
func (r *basicRoutes) getTargetFuture(c *gin.Context) {
	targets := r.t.GetTargetFuture()
	resp.Success(c, http.StatusOK, &pb.FutureDetailList{
		List: targets,
	})
}

// getAllFutureContract -.
//
//	@Tags		Basic V1
//	@Summary	Get all future contracts
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Success	200	{object}	pb.FutureContractList
//	@Failure	500	{object}	pb.APIResponse
//	@Router		/api/capitan/v1/basic/contract/future [get]
func (r *basicRoutes) getAllFutureContract(c *gin.Context) {
	contracts, err := r.t.GetAllFutureContract(c)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &pb.FutureContractList{
		List: contracts,
	})
}

// createFutureContract -.
//
//	@Tags		Basic V1
//	@Summary	Create future contract
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@param		body	body		pb.FutureContract	true	"Body"
//	@Success	200		{object}	emptypb.Empty
//	@Failure	400		{object}	pb.APIResponse
//	@Failure	500		{object}	pb.APIResponse
//	@Router		/api/capitan/v1/basic/contract/future [post]
func (r *basicRoutes) createFutureContract(c *gin.Context) {
	req := &pb.FutureContract{}
	err := c.Bind(req)
	if err != nil {
		resp.Fail(c, http.StatusBadRequest, err)
		return
	}
	err = r.t.CreateFutureContract(c, req)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &emptypb.Empty{})
}

// updateFutureContract -.
//
//	@Tags		Basic V1
//	@Summary	Update future contract
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@param		body	body		pb.FutureContract	true	"Body"
//	@Success	200		{object}	emptypb.Empty
//	@Failure	400		{object}	pb.APIResponse
//	@Failure	500		{object}	pb.APIResponse
//	@Router		/api/capitan/v1/basic/contract/future [put]
func (r *basicRoutes) updateFutureContract(c *gin.Context) {
	req := &pb.FutureContract{}
	err := c.Bind(req)
	if err != nil {
		resp.Fail(c, http.StatusBadRequest, err)
		return
	}
	err = r.t.UpdateFutureContract(c, req)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &emptypb.Empty{})
}

// deleteFutureContract -.
//
//	@Tags		Basic V1
//	@Summary	Delete future contract
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@param		body	body		structpb.ListValue	true	"id(int) of contract"
//	@Success	200		{object}	emptypb.Empty
//	@Failure	400		{object}	pb.APIResponse
//	@Failure	500		{object}	pb.APIResponse
//	@Router		/api/capitan/v1/basic/contract/future [delete]
func (r *basicRoutes) deleteFutureContract(c *gin.Context) {
	indexPB := &structpb.ListValue{}
	err := c.Bind(indexPB)
	if err != nil {
		resp.Fail(c, http.StatusBadRequest, err)
		return
	}

	var idList []int64
	for _, id := range indexPB.GetValues() {
		v := id.GetNumberValue()
		if v == 0 {
			resp.Fail(c, http.StatusBadRequest, resp.ErrTypeWrong)
			return
		}
		idList = append(idList, int64(v))
	}
	err = r.t.DeleteFutureContract(c, idList)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &emptypb.Empty{})
}
