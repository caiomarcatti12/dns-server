package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net"
	"time"

	"github.com/miekg/dns"
	"gopkg.in/yaml.v2"
)

type DNSConfig struct {
	Domain     string `yaml:"domain"`
	Interface  string `yaml:"interface"`
	ResponseIP string `yaml:"response_ip"`
}

type Config struct {
	DNSConfigs []DNSConfig `yaml:"dns_configs"`
	DNSServers []string    `yaml:"dns_servers"`
}

func loadConfig(filename string) (Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

func getResponseIP(dnsConfigs []DNSConfig, domain, iface string) (string, bool) {
	for _, cfg := range dnsConfigs {
		if cfg.Domain == domain && cfg.Interface == iface {
			return cfg.ResponseIP, true
		}
	}
	return "", false
}

func queryExternalDNS(domain string, servers []string) (*dns.Msg, error) {
	m := new(dns.Msg)
	m.SetQuestion(domain, dns.TypeA)
	c := new(dns.Client)
	c.Timeout = 3 * time.Second

	for _, server := range servers {
		resp, _, err := c.Exchange(m, server)
		if err == nil && len(resp.Answer) > 0 {
			return resp, nil
		}
	}

	return nil, errors.New("no external DNS response obtained")
}

func handleDNSRequest(config Config) dns.HandlerFunc {
	return func(w dns.ResponseWriter, req *dns.Msg) {
		msg := dns.Msg{}
		msg.SetReply(req)

		question := req.Question[0]

		localAddr := w.LocalAddr().String()
		serverIP, _, err := net.SplitHostPort(localAddr)
		if err != nil {
			log.Printf("Erro ao obter IP local: %v", err)
			return
		}

		responseIP, exists := getResponseIP(config.DNSConfigs, question.Name, serverIP)
		if exists {
			log.Printf("Received query for %s from interface %s, responding with IP %s\n", question.Name, serverIP, responseIP)
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: question.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
				A:   net.ParseIP(responseIP),
			})
		} else {
			log.Printf("Query not found locally for %s, querying external DNS\n", question.Name)
			extResp, err := queryExternalDNS(question.Name, config.DNSServers)
			if err == nil {
				msg.Answer = extResp.Answer
				log.Printf("External DNS response obtained for %s", question.Name)
			} else {
				log.Printf("Failed to get external DNS response for %s: %v", question.Name, err)
			}
		}

		w.WriteMsg(&msg)
	}
}

func main() {
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Falha ao carregar configuração: %v", err)
	}

	dns.HandleFunc(".", handleDNSRequest(config))

	interfaces := make(map[string]bool)
	for _, cfg := range config.DNSConfigs {
		interfaces[cfg.Interface] = true
	}

	for iface := range interfaces {
		go func(ip string) {
			server := &dns.Server{Addr: ip + ":53", Net: "udp"}
			log.Printf("Servidor DNS escutando em %s", server.Addr)
			if err := server.ListenAndServe(); err != nil {
				log.Fatalf("Falha ao iniciar servidor DNS: %v", err)
			}
		}(iface)
	}

	select {}
}
