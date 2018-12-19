package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/laik/minimal-cache/cache"
	"github.com/laik/minimal-cache/cluster"
	"github.com/laik/minimal-cache/command"
	"github.com/laik/minimal-cache/config"
)

func main() {
	flag.Parse()

	var (
		err      error
		c        = cache.NewCache()
		parser   = command.NewParser(c)
		opts     = config.NewOptions()
		raftNode = cluster.NewRaftNode(c, c, parser, opts)
	)

	if opts.Join != "" {
		err = raftNode.JoinCluster(opts.Join)
		fmt.Fprintf(os.Stdout, "start join raft cluster node.")
	} else {
		err = raftNode.BootstrapCluster()
	}

	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}
}
