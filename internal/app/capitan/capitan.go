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
	"github.com/go-co-op/gocron/v2"
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
	setupStopJob(interrupt)
	<-interrupt
}

func setupStopJob(interrupt chan os.Signal) {
	exit := func() {
		interrupt <- os.Interrupt
	}
	stopSchedule := []string{
		"20 8 * * *",
		"40 14 * * *",
	}
	s, err := gocron.NewScheduler()
	if err != nil {
		log.Get().Fatal(err)
	}
	for _, schedule := range stopSchedule {
		_, err = s.NewJob(
			gocron.CronJob(schedule, false),
			gocron.NewTask(exit),
		)
		if err != nil {
			log.Get().Fatalf("init scheduler error: %v", err)
		}
	}
	go s.Start()
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
