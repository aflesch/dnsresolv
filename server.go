package dnsresolv

import (
	"errors"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/miekg/dns"
)

// Server type
type Server struct {
	host     string
	rTimeout time.Duration
	wTimeout time.Duration
}

// Run starts the server
func (s *Server) run(logger log15.Logger, nameservers []string) error {
	if len(nameservers) == 0 {
		return errors.New("Empty nameservers list")
	}
	Handler := NewHandler(logger, nameservers)

	tcpHandler := dns.NewServeMux()
	tcpHandler.HandleFunc(".", Handler.DoTCP)

	udpHandler := dns.NewServeMux()
	udpHandler.HandleFunc(".", Handler.DoUDP)

	tcpServer := &dns.Server{Addr: s.host,
		Net:          "tcp",
		Handler:      tcpHandler,
		ReadTimeout:  s.rTimeout,
		WriteTimeout: s.wTimeout}

	udpServer := &dns.Server{Addr: s.host,
		Net:          "udp",
		Handler:      udpHandler,
		UDPSize:      65535,
		ReadTimeout:  s.rTimeout,
		WriteTimeout: s.wTimeout}

	go s.start(logger, udpServer)
	go s.start(logger, tcpServer)
	return nil
}

func (s *Server) start(logger log15.Logger, ds *dns.Server) {
	logger.Info("start listener on", "net", ds.Net, "addr", s.host)

	if err := ds.ListenAndServe(); err != nil {
		logger.Error("start listener on failed", "net", ds.Net, "addr", s.host, "error", err)
	}
}
