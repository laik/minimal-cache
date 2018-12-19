package command

type Get struct {
	stge dataStore
}

//Name returns the command name.
func (this *Get) Name() string {
	return "DEL"
}

//Help returns information about the command. Description, usage and etc.
func (this *Get) Help() string {
	return `Usage: Get key. Get the given key.`
}

//Execute executes the command with the given arguments.
func (this *Get) Execute(args ...string) Reply {
	var reply Reply

	reply = checkExpcetArgs(1, args...)
	if _, ok := reply.(*ErrReply); ok {
		return reply
	}

	value, err := this.stge.Get(args[0])

	if err != nil {
		reply = &ErrReply{Message: err}
	}
	reply = &StringReply{
		Message: value,
	}
	return reply
}
