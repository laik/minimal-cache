package config

import "flag"

// var host = flag.String("host", "127.0.0.1", "localhost ip address")
var raft = flag.String("raft", "127.0.0.1:10000", "raft tcp address")
var tcp = flag.String("tcp", "127.0.0.1:10001", "gRPC server address")
var http = flag.String("http", "127.0.0.1:10002", "Http address")
var datadir = flag.String("datadir", "node1", "Raft node name and datadir name")
var joinAddress = flag.String("join", "", "join address for raft cluster")

type Options struct {
	DataDir string // data directory
	Tcp     string
	Http    string // http server address
	Raft    string // construct Raft port
	Join    string // peer address to join
}

func NewOptions() *Options {
	opts := &Options{}
	flag.Parse()

	opts.DataDir = *datadir
	opts.Tcp = *tcp
	opts.Http = *http
	opts.Raft = *raft
	opts.Join = *joinAddress

	return opts
}
