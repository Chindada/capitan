package capitan

import (
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/chindada/capitan/internal/config"
	"github.com/chindada/capitan/internal/controller/http/router"
	"github.com/chindada/capitan/internal/usecases"
	"github.com/chindada/leopard/pkg/httpserver"
	"github.com/chindada/leopard/pkg/log"
)

func Start() {
	logger := log.Get()
	cfg := config.Get()

	// Pre process, do not adjust the order, except for new feature
	ucSystem := usecases.NewSystem()
	ucStream := usecases.NewStream()
	ucBasic := usecases.NewBasic()

	// HTTP Handler
	r := router.NewRouter(ucSystem).
		AddV1BasicRoutes(ucBasic).
		AddV1StreamRoutes(ucStream).
		AddV1SystemRoutes()

	// Start HTTP Server
	if e := httpserver.New(
		r.GetHandler(),
		httpserver.Port(cfg.Server.SRVPort),
		httpserver.AddLogger(logger),
	).Start(); e != nil {
		logger.Fatalf("API Server error: %s", e)
	}

	defer func() {
		cfg.CloseDB()
		tryStopProxyServer()
		logger.Info("Shut down")
	}()

	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	<-interrupt
}

// tryStopProxyServer ROOT_PATH should be set only under docker environment.
func tryStopProxyServer() {
	rootPath := os.Getenv("ROOT_PATH")
	if rootPath == "" {
		return
	}
	proxyPID, err := os.ReadFile(filepath.Join(rootPath, "proxy", "proxy.pid"))
	if err != nil {
		return
	}
	proxyPIDInt, err := strconv.Atoi(strings.ReplaceAll(string(proxyPID), "\n", ""))
	if err != nil {
		return
	}
	p, e := os.FindProcess(proxyPIDInt)
	if e != nil {
		return
	}
	e = p.Signal(syscall.SIGQUIT)
	if e != nil {
		return
	}
}
