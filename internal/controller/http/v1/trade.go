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
	}
	w := ws.Group(base)
	{
		w.GET("/trigger", r.streamTrade)
	}
}

// getAllRecords -.
//
//	@Tags		Trade V1
//	@Summary	Get all trade records
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Success	200	{object}	pb.TradeList
//	@Failure	400	{object}	pb.APIResponse
//	@Failure	500	{object}	pb.APIResponse
//	@Router		/api/capitan/v1/trade/records [get]
func (r *tradeRoutes) getAllRecords(c *gin.Context) {
	trades, err := r.t.GetTrades(&pb.QueryTradeRequest{})
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
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
			r.t.CreateTradeClient(clientID, r.sendTrade(ws))
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
