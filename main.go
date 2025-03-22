package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"

	"nullprogram.com/x/optparse"
)

func main() {
	options := []optparse.Option{
		{Long: "socket", Short: 's', Kind: optparse.KindRequired},
		{Long: "port", Short: 'p', Kind: optparse.KindRequired},
		{Long: "listen-ip", Short: 'l', Kind: optparse.KindRequired},
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
		w.WriteHeader(http.StatusOK)
	})
	server := http.Server{
		Handler: mux,
	}

	if port != 0 {
		server.Addr = listen_ip + ":" + strconv.Itoa(port)
		server.ListenAndServe()
	} else {
		listener, err := net.Listen("unix", socket)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		server.Serve(listener)
		listener.Close()
	}
}
