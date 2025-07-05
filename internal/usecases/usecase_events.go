package usecases

import (
	"context"
	"sync"
	"time"

	"github.com/chindada/capitan/internal/config"
	"github.com/chindada/capitan/internal/usecases/repo"
	"github.com/chindada/leopard/pkg/eventbus"
	"github.com/chindada/leopard/pkg/log"
	"github.com/chindada/panther/golang/pb"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

//go:generate mockgen -source=usecase_events.go -destination=./mocks/mocks_usecase_events_test.go -package=mocks

type Events interface {
	GetShioajiEvent() ([]*pb.ShioajiEvent, error)
	GetCurrentLoginEvent() *pb.LoginEventList
}

type eventsUseCase struct {
	eventRepo repo.EventRepo

	launchTime time.Time

	eventLoginChan       chan *pb.LoginEvent
	cachedEventLogin     []*pb.LoginEvent
	cachedEventLoginLock sync.RWMutex

	logger *log.Log
	bus    *eventbus.Bus

	streamClient pb.StreamInterfaceClient
}

func NewEvents() Events {
	cfg := config.Get()
	pg := cfg.GetPostgresPool()
	uc := &eventsUseCase{
		eventRepo:      repo.NewEventRepo(pg),
		eventLoginChan: make(chan *pb.LoginEvent),
		logger:         log.Get(),
		bus:            eventbus.Get(),
		launchTime:     time.Now(),
		streamClient:   pb.NewStreamInterfaceClient(cfg.GetGRPCConn()),
	}

	go uc.loginEventSaver()
	go uc.subscribeShioajiEvent()

	uc.bus.Subscribe(topicLogin, uc.processLogin)
	return uc
}

const cacheEventSize int64 = 10

func (uc *eventsUseCase) processLogin(event *pb.LoginEvent) {
	uc.eventLoginChan <- event
}

func (uc *eventsUseCase) loginEventSaver() {
	uc.fillCachedLogin()
	ticker := time.NewTicker(time.Minute)
	events := []*pb.LoginEvent{}
	for {
		select {
		case event := <-uc.eventLoginChan:
			events = append(events, event)
			uc.cachedEventLoginLock.Lock()
			uc.cachedEventLogin = append([]*pb.LoginEvent{event}, uc.cachedEventLogin...)
			if len(uc.cachedEventLogin) >= int(cacheEventSize) {
				uc.cachedEventLogin = uc.cachedEventLogin[:cacheEventSize]
			}
			uc.cachedEventLoginLock.Unlock()

		case <-ticker.C:
			if len(events) > 0 {
				_ = uc.eventRepo.InsertLoginEvent(context.Background(), events)
				events = []*pb.LoginEvent{}
			}
		}
	}
}

func (uc *eventsUseCase) fillCachedLogin() {
	events, err := uc.eventRepo.SelectLoginEvent(context.Background(), cacheEventSize)
	if err != nil {
		return
	}

	uc.cachedEventLoginLock.Lock()
	uc.cachedEventLogin = events
	uc.cachedEventLoginLock.Unlock()
}

func (uc *eventsUseCase) GetCurrentLoginEvent() *pb.LoginEventList {
	uc.cachedEventLoginLock.RLock()
	defer uc.cachedEventLoginLock.RUnlock()

	return &pb.LoginEventList{
		List: uc.cachedEventLogin,
	}
}

func (uc *eventsUseCase) subscribeShioajiEvent() {
	eventStream, err := uc.streamClient.SubscribeShioajiEvent(context.Background(), &emptypb.Empty{})
	if err != nil {
		s := status.Convert(err)
		uc.logger.Fatalf("subscribeShioajiEvent error(%d): %s", s.Code(), s.Message())
	}
	for {
		event, rErr := eventStream.Recv()
		if rErr != nil {
			return
		}
		err = uc.eventRepo.InsertShioajiEvent(context.Background(), event)
		if err != nil {
			uc.logger.Errorf("Failed to insert shioaji event: %v", err)
			continue
		}
	}
}

func (uc *eventsUseCase) GetShioajiEvent() ([]*pb.ShioajiEvent, error) {
	return uc.eventRepo.SelectShioajiEvent(context.Background())
}
