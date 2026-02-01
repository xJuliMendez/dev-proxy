package main

import (
	"context"
	"io"
	"log"
	"net/netip"
	"os"
	"strings"

	"codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/rdata"
)

var UPSTREAM = "1.1.1.1:53"

func main() {

	if len(os.Args) < 2 {
		log.Println("Usage: dns-proxy <local-ip> <intercept-domain>")
		os.Exit(1)

	}

	localIp := strings.TrimSpace(os.Args[1])
	if localIp == "" {
		log.Fatalf("Local IP is required")
	}

	addr, err := netip.ParseAddr(localIp)
	if err != nil {
		log.Fatalf("Invalid local IP: %s", localIp)
	}

	localIp = addr.String()

	interceptDomain := strings.TrimSpace(os.Args[2])
	if interceptDomain == "" {
		log.Fatalf("Intercept domain is required")
	}

	dns.HandleFunc(".", func(c context.Context, w dns.ResponseWriter, r *dns.Msg) {
		handleDns(c, w, r, localIp, interceptDomain)
	})

	server := &dns.Server{
		Addr: ":53",
		Net:  "udp",
	}

	log.Println("Starting DNS server on :53")
	log.Fatal(server.ListenAndServe())

}

func handleDns(c context.Context, w dns.ResponseWriter, r *dns.Msg, localIp string, interceptDomain string) {

	res := &dns.Msg{
		Data: r.Data,
		MsgHeader: dns.MsgHeader{
			ID:                 r.ID,
			Response:           true,
			RecursionDesired:   r.RecursionDesired,
			RecursionAvailable: true,
		},
		Question: r.Question,
	}

	for _, q := range r.Question {
		name := strings.ToLower(q.Header().Name)
		log.Println("DNS query:", name)

		if strings.HasSuffix(name, interceptDomain) {
			if dns.RRToType(q) == dns.TypeA {
				a := &dns.A{
					Hdr: dns.Header{Name: name, Class: dns.ClassINET, TTL: 3600},
					A:   rdata.A{Addr: netip.MustParseAddr(localIp)},
				}
				res.Answer = append(res.Answer, a)
				log.Printf("Intercepted A %s -> %s", name, localIp)
			} else {
				log.Printf("Intercepted %s (type %d) -> empty", name, dns.RRToType(q))
			}

			if err := res.Pack(); err != nil {
				log.Println("Error packing response:", err)
				return
			}
			io.Copy(w, res)
			return
		}
	}

	log.Println("Forwarding request to upstream:", UPSTREAM)

	in, err := dns.Exchange(c, r, "udp", UPSTREAM)
	if err != nil {
		log.Println("Error exchanging with upstream:", err)
		return
	}

	if err = in.Pack(); err != nil {
		log.Println("Error packing upstream response:", err)
		return
	}
	io.Copy(w, in)
}
