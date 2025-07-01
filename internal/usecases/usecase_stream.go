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
	GetAllSubscribeCodes() []string

	CreateFutureClient(client *FutureClient)
	CloseFutureClient(clientID string)

	CreateSingleFutureClient(code string, client *FutureClient)
	CloseSingleFutureClient(clientID, code string)
}

type streamUseCase struct {
	logger *log.Log
	bus    *eventbus.Bus

	streamClient pb.StreamInterfaceClient

	singleFutureClientMap sync.Map

	futureClientMap  map[string]*FutureClient
	futureClientLock sync.RWMutex

	clientTickChannel   chan *pb.FutureTick
	clientBidAskChannel chan *pb.FutureBidAsk

	subscribeCodeMap map[string]bool
	subscribeLock    sync.RWMutex
}

func NewStream() Stream {
	cfg := config.Get()
	uc := &streamUseCase{
		logger:              log.Get(),
		bus:                 eventbus.Get(),
		streamClient:        pb.NewStreamInterfaceClient(cfg.GetGRPCConn()),
		futureClientMap:     make(map[string]*FutureClient),
		clientTickChannel:   make(chan *pb.FutureTick),
		clientBidAskChannel: make(chan *pb.FutureBidAsk),
		subscribeCodeMap:    make(map[string]bool),
	}

	go uc.sendFutureTick()
	go uc.sendFutureBidAsk()

	uc.bus.SubscribeAsync(topicStreamSubscribeFutureTick, false, uc.subscribeFutureTick)
	uc.bus.SubscribeAsync(topicStreamSubscribeFutureBidAsk, false, uc.subscribeFutureBidAsk)

	return uc
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

func (uc *streamUseCase) sendFutureTick() {
	singleChannel := uc.sendSingleFutureTick()
	for {
		tick := <-uc.clientTickChannel
		singleChannel <- tick
		uc.futureClientLock.RLock()
		for _, client := range uc.futureClientMap {
			client.TickChannel <- tick
		}
		uc.futureClientLock.RUnlock()
	}
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

func (uc *streamUseCase) sendFutureBidAsk() {
	singleChannel := uc.sendSingleFutureBidAsk()
	for {
		bidAsk := <-uc.clientBidAskChannel
		singleChannel <- bidAsk
		uc.futureClientLock.RLock()
		for _, client := range uc.futureClientMap {
			client.BidAskChannel <- bidAsk
		}
		uc.futureClientLock.RUnlock()
	}
}

func (uc *streamUseCase) subscribeFutureTick(code string) {
	tickStream, err := uc.streamClient.SubscribeFutureTick(context.Background(), &pb.SubscribeFutureRequest{
		Code: code,
	})
	if err != nil {
		uc.logger.Errorf("Failed to subscribe to future tick for code %s: %v", code, err)
		return
	}
	uc.subscribeLock.Lock()
	uc.subscribeCodeMap[code] = true
	uc.subscribeLock.Unlock()
	for {
		t, rErr := tickStream.Recv()
		if rErr != nil {
			return
		}
		uc.bus.PublishTopicEvent(fmt.Sprintf("%s/%s", topicStreamSubscribeFutureTick, code), t)
		uc.clientTickChannel <- t
	}
}

func (uc *streamUseCase) subscribeFutureBidAsk(code string) {
	bidAskStream, err := uc.streamClient.SubscribeFutureBidAsk(context.Background(), &pb.SubscribeFutureRequest{
		Code: code,
	})
	if err != nil {
		uc.logger.Errorf("Failed to subscribe to future bid-ask for code %s: %v", code, err)
		return
	}
	for {
		bidAsk, rErr := bidAskStream.Recv()
		if rErr != nil {
			return
		}
		uc.bus.PublishTopicEvent(fmt.Sprintf("%s/%s", topicStreamSubscribeFutureBidAsk, code), bidAsk)
		uc.clientBidAskChannel <- bidAsk
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

func (uc *streamUseCase) GetAllSubscribeCodes() []string {
	uc.subscribeLock.RLock()
	defer uc.subscribeLock.RUnlock()

	codes := make([]string, 0, len(uc.subscribeCodeMap))
	for code := range uc.subscribeCodeMap {
		codes = append(codes, code)
	}
	return codes
}
