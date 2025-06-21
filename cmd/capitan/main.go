package main

import (
	"github.com/chindada/capitan/internal/app/capitan"
	"github.com/chindada/capitan/internal/config"
	"github.com/chindada/leopard/pkg/log"
)

func main() {
	// Init log
	log.Init()

	// Init config
	config.Init()

	// Start app
	capitan.Start()
}
