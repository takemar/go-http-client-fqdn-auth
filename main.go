package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"nullprogram.com/x/optparse"
)

func main() {
	options := []optparse.Option{
		{Long: "socket", Short: 's', Kind: optparse.KindRequired},
		{Long: "port", Short: 'p', Kind: optparse.KindRequired},
		{Long: "listen-ip", Kind: optparse.KindRequired},
		{Long: "trusted-proxy", Kind: optparse.KindRequired},
	}
	results, allowed_domains, err := optparse.Parse(options, os.Args)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	if len(allowed_domains) == 0 {
		fmt.Fprint(os.Stderr, "allowed domain must be specified")
		os.Exit(1)
	}

	var socket string
	var port int
	var listen_ip string
	var trusted_proxies []string

	for _, result := range results {
		switch result.Long {
		case "socket":
			socket = result.Optarg
		case "port":
			port, err = strconv.Atoi(result.Optarg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "port is expected to be int, but given \"%s\"", result.Optarg)
				os.Exit(1)
			}
		case "listen-ip":
			listen_ip = result.Optarg
		case "trusted-proxy":
			ip_address := net.ParseIP(result.Optarg)
			if ip_address == nil {
				fmt.Fprintf(os.Stderr, "trusted proxy ip address \"%s\" is invalid format", result.Optarg)
				os.Exit(1)
			}
			trusted_proxies = append(trusted_proxies, ip_address.String())
		}
	}

	if socket != "" && port != 0 {
		fmt.Fprint(os.Stderr, "socket and port are not specified at the same time")
		os.Exit(1)
	}
	if listen_ip != "" && port == 0 {
		fmt.Fprint(os.Stderr, "port must be specified")
		os.Exit(1)
	}
	if socket == "" && port == 0 {
		port = 80
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		x_forwarded_for := r.Header.Get("X-Forwarded-For")
		if x_forwarded_for == "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		for _, forwarded_ip_address := range slices.Backward(strings.Split(x_forwarded_for, ",")) {
			forwarded_ip_address := net.ParseIP(strings.TrimSpace(forwarded_ip_address)).String()
			if slices.Contains(trusted_proxies, forwarded_ip_address) {
				continue
			} else {
				for _, allowed_domain := range allowed_domains {
					allowed_ip_addresses, err := net.LookupHost(allowed_domain)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					if slices.Contains(allowed_ip_addresses, forwarded_ip_address) {
						w.WriteHeader(http.StatusOK)
						return
					}
				}
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}
		w.WriteHeader(http.StatusInternalServerError)
	})
	server := http.Server{
		Handler: mux,
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		err = server.Shutdown(context.Background())
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}()

	if port != 0 {
		server.Addr = listen_ip + ":" + strconv.Itoa(port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Fprint(os.Stderr, err)
		}
	} else {
		listener, err := net.Listen("unix", socket)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		if err := server.Serve(listener); err != http.ErrServerClosed {
			fmt.Fprint(os.Stderr, err)
		}
	}
}
