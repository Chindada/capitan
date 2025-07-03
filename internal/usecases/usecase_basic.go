package usecases

import (
	"context"
	"sync"
	"time"

	"github.com/chindada/capitan/internal/config"
	"github.com/chindada/capitan/internal/usecases/entity"
	"github.com/chindada/capitan/internal/usecases/modules/calendar"
	"github.com/chindada/capitan/internal/usecases/repo"
	"github.com/chindada/leopard/pkg/eventbus"
	"github.com/chindada/leopard/pkg/log"
	"github.com/chindada/panther/golang/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

//go:generate mockgen -source=usecase_basic.go -destination=./mocks/mocks_usecase_basic_test.go -package=mocks

var mainFutures = []string{"TXF", "MXF", "TMF"}

const (
	batchSize = 2000
)

type Basic interface {
	GetAllStockDetail(ctx context.Context) ([]*pb.StockDetail, error)
	GetAllFutureDetail(ctx context.Context) ([]*pb.FutureDetail, error)
	GetAllOptionDetail(ctx context.Context) ([]*pb.OptionDetail, error)

	GetFutureKbar(ctx context.Context, req *pb.HistoryKbarRequest) (*pb.HistoryKbarList, error)

	GetTargetStock() []*pb.StockDetail
	GetTargetFuture() []*pb.FutureDetail
}

type basicUseCase struct {
	calendar  calendar.Calendar
	basicRepo repo.BasicRepo

	logger *log.Log
	bus    *eventbus.Bus

	basicClient pb.BasicInterfaceClient

	targetFuture []*pb.FutureDetail
	targetStock  []*pb.StockDetail
	tragetLock   sync.RWMutex
}

func NewBasic() Basic {
	cfg := config.Get()
	pg := cfg.GetPostgresPool()
	uc := &basicUseCase{
		calendar:    calendar.NewCalendar(),
		basicRepo:   repo.NewBasic(pg),
		logger:      log.Get(),
		bus:         eventbus.Get(),
		basicClient: pb.NewBasicInterfaceClient(cfg.GetGRPCConn()),
	}
	updaters := []func() error{
		uc.updateStock,
		uc.updateFuture,
		uc.updateOption,
		uc.fillTargetStock,
		uc.fillClosetFutures,
	}
	for _, updater := range updaters {
		if err := updater(); err != nil {
			uc.logger.Fatalf("Failed to update data: %v", err)
		}
	}
	uc.bus.PublishTopicEvent(topicBasicDataUpdated)
	return uc
}

func (uc *basicUseCase) updateStock() error {
	stocks, err := uc.basicClient.GetAllStockDetail(context.Background(), &emptypb.Empty{})
	if err != nil {
		return err
	}
	if len(stocks.GetList()) <= batchSize {
		err = uc.basicRepo.InsertStockDetail(context.Background(), stocks.GetList())
		if err != nil {
			return err
		}
	}
	spilts := [][]*pb.StockDetail{}
	for i := 0; i < len(stocks.GetList()); i += batchSize {
		end := min(i+batchSize, len(stocks.GetList()))
		spilts = append(spilts, stocks.GetList()[i:end])
	}
	for _, split := range spilts {
		err = uc.basicRepo.InsertStockDetail(context.Background(), split)
		if err != nil {
			return err
		}
	}
	return nil
}

func (uc *basicUseCase) fillTargetStock() error {
	volumeRank, err := uc.basicClient.GetStockVolumeRank(context.Background(), &pb.VolumeRankRequest{
		Date: uc.calendar.GetStockLastTradeDay().ToDateOnlyString(),
	})
	if err != nil {
		return err
	}
	if len(volumeRank.GetList()) == 0 {
		uc.logger.Warnf("No stock volume rank found for date %s", uc.calendar.GetStockLastTradeDay().ToDateOnlyString())
		return nil
	}

	uc.tragetLock.Lock()
	defer uc.tragetLock.Unlock()

	for _, rank := range volumeRank.GetList() {
		stockDetail, sErr := uc.basicRepo.SelectStockDetailByCode(context.Background(), rank.GetCode())
		if sErr != nil {
			continue
		}
		uc.bus.PublishTopicEvent(topicStreamSubscribeStockTick, stockDetail)
		uc.targetStock = append(uc.targetStock, stockDetail)
	}
	return nil
}

func (uc *basicUseCase) updateFuture() error {
	futures, err := uc.basicClient.GetAllFutureDetail(context.Background(), &emptypb.Empty{})
	if err != nil {
		return err
	}
	if len(futures.GetList()) <= batchSize {
		err = uc.basicRepo.InsertFutureDetail(context.Background(), futures.GetList())
		if err != nil {
			return err
		}
	}
	splits := [][]*pb.FutureDetail{}
	for i := 0; i < len(futures.GetList()); i += batchSize {
		end := min(i+batchSize, len(futures.GetList()))
		splits = append(splits, futures.GetList()[i:end])
	}
	for _, split := range splits {
		err = uc.basicRepo.InsertFutureDetail(context.Background(), split)
		if err != nil {
			return err
		}
	}
	return nil
}

func (uc *basicUseCase) fillClosetFutures() error {
	uc.tragetLock.Lock()
	defer uc.tragetLock.Unlock()

	for _, future := range mainFutures {
		result, err := uc.basicRepo.SearchFutureDetail(context.Background(), future)
		if err != nil {
			return err
		}
		if len(result) == 0 {
			uc.logger.Errorf("No future detail found for %s", future)
			continue
		}
		for _, detail := range result {
			dTime, pErr := time.ParseInLocation(entity.ShortSlashTimeLayout, detail.GetDeliveryDate(), time.Local)
			if pErr != nil {
				uc.logger.Errorf("Failed to parse delivery date for %s: %v", detail.GetCode(), pErr)
				continue
			}
			if time.Now().Before(dTime) {
				uc.bus.PublishTopicEvent(topicStreamSubscribeFutureTick, detail)
				uc.bus.PublishTopicEvent(topicStreamSubscribeFutureBidAsk, detail)
				uc.targetFuture = append(uc.targetFuture, detail)
				uc.logger.Infof("Found closest future: %s, delivery date: %s", detail.GetCode(), detail.GetDeliveryDate())
				break
			}
		}
	}
	return nil
}

func (uc *basicUseCase) updateOption() error {
	options, err := uc.basicClient.GetAllOptionDetail(context.Background(), &emptypb.Empty{})
	if err != nil {
		return err
	}
	if len(options.GetList()) <= batchSize {
		err = uc.basicRepo.InsertOptionDetail(context.Background(), options.GetList())
		if err != nil {
			return err
		}
	}
	splits := [][]*pb.OptionDetail{}
	for i := 0; i < len(options.GetList()); i += batchSize {
		end := min(i+batchSize, len(options.GetList()))
		splits = append(splits, options.GetList()[i:end])
	}
	for _, split := range splits {
		err = uc.basicRepo.InsertOptionDetail(context.Background(), split)
		if err != nil {
			return err
		}
	}
	return nil
}

func (uc *basicUseCase) GetAllStockDetail(ctx context.Context) ([]*pb.StockDetail, error) {
	return uc.basicRepo.SelectAllStockDetail(ctx)
}

func (uc *basicUseCase) GetAllFutureDetail(ctx context.Context) ([]*pb.FutureDetail, error) {
	return uc.basicRepo.SelectAllFutureDetail(ctx)
}

func (uc *basicUseCase) GetAllOptionDetail(ctx context.Context) ([]*pb.OptionDetail, error) {
	return uc.basicRepo.SelectAllOptionDetail(ctx)
}

func (uc *basicUseCase) GetFutureKbar(ctx context.Context, req *pb.HistoryKbarRequest) (*pb.HistoryKbarList, error) {
	return uc.basicClient.GetFutureHistoryKbar(ctx, req)
}

func (uc *basicUseCase) GetTargetStock() []*pb.StockDetail {
	uc.tragetLock.RLock()
	defer uc.tragetLock.RUnlock()

	return uc.targetStock
}

func (uc *basicUseCase) GetTargetFuture() []*pb.FutureDetail {
	uc.tragetLock.RLock()
	defer uc.tragetLock.RUnlock()

	return uc.targetFuture
}
