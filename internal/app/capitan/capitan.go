package capitan

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/chindada/capitan/internal/config"
	"github.com/chindada/capitan/internal/controller/http/router"
	"github.com/chindada/capitan/internal/usecases"
	"github.com/chindada/leopard/pkg/httpserver"
	"github.com/chindada/leopard/pkg/log"
)

func Start() {
	// Waiting signal
	exit := make(chan os.Signal, 1)

	logger := log.Get()
	cfg := config.Get()

	// Connect gRPC server
	cfg.ConnectGRPC(exit)

	// Pre process, do not adjust the order, except for new feature
	ucEvents := usecases.NewEvents()
	ucTrade := usecases.NewTrade()
	ucStream := usecases.NewStream()
	ucBasic := usecases.NewBasic()
	ucSystem := usecases.NewSystem()

	// HTTP Handler
	r := router.NewRouter(ucSystem).
		AddV1SystemRoutes(ucSystem).
		AddV1BasicRoutes(ucBasic).
		AddV1TradeRoutes(ucTrade).
		AddV1EventsRoutes(ucEvents).
		AddV1StreamRoutes(ucStream)

	// Start HTTP Server
	if e := httpserver.New(
		r.GetHandler(),
		httpserver.Port(cfg.Server.SRVPort),
		httpserver.AddLogger(logger),
	).Start(); e != nil {
		logger.Fatalf("API Server error: %s", e)
	}

	cfg.StartProxy()
	defer func() {
		cfg.StopProxy()
		cfg.CloseDB()
		logger.Info("Shut down")
	}()

	signal.Notify(exit, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	<-exit
}
