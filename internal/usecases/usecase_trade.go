package usecases

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/chindada/capitan/internal/config"
	"github.com/chindada/capitan/internal/usecases/repo"
	"github.com/chindada/leopard/pkg/eventbus"
	"github.com/chindada/leopard/pkg/log"
	"github.com/chindada/panther/golang/pb"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

//go:generate mockgen -source=usecase_trade.go -destination=./mocks/mocks_usecase_trade_test.go -package=mocks

type Trade interface {
	TriggerUpdateAndPublishTrade() error
	GetTrades(req *pb.QueryTradeRequest) ([]*pb.Trade, error)
	GetTradeByOrderID(orderID string) (*pb.Trade, error)

	CreateTradeClient(clientID string, client chan *pb.Trade)
	CloseTradeClient(clientID string)
}

type tradeUseCase struct {
	tradeRepo repo.TradeRepo

	logger *log.Log
	bus    *eventbus.Bus

	tradeClient pb.TradeInterfaceClient

	tradeChannel    chan *pb.Trade
	tradeClientMap  map[string]chan *pb.Trade
	tradeClientLock sync.RWMutex
}

func NewTrade() Trade {
	cfg := config.Get()
	pg := cfg.GetPostgresPool()
	uc := &tradeUseCase{
		tradeRepo:      repo.NewTrade(pg),
		logger:         log.Get(),
		bus:            eventbus.Get(),
		tradeClient:    pb.NewTradeInterfaceClient(cfg.GetGRPCConn()),
		tradeClientMap: make(map[string]chan *pb.Trade),
		tradeChannel:   make(chan *pb.Trade),
	}

	go uc.sendTrade()
	go uc.subscribeTrade()

	uc.bus.SubscribeAsync(topicStreamSubscribeFutureTick, false, uc.subscribeFutureTick)
	uc.bus.SubscribeAsync(topicStreamSubscribeFutureBidAsk, false, uc.subscribeFutureBidAsk)
	return uc
}

func (uc *tradeUseCase) GetTrades(req *pb.QueryTradeRequest) ([]*pb.Trade, error) {
	if req.GetOrderId() != "" && req.GetStartTime().IsValid() && req.GetEndTime().IsValid() {
		return nil, errors.New("cannot specify both OrderId and time range")
	}
	return uc.tradeRepo.SelectTradesByRequest(context.Background(), req)
}

func (uc *tradeUseCase) TriggerUpdateAndPublishTrade() error {
	_, err := uc.tradeClient.UpdateAndPublishTrade(context.Background(), &emptypb.Empty{})
	if err != nil {
		s := status.Convert(err)
		return fmt.Errorf("error(%d): %s", s.Code(), s.Message())
	}
	return nil
}

func (uc *tradeUseCase) GetTradeByOrderID(orderID string) (*pb.Trade, error) {
	ctx := context.Background()
	trade, err := uc.tradeClient.GetTradeByOrderID(ctx, &pb.QueryTradeRequest{OrderId: orderID})
	if err != nil {
		s := status.Convert(err)
		return nil, fmt.Errorf("error(%d): %s", s.Code(), s.Message())
	}
	return trade, nil
}

func (uc *tradeUseCase) sendTrade() {
	for {
		t := <-uc.tradeChannel
		uc.tradeClientLock.RLock()
		for _, client := range uc.tradeClientMap {
			client <- t
		}
		uc.tradeClientLock.RUnlock()
	}
}

func (uc *tradeUseCase) subscribeTrade() {
	trade, err := uc.tradeClient.SubscribeTrade(context.Background(), &emptypb.Empty{})
	if err != nil {
		s := status.Convert(err)
		uc.logger.Fatalf("Error(%d): %s", s.Code(), s.Message())
	}
	for {
		t, rErr := trade.Recv()
		if rErr != nil {
			s := status.Convert(rErr)
			uc.logger.Fatalf("Error(%d): %s", s.Code(), s.Message())
		}
		uc.tradeChannel <- t
		go func() {
			err = uc.tradeRepo.InsertOrUpdateTrade(context.Background(), t)
			if err != nil {
				uc.logger.Errorf("Failed to insert trade: %v", err)
			}
		}()
	}
}

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

func (uc *tradeUseCase) CreateTradeClient(clientID string, client chan *pb.Trade) {
	uc.tradeClientLock.Lock()
	uc.tradeClientMap[clientID] = client
	uc.tradeClientLock.Unlock()
}

func (uc *tradeUseCase) CloseTradeClient(clientID string) {
	if clientID == "" {
		return
	}

	defer uc.tradeClientLock.Unlock()
	uc.tradeClientLock.Lock()

	if c, exist := uc.tradeClientMap[clientID]; exist {
		close(c)
		for {
			_, ok := <-c
			if !ok {
				break
			}
		}
		delete(uc.tradeClientMap, clientID)
	}
}
