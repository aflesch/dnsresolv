package dnsresolv

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/miekg/dns"
)

const (
	concurrencyInterval = 200 //concurrency interval for lookups in miliseconds
	queryTimeout        = 5   //query timeout for dns lookups in seconds
)

// ResolvError type
type ResolvError struct {
	qname, net  string
	nameservers []string
}

// Error formats a ResolvError
func (e ResolvError) Error() string {
	errmsg := fmt.Sprintf("%s resolv failed on %s (%s)", e.qname, strings.Join(e.nameservers, "; "), e.net)
	return errmsg
}

// Resolver type
type Resolver struct {
	config      *dns.ClientConfig
	nameservers []string
	logger      log15.Logger
}

// Lookup will ask each nameserver in top-to-bottom fashion, starting a new request
// in every second, and return as early as possbile (have an answer).
// It returns an error if no request has succeeded.
func (r *Resolver) lookup(net string, req *dns.Msg) (message *dns.Msg, err error) {
	c := &dns.Client{
		Net:          net,
		ReadTimeout:  r.timeout(),
		WriteTimeout: r.timeout(),
	}

	qname := req.Question[0].Name
	res := make(chan *dns.Msg, 1)
	logger := r.logger
	var wg sync.WaitGroup
	L := func(nameserver string) {
		defer wg.Done()
		r, _, err := c.Exchange(req, nameserver)
		if err != nil {
			logger.Error("socket error on", "qname", qname, "nameserver", nameserver, "err", err)
			return
		}
		if r != nil && r.Rcode != dns.RcodeSuccess {
			logger.Debug("failed to get an valid answer", "name", qname, "nameserver", nameserver)
			if r.Rcode == dns.RcodeServerFailure {
				return
			}
		} else {
			logger.Debug("resolv", "name", UnFqdn(qname), "nameserver", nameserver, "net", net)
		}
		select {
		case res <- r:
		default:
		}
	}

	ticker := time.NewTicker(time.Duration(concurrencyInterval) * time.Millisecond)
	defer ticker.Stop()

	// Start lookup on each nameserver top-down, in every second
	for _, nameserver := range r.nameservers {
		wg.Add(1)
		go L(nameserver)
		// but exit early, if we have an answer
		select {
		case r := <-res:
			return r, nil
		case <-ticker.C:
			continue
		}
	}

	// wait for all the namservers to finish
	wg.Wait()
	select {
	case r := <-res:
		return r, nil
	default:
		return nil, ResolvError{qname, net, r.nameservers}
	}
}

// Timeout returns the resolver timeout
func (r *Resolver) timeout() time.Duration {
	return time.Duration(queryTimeout) * time.Second
}
