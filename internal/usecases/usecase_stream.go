package usecases

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/chindada/capitan/internal/config"
	"github.com/chindada/leopard/pkg/eventbus"
	"github.com/chindada/leopard/pkg/log"
	"github.com/chindada/panther/golang/pb"
	"github.com/maruel/natural"
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

	futureCodes []string
	stockCodes  []string
	targetLock  sync.RWMutex
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
	uc.targetLock.Lock()
	uc.stockCodes = append(uc.stockCodes, detail.GetCode())
	sort.SliceStable(uc.stockCodes, func(i, j int) bool {
		return natural.Less(uc.stockCodes[i], uc.stockCodes[j])
	})
	uc.targetLock.Unlock()
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
	uc.targetLock.Lock()
	uc.futureCodes = append(uc.futureCodes, detail.GetCode())
	sort.SliceStable(uc.futureCodes, func(i, j int) bool {
		return natural.Less(uc.futureCodes[i], uc.futureCodes[j])
	})
	uc.targetLock.Unlock()
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
	uc.targetLock.RLock()
	defer uc.targetLock.RUnlock()

	snapshots, err := uc.streamClient.GetSnapshot(context.Background(), &pb.SnapshotRequest{
		Type:  pb.SnapshotRequestType_SNAPSHOT_REQUEST_TYPE_FUTURE,
		Codes: uc.futureCodes,
	})
	if err != nil {
		uc.logger.Errorf("Failed to get future snapshot: %v", err)
		return
	}
	for _, v := range uc.futureCodes {
		code := v
		if snapshots == nil || snapshots.GetSnapshots() == nil {
			continue
		}
		s, exist := snapshots.GetSnapshots()[code]
		if !exist {
			uc.logger.Warnf("No snapshot found for code %s", code)
			continue
		}
		client.TickChannel <- &pb.FutureTick{
			Code:     code,
			Close:    s.GetClose(),
			PriceChg: s.GetChangePrice(),
		}
	}
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
	snapshots, err := uc.streamClient.GetSnapshot(context.Background(), &pb.SnapshotRequest{
		Type:  pb.SnapshotRequestType_SNAPSHOT_REQUEST_TYPE_FUTURE,
		Codes: []string{code},
	})
	if err != nil {
		uc.logger.Errorf("Failed to get future snapshot: %v", err)
		return
	}
	if snapshots == nil || snapshots.GetSnapshots() == nil {
		uc.logger.Warnf("No snapshot found for code %s", code)
		return
	}
	s, exist := snapshots.GetSnapshots()[code]
	if !exist {
		uc.logger.Warnf("No snapshot found for code %s", code)
		return
	}
	client.TickChannel <- &pb.FutureTick{
		Code:     code,
		Close:    s.GetClose(),
		PriceChg: s.GetChangePrice(),
	}
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
	uc.targetLock.RLock()
	defer uc.targetLock.RUnlock()

	snapshots, err := uc.streamClient.GetSnapshot(context.Background(), &pb.SnapshotRequest{
		Type:  pb.SnapshotRequestType_SNAPSHOT_REQUEST_TYPE_STOCK,
		Codes: uc.stockCodes,
	})
	if err != nil {
		uc.logger.Errorf("Failed to get future snapshot: %v", err)
		return
	}
	for _, v := range uc.stockCodes {
		code := v
		if snapshots == nil || snapshots.GetSnapshots() == nil {
			continue
		}
		s, exist := snapshots.GetSnapshots()[code]
		if !exist {
			uc.logger.Warnf("No snapshot found for code %s", code)
			continue
		}
		client.QuoteChannel <- &pb.StockQuote{
			Code:     code,
			Close:    s.GetClose(),
			PriceChg: s.GetChangePrice(),
		}
	}
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
