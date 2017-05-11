package dnsresolv

import (
	"os"
	"os/signal"
	"time"

	"github.com/inconshreveable/log15"
)

type Config struct {
	Bind        string
	Nameservers []string
}

func Start(logger log15.Logger, config Config) error {
	server := &Server{
		host:     config.Bind,
		rTimeout: 5 * time.Second,
		wTimeout: 5 * time.Second,
	}

	if err := server.run(logger, config.Nameservers); err != nil {
		return err
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

forever:
	for {
		select {
		case <-sig:
			logger.Debug("signal received, stopping")
			break forever
		}
	}
	return nil
}
