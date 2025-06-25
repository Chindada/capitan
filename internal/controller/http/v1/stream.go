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
)

type streamRoutes struct {
	t usecases.Stream

	futureTickPool   sync.Pool
	futureBidAskPool sync.Pool
}

func NewStreamRoutes(ws *gin.RouterGroup, t usecases.Stream) {
	r := &streamRoutes{
		t:                t,
		futureTickPool:   sync.Pool{New: func() any { return &pb.FutureStream_Tick{} }},
		futureBidAskPool: sync.Pool{New: func() any { return &pb.FutureStream_BidAsk{} }},
	}
	w := ws.Group("/stream")
	{
		w.GET("/futures/trigger", r.streamFutrues)
	}
}

// triggerWorkshop /ws/capitan/v1/stream/futures/trigger [get].
func (r *streamRoutes) streamFutrues(c *gin.Context) {
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
			r.t.CreateFutureClient(clientID, &usecases.FutureClient{
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
		}
	}()
	return channel
}
