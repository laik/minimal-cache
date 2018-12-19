package command

type Del struct {
	stge dataStore
}

//Name returns the command name.
func (this *Del) Name() string {
	return "DEL"
}

//Help returns information about the command. Description, usage and etc.
func (this *Del) Help() string {
	return `Usage: DEL key; Del the given key.`
}

//Execute executes the command with the given arguments.
func (this *Del) Execute(args ...string) Reply {
	var reply Reply

	reply = checkExpcetArgs(1, args...)

	if _, ok := reply.(*ErrReply); ok {
		return reply
	}

	if err := this.stge.Del(args[0]); err != nil {
		reply = &ErrReply{Message: err}
	}
	return reply
}
