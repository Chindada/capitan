package usecases

import (
	"fmt"

	"github.com/chindada/leopard/pkg/eventbus"
	"github.com/chindada/leopard/pkg/log"
	"github.com/chindada/panther/golang/pb"
)

//go:generate mockgen -source=usecase_trade.go -destination=./mocks/mocks_usecase_trade_test.go -package=mocks

type Trade interface {
	TradeMock()
}

type tradeUseCase struct {
	logger *log.Log
	bus    *eventbus.Bus
}

func NewTrade() Trade {
	uc := &tradeUseCase{
		logger: log.Get(),
		bus:    eventbus.Get(),
	}
	uc.bus.SubscribeAsync(topicStreamSubscribeFutureTick, false, uc.subscribeFutureTick)
	uc.bus.SubscribeAsync(topicStreamSubscribeFutureBidAsk, false, uc.subscribeFutureBidAsk)
	return uc
}

func (uc *tradeUseCase) TradeMock() {}

func (uc *tradeUseCase) subscribeFutureTick(code string) {
	fn := func(_ *pb.FutureTick) {
		// tickTime, _ := time.ParseInLocation(time.DateTime, tick.GetDateTime(), time.Local)
		// fmt.Printf("Received tick: %s, price: %f, volume: %d, time: %v, time gap: %d\n",
		// 	tick.GetCode(), tick.GetClose(), tick.GetVolume(), tickTime, time.Since(tickTime).Milliseconds())
	}
	uc.bus.Subscribe(fmt.Sprintf("%s/%s", topicStreamSubscribeFutureTick, code), fn)
}

func (uc *tradeUseCase) subscribeFutureBidAsk(code string) {
	fn := func(_ *pb.FutureBidAsk) {
		// bidAskTime, _ := time.ParseInLocation(time.DateTime, bidAsk.GetDateTime(), time.Local)
		// fmt.Printf("Received bid-ask: %s, time: %v, time gap: %d\n", bidAsk.GetCode(), bidAskTime, time.Since(bidAskTime).Milliseconds())
	}
	uc.bus.Subscribe(fmt.Sprintf("%s/%s", topicStreamSubscribeFutureBidAsk, code), fn)
}
