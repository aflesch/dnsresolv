package main

import (
	"bufio"
	"os"
	"strings"

	"github.com/aflesch/dnsresolv"
	"github.com/inconshreveable/log15"
)

const (
	bindHost         = "0.0.0.0:8053"
	resolvConfigFile = "/etc/resolv.conf"
)

//func init() {
//	topicfilter.Set("dnsresolv", log15.LvlInfo)
//}

func main() {
	// Create context and topic
	logger := log15.New("topic", "dnsresolver")

	// Create Config
	nameservers, err := parseResolvConfigFile(resolvConfigFile)
	if err != nil {
		logger.Error("Parse resolv config file", "error", err)
	}

	err = dnsresolv.Start(logger, dnsresolv.Config{Bind: bindHost, Nameservers: nameservers})
	logger.Warn("dnsresolv Done", "error", err)
}

func parseResolvConfigFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var nameservers []string
	for scanner.Scan() {
		// check for #
		if scanner.Bytes()[0] == '#' {
			continue
		}
		splits := strings.Split(scanner.Text(), " ")
		if splits[0] == "nameserver" {
			log15.Debug("Nameserver", "name", splits[1])
			nameservers = append(nameservers, splits[1]+":53")
		}
	}
	err = scanner.Err()
	return nameservers, err
}
