package usecases

import (
	"context"
	"fmt"
	"sync"

	"github.com/chindada/capitan/internal/config"
	"github.com/chindada/leopard/pkg/eventbus"
	"github.com/chindada/leopard/pkg/log"
	"github.com/chindada/panther/golang/pb"
)

//go:generate mockgen -source=usecase_stream.go -destination=./mocks/mocks_usecase_stream_test.go -package=mocks

type Stream interface {
	CreateFutureClient(client *FutureClient)
	CloseFutureClient(clientID string)
	CreateSingleFutureClient(code string, client *FutureClient)
	CloseSingleFutureClient(clientID, code string)

	CreateStockClient(client *StockClient)
	CloseStockClient(clientID string)
}

type streamUseCase struct {
	logger *log.Log
	bus    *eventbus.Bus

	streamClient pb.StreamInterfaceClient

	singleFutureClientMap sync.Map

	futureClientMap  map[string]*FutureClient
	futureClientLock sync.RWMutex

	stockClientMap  map[string]*StockClient
	stockClientLock sync.RWMutex
}

func NewStream() Stream {
	cfg := config.Get()
	uc := &streamUseCase{
		logger:          log.Get(),
		bus:             eventbus.Get(),
		streamClient:    pb.NewStreamInterfaceClient(cfg.GetGRPCConn()),
		futureClientMap: make(map[string]*FutureClient),
		stockClientMap:  make(map[string]*StockClient),
	}
	uc.bus.SubscribeAsync(topicStreamSubscribeStockQuote, false, uc.subscribeStockQuote)
	uc.bus.SubscribeAsync(topicStreamSubscribeFutureTick, false, uc.subscribeFutureTick)
	uc.bus.SubscribeAsync(topicStreamSubscribeFutureBidAsk, false, uc.subscribeFutureBidAsk)
	return uc
}

func (uc *streamUseCase) sendSingleStockQuote() chan *pb.StockQuote {
	ch := make(chan *pb.StockQuote)
	go func() {
		for {
			quote := <-ch
			channels, ok := uc.singleFutureClientMap.Load(quote.GetCode())
			if !ok {
				continue
			}
			chs, _ := channels.([]*StockClient)
			for _, client := range chs {
				client.QuoteChannel <- quote
			}
		}
	}()
	return ch
}

func (uc *streamUseCase) sendStockQuote() chan *pb.StockQuote {
	channel := make(chan *pb.StockQuote)
	go func() {
		singleChannel := uc.sendSingleStockQuote()
		for {
			tick := <-channel
			singleChannel <- tick
			uc.stockClientLock.RLock()
			for _, client := range uc.stockClientMap {
				client.QuoteChannel <- tick
			}
			uc.stockClientLock.RUnlock()
		}
	}()
	return channel
}

func (uc *streamUseCase) sendSingleFutureTick() chan *pb.FutureTick {
	ch := make(chan *pb.FutureTick)
	go func() {
		for {
			tick := <-ch
			channels, ok := uc.singleFutureClientMap.Load(tick.GetCode())
			if !ok {
				continue
			}
			chs, _ := channels.([]*FutureClient)
			for _, client := range chs {
				client.TickChannel <- tick
			}
		}
	}()
	return ch
}

func (uc *streamUseCase) sendFutureTick() chan *pb.FutureTick {
	channel := make(chan *pb.FutureTick)
	go func() {
		singleChannel := uc.sendSingleFutureTick()
		for {
			tick := <-channel
			singleChannel <- tick
			uc.futureClientLock.RLock()
			for _, client := range uc.futureClientMap {
				client.TickChannel <- tick
			}
			uc.futureClientLock.RUnlock()
		}
	}()
	return channel
}

func (uc *streamUseCase) sendSingleFutureBidAsk() chan *pb.FutureBidAsk {
	ch := make(chan *pb.FutureBidAsk)
	go func() {
		for {
			bidAsk := <-ch
			channels, ok := uc.singleFutureClientMap.Load(bidAsk.GetCode())
			if !ok {
				continue
			}
			chs, _ := channels.([]*FutureClient)
			for _, client := range chs {
				client.BidAskChannel <- bidAsk
			}
		}
	}()
	return ch
}

func (uc *streamUseCase) sendFutureBidAsk() chan *pb.FutureBidAsk {
	channel := make(chan *pb.FutureBidAsk)
	go func() {
		singleChannel := uc.sendSingleFutureBidAsk()
		for {
			bidAsk := <-channel
			singleChannel <- bidAsk
			uc.futureClientLock.RLock()
			for _, client := range uc.futureClientMap {
				client.BidAskChannel <- bidAsk
			}
			uc.futureClientLock.RUnlock()
		}
	}()
	return channel
}

func (uc *streamUseCase) subscribeStockQuote(detail *pb.StockDetail) {
	tickStream, err := uc.streamClient.SubscribeStockQuote(context.Background(), &pb.SubscribeRequest{
		Code: detail.GetCode(),
	})
	if err != nil {
		uc.logger.Errorf("Failed to subscribe to future tick for code %s: %v", detail.GetCode(), err)
		return
	}
	ch := uc.sendStockQuote()
	for {
		t, rErr := tickStream.Recv()
		if rErr != nil {
			return
		}
		ch <- t
	}
}

func (uc *streamUseCase) subscribeFutureTick(detail *pb.FutureDetail) {
	tickStream, err := uc.streamClient.SubscribeFutureTick(context.Background(), &pb.SubscribeRequest{
		Code: detail.GetCode(),
	})
	if err != nil {
		uc.logger.Errorf("Failed to subscribe to future tick for code %s: %v", detail.GetCode(), err)
		return
	}
	ch := uc.sendFutureTick()
	for {
		t, rErr := tickStream.Recv()
		if rErr != nil {
			return
		}
		uc.bus.PublishTopicEvent(fmt.Sprintf("%s/%s", topicStreamSubscribeFutureTick, detail.GetCode()), t)
		ch <- t
	}
}

func (uc *streamUseCase) subscribeFutureBidAsk(detail *pb.FutureDetail) {
	bidAskStream, err := uc.streamClient.SubscribeFutureBidAsk(context.Background(), &pb.SubscribeRequest{
		Code: detail.GetCode(),
	})
	if err != nil {
		uc.logger.Errorf("Failed to subscribe to future bid-ask for code %s: %v", detail.GetCode(), err)
		return
	}
	ch := uc.sendFutureBidAsk()
	for {
		bidAsk, rErr := bidAskStream.Recv()
		if rErr != nil {
			return
		}
		uc.bus.PublishTopicEvent(fmt.Sprintf("%s/%s", topicStreamSubscribeFutureBidAsk, detail.GetCode()), bidAsk)
		ch <- bidAsk
	}
}

type FutureClient struct {
	ClientID      string
	TickChannel   chan *pb.FutureTick
	BidAskChannel chan *pb.FutureBidAsk
}

func (uc *streamUseCase) CreateFutureClient(client *FutureClient) {
	uc.futureClientLock.Lock()
	uc.futureClientMap[client.ClientID] = client
	uc.futureClientLock.Unlock()
}

func (uc *streamUseCase) CloseFutureClient(clientID string) {
	if clientID == "" {
		return
	}

	defer uc.futureClientLock.Unlock()
	uc.futureClientLock.Lock()

	if c, exist := uc.futureClientMap[clientID]; exist {
		close(c.TickChannel)
		close(c.BidAskChannel)
		delete(uc.futureClientMap, clientID)
	}
}

func (uc *streamUseCase) CreateSingleFutureClient(code string, client *FutureClient) {
	channels, _ := uc.singleFutureClientMap.LoadOrStore(code, []*FutureClient{})
	chs, _ := channels.([]*FutureClient)
	chs = append(chs, client)
	uc.singleFutureClientMap.Store(code, chs)
}

func (uc *streamUseCase) CloseSingleFutureClient(clientID, code string) {
	if clientID == "" || code == "" {
		return
	}

	channels, exist := uc.singleFutureClientMap.Load(code)
	if !exist {
		return
	}

	chs, _ := channels.([]*FutureClient)
	for i, c := range chs {
		if c.ClientID == clientID {
			close(c.TickChannel)
			close(c.BidAskChannel)
			chs = append(chs[:i], chs[i+1:]...)
			break
		}
	}
	if len(chs) == 0 {
		uc.singleFutureClientMap.Delete(code)
	} else {
		uc.singleFutureClientMap.Store(code, chs)
	}
}

type StockClient struct {
	ClientID     string
	QuoteChannel chan *pb.StockQuote
}

func (uc *streamUseCase) CreateStockClient(client *StockClient) {
	uc.stockClientLock.Lock()
	uc.stockClientMap[client.ClientID] = client
	uc.stockClientLock.Unlock()
}

func (uc *streamUseCase) CloseStockClient(clientID string) {
	if clientID == "" {
		return
	}

	defer uc.stockClientLock.Unlock()
	uc.stockClientLock.Lock()

	if c, exist := uc.stockClientMap[clientID]; exist {
		close(c.QuoteChannel)
		delete(uc.stockClientMap, clientID)
	}
}
