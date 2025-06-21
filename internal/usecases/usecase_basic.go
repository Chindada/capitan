package usecases

import (
	"context"

	"github.com/chindada/capitan/internal/config"
	"github.com/chindada/capitan/internal/usecases/repo"
	"github.com/chindada/leopard/pkg/eventbus"
	"github.com/chindada/leopard/pkg/log"
	"github.com/chindada/panther/golang/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

//go:generate mockgen -source=usecase_basic.go -destination=./mocks/mocks_usecase_basic_test.go -package=mocks

type Basic interface {
	GetAllStockDetail(ctx context.Context) (*pb.StockDetailList, error)
}

type basicUseCase struct {
	basicRepo repo.BasicRepo

	logger *log.Log
	bus    *eventbus.Bus

	basicClient pb.BasicInterfaceClient
}

func NewBasic() Basic {
	cfg := config.Get()
	pg := cfg.GetPostgresPool()
	uc := &basicUseCase{
		basicRepo:   repo.NewBasic(pg),
		logger:      log.Get(),
		bus:         eventbus.Get(),
		basicClient: pb.NewBasicInterfaceClient(cfg.GetGRPCConn()),
	}

	routines := []func() error{
		uc.updateStock,
		uc.updateFuture,
		uc.updateOption,
	}
	for _, routine := range routines {
		if err := routine(); err != nil {
			uc.logger.Fatalf("Failed to update data: %v", err)
		}
	}
	// uc.healthCheck()
	return uc
}

// func (uc *basicUseCase) healthCheck() {
// 	stream, err := uc.basicClient.HealthChannel(context.Background())
// 	if err != nil {
// 		uc.logger.Fatalf("Failed to connect to BasicDataInterface: %v", err)
// 	}
// 	go func() {
// 		for {
// 			_, err = stream.Recv()
// 			if err != nil {
// 				uc.logger.Fatalf("Health check stream error: %v", err)
// 			}
// 		}
// 	}()
// }

func (uc *basicUseCase) GetAllStockDetail(ctx context.Context) (*pb.StockDetailList, error) {
	return uc.basicClient.GetAllStockDetail(ctx, &emptypb.Empty{})
}

func (uc *basicUseCase) updateStock() error {
	stocks, err := uc.GetAllStockDetail(context.Background())
	if err != nil {
		return err
	}
	if len(stocks.GetList()) <= 1000 {
		err = uc.basicRepo.InsertStockDetail(context.Background(), stocks.GetList())
		if err != nil {
			return err
		}
	}
	spilts := [][]*pb.StockDetail{}
	for i := 0; i < len(stocks.GetList()); i += 1000 {
		end := min(i+1000, len(stocks.GetList()))
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

func (uc *basicUseCase) updateFuture() error {
	futures, err := uc.basicClient.GetAllFutureDetail(context.Background(), &emptypb.Empty{})
	if err != nil {
		return err
	}
	if len(futures.GetList()) <= 1000 {
		err = uc.basicRepo.InsertFutureDetail(context.Background(), futures.GetList())
		if err != nil {
			return err
		}
	}
	splits := [][]*pb.FutureDetail{}
	for i := 0; i < len(futures.GetList()); i += 1000 {
		end := min(i+1000, len(futures.GetList()))
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

func (uc *basicUseCase) updateOption() error {
	options, err := uc.basicClient.GetAllOptionDetail(context.Background(), &emptypb.Empty{})
	if err != nil {
		return err
	}
	if len(options.GetList()) <= 1000 {
		err = uc.basicRepo.InsertOptionDetail(context.Background(), options.GetList())
		if err != nil {
			return err
		}
	}
	splits := [][]*pb.OptionDetail{}
	for i := 0; i < len(options.GetList()); i += 1000 {
		end := min(i+1000, len(options.GetList()))
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
