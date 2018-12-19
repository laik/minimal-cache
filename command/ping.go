package command

import (
	"strings"
)

const pong = "PONG"

//Ping is a PING command.
type Ping struct{}

//Name returns the command name.
func (c *Ping) Name() string {
	return "PING"
}

//Help returns usage of a PING command
func (c *Ping) Help() string {
	return `Usage: PING [message]
Returns PONG if no argument is provided, otherwise return a copy of the argument as a bulk.`
}

//Execute executes a PING command
func (c *Ping) Execute(args ...string) Reply {
	reply := checkExpcetArgs(0, args...)
	if _, ok := reply.(*OkReply); ok {
		return &StringReply{Message: "pong"}
	}
	return &StringReply{Message: strings.Join(args, " ")}
}
