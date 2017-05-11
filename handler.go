package dnsresolv

import (
	"log"
	"net"

	"github.com/inconshreveable/log15"
	"github.com/miekg/dns"
)

const (
	notIPQuery = 0
	_IP4Query  = 4
	_IP6Query  = 6
)

// Question type
type Question struct {
	Qname  string `json:"name"`
	Qtype  string `json:"type"`
	Qclass string `json:"class"`
}

// QuestionCacheEntry represents a full query from a client with metadata
type QuestionCacheEntry struct {
	Date    int64    `json:"date"`
	Remote  string   `json:"client"`
	Blocked bool     `json:"blocked"`
	Query   Question `json:"query"`
}

// String formats a question
func (q *Question) String() string {
	return q.Qname + " " + q.Qclass + " " + q.Qtype
}

// DNSHandler type
type DNSHandler struct {
	resolver *Resolver
	logger   log15.Logger
}

// NewHandler returns a new DNSHandler
func NewHandler(logger log15.Logger, nameservers []string) *DNSHandler {
	var (
		clientConfig *dns.ClientConfig
		resolver     *Resolver
	)

	resolver = &Resolver{clientConfig, nameservers, logger}

	return &DNSHandler{resolver, logger}
}

func (h *DNSHandler) do(Net string, w dns.ResponseWriter, req *dns.Msg) {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	defer w.Close()
	q := req.Question[0]
	Q := Question{UnFqdn(q.Name), dns.TypeToString[q.Qtype], dns.ClassToString[q.Qclass]}

	var remote net.IP
	if Net == "tcp" {
		remote = w.RemoteAddr().(*net.TCPAddr).IP
	} else {
		remote = w.RemoteAddr().(*net.UDPAddr).IP
	}

	h.logger.Debug("lookup", "remote", remote, "question", Q.String())

	mesg, err := h.resolver.lookup(Net, req)
	if err != nil {
		h.logger.Error("resolve query error", "error", err)
		h.HandleFailed(w, req)
		return
	}

	h.logger.Debug("lookup", "msg", mesg, "1", mesg.Answer[0], "2", mesg.Answer[1])
	if mesg.Truncated && Net == "udp" {
		mesg, err = h.resolver.lookup("tcp", req)
		if err != nil {
			h.logger.Error("resolve tcp query error", "error", err)
			h.HandleFailed(w, req)
			return
		}
	}

	h.WriteReplyMsg(w, mesg)
}

// DoTCP begins a tcp query
func (h *DNSHandler) DoTCP(w dns.ResponseWriter, req *dns.Msg) {
	go h.do("tcp", w, req)
}

// DoUDP begins a udp query
func (h *DNSHandler) DoUDP(w dns.ResponseWriter, req *dns.Msg) {
	go h.do("udp", w, req)
}

func (h *DNSHandler) HandleFailed(w dns.ResponseWriter, message *dns.Msg) {
	m := new(dns.Msg)
	m.SetRcode(message, dns.RcodeServerFailure)
	h.WriteReplyMsg(w, m)
}

func (h *DNSHandler) WriteReplyMsg(w dns.ResponseWriter, message *dns.Msg) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Debug("Recovered in WriteReplyMsg", "recover", r)
		}
	}()

	err := w.WriteMsg(message)
	if err != nil {
		log.Println(err)
	}
}

func (h *DNSHandler) isIPQuery(q dns.Question) int {
	if q.Qclass != dns.ClassINET {
		return notIPQuery
	}

	switch q.Qtype {
	case dns.TypeA:
		return _IP4Query
	case dns.TypeAAAA:
		return _IP6Query
	default:
		return notIPQuery
	}
}

// UnFqdn function
func UnFqdn(s string) string {
	if dns.IsFqdn(s) {
		return s[:len(s)-1]
	}
	return s
}
