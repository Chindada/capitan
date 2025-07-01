package v1

import (
	"net/http"
	"sync"

	"github.com/chindada/capitan/internal/controller/http/resp"
	"github.com/chindada/capitan/internal/controller/http/ws"
	"github.com/chindada/capitan/internal/usecases"
	"github.com/chindada/panther/golang/pb"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type streamRoutes struct {
	t usecases.Stream

	futureTickPool   sync.Pool
	futureBidAskPool sync.Pool
}

func NewStreamRoutes(handler *gin.RouterGroup, ws *gin.RouterGroup, t usecases.Stream) {
	r := &streamRoutes{
		t:                t,
		futureTickPool:   sync.Pool{New: func() any { return &pb.FutureStream_Tick{} }},
		futureBidAskPool: sync.Pool{New: func() any { return &pb.FutureStream_BidAsk{} }},
	}
	base := "/stream"
	h := handler.Group(base)
	{
		h.GET("/subscribe/codes", r.getAllSubscribeCodes)
	}
	w := ws.Group(base)
	{
		w.GET("/futures/trigger", r.streamAllFutrues)
		w.GET("/futures/single/trigger", r.streamSingleFutrues)
	}
}

// getAllRecords -.
//
//	@Tags		Stream V1
//	@Summary	Get all trade records
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Success	200	{object}	structpb.ListValue
//	@Router		/api/capitan/v1/stream/subscribe/codes [get]
func (r *streamRoutes) getAllSubscribeCodes(c *gin.Context) {
	codes := r.t.GetAllSubscribeCodes()
	arr := make([]any, len(codes))
	for i, code := range codes {
		arr[i] = code
	}
	pbArr, err := structpb.NewList(arr)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, pbArr)
}

// streamSingleFutrues /ws/capitan/v1/stream/futures/single/trigger [get].
func (r *streamRoutes) streamSingleFutrues(c *gin.Context) {
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
			r.t.CloseSingleFutureClient(clientID, code)
		}()
		for {
			r.t.CreateSingleFutureClient(code, &usecases.FutureClient{
				ClientID:      clientID,
				TickChannel:   r.sendTick(ws),
				BidAskChannel: r.sendBidAsk(ws),
			})
			_, ok := <-forwardChan
			if !ok {
				break
			}
		}
	}()
	ws.ReadMessage()
}

// streamAllFutrues /ws/capitan/v1/stream/futures/trigger [get].
func (r *streamRoutes) streamAllFutrues(c *gin.Context) {
	forwardChan := make(chan []byte)
	ws, err := ws.New(c, forwardChan)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	clientID := uuid.NewString()
	go func() {
		defer func() {
			r.t.CloseFutureClient(clientID)
		}()
		for {
			r.t.CreateFutureClient(&usecases.FutureClient{
				ClientID:      clientID,
				TickChannel:   r.sendTick(ws),
				BidAskChannel: r.sendBidAsk(ws),
			})
			_, ok := <-forwardChan
			if !ok {
				break
			}
		}
	}()
	ws.ReadMessage()
}

func (r *streamRoutes) sendTick(ws ws.WS) chan *pb.FutureTick {
	channel := make(chan *pb.FutureTick)
	go func() {
		data := &pb.FutureStream{}
		for {
			cl, ok := <-channel
			if !ok {
				return
			}
			tick, _ := r.futureTickPool.Get().(*pb.FutureStream_Tick)
			tick.Tick = cl
			data.Code = cl.GetCode()
			data.Data = tick
			b, mErr := proto.Marshal(data)
			if mErr == nil {
				ws.WriteBinaryMessage(b)
			}
			r.futureTickPool.Put(tick)
		}
	}()
	return channel
}

func (r *streamRoutes) sendBidAsk(ws ws.WS) chan *pb.FutureBidAsk {
	channel := make(chan *pb.FutureBidAsk)
	go func() {
		data := &pb.FutureStream{}
		for {
			cl, ok := <-channel
			if !ok {
				return
			}
			bidAsk, _ := r.futureBidAskPool.Get().(*pb.FutureStream_BidAsk)
			bidAsk.BidAsk = cl
			data.Code = cl.GetCode()
			data.Data = bidAsk
			b, mErr := proto.Marshal(data)
			if mErr == nil {
				ws.WriteBinaryMessage(b)
			}
			r.futureBidAskPool.Put(bidAsk)
		}
	}()
	return channel
}
