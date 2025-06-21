package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/chindada/capitan/internal/config"
	"github.com/chindada/leopard/pkg/eventbus"
	"github.com/chindada/leopard/pkg/log"
	"github.com/chindada/panther/golang/pb"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

//go:generate mockgen -source=usecase_stream.go -destination=./mocks/mocks_usecase_stream_test.go -package=mocks

type Stream interface{}

type streamUseCase struct {
	logger *log.Log
	bus    *eventbus.Bus

	streamClient pb.StreamInterfaceClient
}

func NewStream() Stream {
	cfg := config.Get()
	uc := &streamUseCase{
		logger:       log.Get(),
		bus:          eventbus.Get(),
		streamClient: pb.NewStreamInterfaceClient(cfg.GetGRPCConn()),
	}
	go uc.subscribeShioajiEvent()
	codes := []string{"TXFG5", "MXFG5", "TMFG5"}
	for _, code := range codes {
		go func(c string) {
			_ = uc.subscribeFutureTick(c)
		}(code)
	}
	return uc
}

func (uc *streamUseCase) subscribeShioajiEvent() {
	eventStream, err := uc.streamClient.SubscribeShioajiEvent(context.Background(), &emptypb.Empty{})
	if err != nil {
		s := status.Convert(err)
		uc.logger.Fatalf("Error(%d): %s", s.Code(), s.Message())
	}
	for {
		event, rErr := eventStream.Recv()
		if rErr != nil {
			s := status.Convert(rErr)
			uc.logger.Fatalf("Error(%d): %s", s.Code(), s.Message())
		}
		uc.logger.Warnf("Resp code: %d, Event code: %d, Info: %s, Event: %s",
			event.GetRespCode(), event.GetEventCode(), event.GetInfo(), event.GetEvent())
	}
}

func (uc *streamUseCase) subscribeFutureTick(code string) error {
	tickStream, err := uc.streamClient.SubscribeFutureTick(context.Background(), &pb.SubscribeFutureRequest{
		Code: code,
	})
	if err != nil {
		return err
	}
	for {
		tick, rErr := tickStream.Recv()
		if rErr != nil {
			return rErr
		}
		tickTime, _ := time.ParseInLocation(time.DateTime, tick.GetDateTime(), time.Local)
		fmt.Printf("Received tick: %s, price: %f, volume: %d, time: %v,time gap: %d\n",
			tick.GetCode(), tick.GetClose(), tick.GetVolume(), tickTime, time.Since(tickTime).Milliseconds())
	}
}

// func (uc *streamUseCase) subscribeFutureBidAsk(code string) error {
// 	bidAskStream, err := uc.streamClient.SubscribeFutureBidAsk(context.Background(), &pb.SubscribeFutureRequest{
// 		Code: code,
// 	})
// 	if err != nil {
// 		return err
// 	}
// 	for {
// 		bidAsk, err := bidAskStream.Recv()
// 		if err != nil {
// 			return err
// 		}
// 		fmt.Printf("%v\n", bidAsk)
// 	}
// }
