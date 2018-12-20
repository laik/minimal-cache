package main

import (
	"flag"
	"fmt"
	"os"

	cli "github.com/laik/minimal-cache/client"
)

//Values populated by the Go linker.
var (
	version = "v0.0.1"
	commit  = "v0.0.1"
	date    = "20181217"
)

// var host = flag.String("host", "127.0.0.1", "Host to connect to a server")
// var port = flag.String("port", "10001", "Port to connect to a server")
var hosts = flag.String("host", "127.0.0.1:10001", "Host to connect to a server support multipe address, example 127.0.0.1:10001,127.0.0.1:20001")
var showVersion = flag.Bool("version", false, "Show godown version.")

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("version: %s\ncommit: %s\nbuildtime: %s", version, commit, date)
		os.Exit(0)
	}

	// hostPort := net.JoinHostPort(hosts)

	if err := cli.Run(*hosts); err != nil {
		fmt.Fprintf(os.Stderr, "could not run CLI: %v", err)
	}
}
