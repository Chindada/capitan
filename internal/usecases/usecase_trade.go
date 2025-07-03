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
	CreateSingleCodeClient(code string, client *TradeClient)
	CloseSingleCodeClient(clientID, code string)

	CreateTradeClient(client *TradeClient)
	CloseTradeClient(clientID string)

	GetTrades(req *pb.QueryTradeRequest) ([]*pb.Trade, error)
	GetUndoneTradesByCode(code string) ([]*pb.Trade, error)

	TriggerUpdateAndPublishTrade() error
	GetTradeByOrderID(orderID string) (*pb.Trade, error)
	CancelTrade(ctx context.Context, in *pb.Trade) (*pb.Trade, error)

	BuyFuture(ctx context.Context, in *pb.OrderDetail) (*pb.Trade, error)
	SellFuture(ctx context.Context, in *pb.OrderDetail) (*pb.Trade, error)

	GetFuturePositionByCode(ctx context.Context, code string) (*pb.FuturePositionList, error)
	GetFuturePosition(ctx context.Context) (*pb.FuturePositionList, error)
	GetMargin(ctx context.Context) (*pb.Margin, error)
}

type tradeUseCase struct {
	tradeRepo repo.TradeRepo

	logger *log.Log
	bus    *eventbus.Bus

	tradeClient pb.TradeInterfaceClient

	tradeChannel    chan *pb.Trade
	tradeClientMap  map[string]*TradeClient
	tradeClientLock sync.RWMutex

	SingleCodeClientMap sync.Map
}

func NewTrade() Trade {
	cfg := config.Get()
	pg := cfg.GetPostgresPool()
	uc := &tradeUseCase{
		tradeRepo:      repo.NewTrade(pg),
		logger:         log.Get(),
		bus:            eventbus.Get(),
		tradeClient:    pb.NewTradeInterfaceClient(cfg.GetGRPCConn()),
		tradeClientMap: make(map[string]*TradeClient),
		tradeChannel:   make(chan *pb.Trade),
	}
	uc.bus.SubscribeAsync(topicBasicDataUpdated, false, uc.streamTrade)
	uc.bus.SubscribeAsync(topicStreamSubscribeFutureTick, false, uc.subscribeFutureTick)
	uc.bus.SubscribeAsync(topicStreamSubscribeFutureBidAsk, false, uc.subscribeFutureBidAsk)
	return uc
}

func (uc *tradeUseCase) streamTrade() {
	go uc.sendTrade()
	go uc.subscribeTrade()
}

func (uc *tradeUseCase) GetTrades(req *pb.QueryTradeRequest) ([]*pb.Trade, error) {
	if req.GetOrderId() != "" && req.GetStartTime().IsValid() && req.GetEndTime().IsValid() {
		return nil, errors.New("cannot specify both OrderId and time range")
	}
	return uc.tradeRepo.SelectTradesByRequest(context.Background(), req)
}

func (uc *tradeUseCase) GetUndoneTradesByCode(code string) ([]*pb.Trade, error) {
	return uc.tradeRepo.SelectUndoneTradesByCode(context.Background(), code)
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

func (uc *tradeUseCase) sendSingleFutureTick() chan *pb.Trade {
	ch := make(chan *pb.Trade)
	go func() {
		for {
			t := <-ch
			channels, ok := uc.SingleCodeClientMap.Load(t.GetCode())
			if !ok {
				continue
			}
			chs, _ := channels.([]chan *pb.Trade)
			for _, ch := range chs {
				ch <- t
			}
		}
	}()
	return ch
}

func (uc *tradeUseCase) sendTrade() {
	singleChannel := uc.sendSingleFutureTick()
	for {
		t := <-uc.tradeChannel
		singleChannel <- t
		uc.tradeClientLock.RLock()
		for _, client := range uc.tradeClientMap {
			client.Channel <- t
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
				uc.logger.Errorf("Failed to insert trade: %v(%+v)", err, t)
			}
		}()
	}
}

func (uc *tradeUseCase) subscribeFutureTick(detail *pb.FutureDetail) {
	fn := func(_ *pb.FutureTick) {
		// tickTime, _ := time.ParseInLocation(time.DateTime, tick.GetDateTime(), time.Local)
		// fmt.Printf("Received tick: %s, price: %f, volume: %d, time: %v, time gap: %d\n",
		// 	tick.GetCode(), tick.GetClose(), tick.GetVolume(), tickTime, time.Since(tickTime).Milliseconds())
	}
	uc.bus.Subscribe(fmt.Sprintf("%s/%s", topicStreamSubscribeFutureTick, detail.GetCode()), fn)
}

func (uc *tradeUseCase) subscribeFutureBidAsk(detail *pb.FutureDetail) {
	fn := func(_ *pb.FutureBidAsk) {
		// bidAskTime, _ := time.ParseInLocation(time.DateTime, bidAsk.GetDateTime(), time.Local)
		// fmt.Printf("Received bid-ask: %s, time: %v, time gap: %d\n", bidAsk.GetCode(), bidAskTime, time.Since(bidAskTime).Milliseconds())
	}
	uc.bus.Subscribe(fmt.Sprintf("%s/%s", topicStreamSubscribeFutureBidAsk, detail.GetCode()), fn)
}

type TradeClient struct {
	ID      string
	Channel chan *pb.Trade
}

func (uc *tradeUseCase) CreateTradeClient(client *TradeClient) {
	uc.tradeClientLock.Lock()
	uc.tradeClientMap[client.ID] = client
	uc.tradeClientLock.Unlock()
}

func (uc *tradeUseCase) CloseTradeClient(clientID string) {
	if clientID == "" {
		return
	}

	defer uc.tradeClientLock.Unlock()
	uc.tradeClientLock.Lock()

	if c, exist := uc.tradeClientMap[clientID]; exist {
		close(c.Channel)
		delete(uc.tradeClientMap, clientID)
	}
}

func (uc *tradeUseCase) CreateSingleCodeClient(code string, client *TradeClient) {
	channels, _ := uc.SingleCodeClientMap.LoadOrStore(code, []chan *pb.Trade{})
	chs, _ := channels.([]chan *pb.Trade)
	chs = append(chs, client.Channel)
	uc.SingleCodeClientMap.Store(code, chs)
}

func (uc *tradeUseCase) CloseSingleCodeClient(clientID, code string) {
	if clientID == "" || code == "" {
		return
	}

	channels, exist := uc.SingleCodeClientMap.Load(code)
	if !exist {
		return
	}

	chs, _ := channels.([]*TradeClient)
	for i, c := range chs {
		if c.ID == clientID {
			close(c.Channel)
			chs = append(chs[:i], chs[i+1:]...)
			break
		}
	}
	if len(chs) == 0 {
		uc.SingleCodeClientMap.Delete(code)
	} else {
		uc.SingleCodeClientMap.Store(code, chs)
	}
}

func (uc *tradeUseCase) BuyFuture(ctx context.Context, in *pb.OrderDetail) (*pb.Trade, error) {
	return uc.tradeClient.BuyFuture(ctx, in)
}

func (uc *tradeUseCase) SellFuture(ctx context.Context, in *pb.OrderDetail) (*pb.Trade, error) {
	return uc.tradeClient.SellFuture(ctx, in)
}

func (uc *tradeUseCase) CancelTrade(ctx context.Context, in *pb.Trade) (*pb.Trade, error) {
	return uc.tradeClient.CancelTrade(ctx, in)
}

func (uc *tradeUseCase) GetFuturePosition(ctx context.Context) (*pb.FuturePositionList, error) {
	return uc.tradeClient.GetFuturePosition(ctx, &emptypb.Empty{})
}

func (uc *tradeUseCase) GetFuturePositionByCode(ctx context.Context, code string) (*pb.FuturePositionList, error) {
	positionList, err := uc.tradeClient.GetFuturePosition(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	filterd := make([]*pb.FuturePosition, 0, len(positionList.GetList()))
	for _, pos := range positionList.GetList() {
		if pos.GetCode() == code {
			filterd = append(filterd, pos)
		}
	}
	positionList.List = filterd
	return positionList, nil
}

func (uc *tradeUseCase) GetMargin(ctx context.Context) (*pb.Margin, error) {
	return uc.tradeClient.GetMargin(ctx, &emptypb.Empty{})
}
