package usecases

import (
	"context"
	"fmt"

	"github.com/chindada/capitan/internal/config"
	"github.com/chindada/leopard/pkg/eventbus"
	"github.com/chindada/leopard/pkg/log"
	"github.com/chindada/panther/golang/pb"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

//go:generate mockgen -source=usecase_stream.go -destination=./mocks/mocks_usecase_stream_test.go -package=mocks

type Stream interface {
	StreamMock()
}

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

	uc.bus.SubscribeAsync(topicStreamSubscribeFutureTick, false, uc.subscribeFutureTick)
	uc.bus.SubscribeAsync(topicStreamSubscribeFutureBidAsk, false, uc.subscribeFutureBidAsk)

	return uc
}

func (uc *streamUseCase) StreamMock() {}

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

func (uc *streamUseCase) subscribeFutureTick(code string) {
	tickStream, err := uc.streamClient.SubscribeFutureTick(context.Background(), &pb.SubscribeFutureRequest{
		Code: code,
	})
	if err != nil {
		uc.logger.Errorf("Failed to subscribe to future tick for code %s: %v", code, err)
		return
	}
	for {
		t, rErr := tickStream.Recv()
		if rErr != nil {
			return
		}
		uc.bus.PublishTopicEvent(fmt.Sprintf("%s/%s", topicStreamSubscribeFutureTick, code), t)
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
	}
}
