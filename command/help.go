package command

import (
	"fmt"
)

//Help is the Help command
type Help struct {
	parser CommandParser
}

//Name implements Name of Command interface
func (c *Help) Name() string {
	return "HELP"
}

//Help implements Help of Command interface
func (c *Help) Help() string {
	return `Usage: HELP command [Show the usage of the given command]`
}

//Execute implements Execute of Command interface
func (c *Help) Execute(args ...string) Reply {
	var reply Reply
	switch len(args) {
	case 1:
		reply = &StringReply{Message: c.Help()}
	case 2:
		cmdName := args[0]
		cmd, _, err := c.parser.Parse(cmdName)
		if err != nil {
			if err == ErrCommandNotFound {
				return &ErrReply{Message: fmt.Errorf("command %q not found", cmdName)}
			}
			return &ErrReply{Message: err}
		}
		reply = &StringReply{Message: cmd.Help()}
	default:
		helpErr := fmt.Errorf("%s: %s. arguments length %d", ErrWrongArgsNumber, c.Help(), len(args))
		reply = &ErrReply{Message: helpErr}
	}
	return reply
}
