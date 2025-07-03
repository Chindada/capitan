package v1

import (
	"errors"
	"net/http"
	"sync"

	"github.com/chindada/capitan/internal/controller/http/resp"
	"github.com/chindada/capitan/internal/controller/http/ws"
	"github.com/chindada/capitan/internal/usecases"
	"github.com/chindada/panther/golang/pb"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type tradeRoutes struct {
	t usecases.Trade

	tradePool sync.Pool
}

func NewTradeRoutes(handler *gin.RouterGroup, ws *gin.RouterGroup, t usecases.Trade) {
	r := &tradeRoutes{
		t:         t,
		tradePool: sync.Pool{New: func() any { return &pb.Trade{} }},
	}
	base := "/trade"
	h := handler.Group(base)
	{
		h.GET("/records", r.getAllRecords)
		h.POST("/records", r.getRecords)
		h.PUT("/records", r.triggerUpdateRecords)

		h.POST("/cancel", r.cancelTrade)

		h.POST("/buy/future", r.buyFuture)
		h.POST("/sell/future", r.sellFuture)

		h.GET("/margin", r.getMargin)
		h.GET("/future/position", r.getFuturePosition)
	}
	w := ws.Group(base)
	{
		w.GET("/trigger", r.streamTrade)
		w.GET("/single/trigger", r.streamSingleCodeTrade)
	}
}

// getAllRecords -.
//
//	@Tags		Trade V1
//	@Summary	Get all trade records
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Param		code	query		string	false	"code"
//	@Success	200		{object}	pb.TradeList
//	@Failure	500		{object}	pb.APIResponse
//	@Router		/api/capitan/v1/trade/records [get]
func (r *tradeRoutes) getAllRecords(c *gin.Context) {
	code := c.Query("code")
	var trades []*pb.Trade
	var err error
	if code == "" {
		trades, err = r.t.GetTrades(&pb.QueryTradeRequest{})
		if err != nil {
			resp.Fail(c, http.StatusInternalServerError, err)
			return
		}
	} else {
		trades, err = r.t.GetUndoneTradesByCode(code)
		if err != nil {
			resp.Fail(c, http.StatusInternalServerError, err)
			return
		}
	}
	resp.Success(c, http.StatusOK, &pb.TradeList{List: trades})
}

// getRecords -.
//
//	@Tags		Trade V1
//	@Summary	Get trade records by request
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@param		body	body		pb.QueryTradeRequest	true	"Body"
//	@Success	200		{object}	pb.TradeList
//	@Failure	400		{object}	pb.APIResponse
//	@Failure	500		{object}	pb.APIResponse
//	@Router		/api/capitan/v1/trade/records [post]
func (r *tradeRoutes) getRecords(c *gin.Context) {
	req := &pb.QueryTradeRequest{}
	err := c.Bind(req)
	if err != nil {
		resp.Fail(c, http.StatusBadRequest, err)
		return
	}
	if req.GetOrderId() == "" && (!req.GetStartTime().IsValid() || !req.GetEndTime().IsValid()) {
		resp.Fail(c, http.StatusBadRequest, errors.New("either OrderId or time range must be specified"))
		return
	}
	trades, err := r.t.GetTrades(req)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &pb.TradeList{List: trades})
}

// triggerUpdateRecords -.
//
//	@Tags		Trade V1
//	@Summary	Get trade records by request
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Success	200	{object}	emptypb.Empty
//	@Failure	500	{object}	pb.APIResponse
//	@Router		/api/capitan/v1/trade/records [put]
func (r *tradeRoutes) triggerUpdateRecords(c *gin.Context) {
	err := r.t.TriggerUpdateAndPublishTrade()
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &emptypb.Empty{})
}

// streamTrade /ws/capitan/v1/trade/trigger [get].
func (r *tradeRoutes) streamTrade(c *gin.Context) {
	forwardChan := make(chan []byte)
	ws, err := ws.New(c, forwardChan)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	clientID := uuid.NewString()
	go func() {
		defer func() {
			r.t.CloseTradeClient(clientID)
		}()
		for {
			r.t.CreateTradeClient(&usecases.TradeClient{
				ID:      clientID,
				Channel: r.sendTrade(ws),
			})
			_, ok := <-forwardChan
			if !ok {
				break
			}
		}
	}()
	ws.ReadMessage()
}

// streamSingleCodeTrade /ws/capitan/v1/trade/single/trigger [get].
func (r *tradeRoutes) streamSingleCodeTrade(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		resp.Fail(c, http.StatusBadRequest, resp.ErrNotFound)
		return
	}

	forwardChan := make(chan []byte)
	ws, err := ws.New(c, forwardChan)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	clientID := uuid.NewString()
	go func() {
		defer func() {
			r.t.CloseSingleCodeClient(clientID, code)
		}()
		for {
			r.t.CreateSingleCodeClient(code, &usecases.TradeClient{
				ID:      clientID,
				Channel: r.sendTrade(ws),
			})
			_, ok := <-forwardChan
			if !ok {
				break
			}
		}
	}()
	ws.ReadMessage()
}

func (r *tradeRoutes) sendTrade(ws ws.WS) chan *pb.Trade {
	channel := make(chan *pb.Trade)
	go func() {
		for {
			cl, ok := <-channel
			if !ok {
				return
			}
			trade, _ := r.tradePool.Get().(*pb.Trade)
			trade.Uid = cl.GetUid()
			trade.Type = cl.GetType()
			trade.Code = cl.GetCode()
			trade.OrderId = cl.GetOrderId()
			trade.Action = cl.GetAction()
			trade.Price = cl.GetPrice()
			trade.Quantity = cl.GetQuantity()
			trade.FilledQuantity = cl.GetFilledQuantity()
			trade.Status = cl.GetStatus()
			trade.OrderTime = cl.GetOrderTime()
			trade.Stock = cl.GetStock()
			trade.Future = cl.GetFuture()
			trade.Option = cl.GetOption()
			b, mErr := proto.Marshal(trade)
			if mErr == nil {
				ws.WriteBinaryMessage(b)
			}
			r.tradePool.Put(trade)
		}
	}()
	return channel
}

// cancelTrade -.
//
//	@Tags		Trade V1
//	@Summary	Cancel trade
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@param		body	body		pb.Trade	true	"Body"
//	@Success	200		{object}	pb.Trade
//	@Failure	400		{object}	pb.APIResponse
//	@Failure	500		{object}	pb.APIResponse
//	@Router		/api/capitan/v1/trade/cancel [post]
func (r *tradeRoutes) cancelTrade(c *gin.Context) {
	req := &pb.Trade{}
	err := c.Bind(req)
	if err != nil {
		resp.Fail(c, http.StatusBadRequest, err)
		return
	}
	result, err := r.t.CancelTrade(c, req)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, result)
}

// buyFuture -.
//
//	@Tags		Trade V1
//	@Summary	Buy future
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@param		body	body		pb.OrderDetail	true	"Body"
//	@Success	200		{object}	pb.Trade
//	@Failure	400		{object}	pb.APIResponse
//	@Failure	500		{object}	pb.APIResponse
//	@Router		/api/capitan/v1/trade/buy/future [post]
func (r *tradeRoutes) buyFuture(c *gin.Context) {
	req := &pb.OrderDetail{}
	err := c.Bind(req)
	if err != nil {
		resp.Fail(c, http.StatusBadRequest, err)
		return
	}
	result, err := r.t.BuyFuture(c, req)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, result)
}

// sellFuture -.
//
//	@Tags		Trade V1
//	@Summary	Sell future
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@param		body	body		pb.OrderDetail	true	"Body"
//	@Success	200		{object}	pb.Trade
//	@Failure	400		{object}	pb.APIResponse
//	@Failure	500		{object}	pb.APIResponse
//	@Router		/api/capitan/v1/trade/sell/future [post]
func (r *tradeRoutes) sellFuture(c *gin.Context) {
	req := &pb.OrderDetail{}
	err := c.Bind(req)
	if err != nil {
		resp.Fail(c, http.StatusBadRequest, err)
		return
	}
	result, err := r.t.SellFuture(c, req)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, result)
}

// getMargin -.
//
//	@Tags		Trade V1
//	@Summary	Get margin information
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Success	200	{object}	pb.Margin
//	@Failure	500	{object}	pb.APIResponse
//	@Router		/api/capitan/v1/trade/margin [get]
func (r *tradeRoutes) getMargin(c *gin.Context) {
	margin, err := r.t.GetMargin(c)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, margin)
}

// getAllFuturePosition -.
//
//	@Tags		Trade V1
//	@Summary	Get all future positions
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Param		code	query		string	false	"code"
//	@Success	200		{object}	pb.FuturePositionList
//	@Failure	500		{object}	pb.APIResponse
//	@Router		/api/capitan/v1/trade/future/position [get]
func (r *tradeRoutes) getFuturePosition(c *gin.Context) {
	code := c.Query("code")
	var positionList *pb.FuturePositionList
	var err error
	if code == "" {
		positionList, err = r.t.GetFuturePosition(c)
		if err != nil {
			resp.Fail(c, http.StatusInternalServerError, err)
			return
		}
	} else {
		positionList, err = r.t.GetFuturePositionByCode(c, code)
		if err != nil {
			resp.Fail(c, http.StatusInternalServerError, err)
			return
		}
	}
	resp.Success(c, http.StatusOK, positionList)
}
