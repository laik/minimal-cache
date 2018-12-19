package command

type Set struct {
	stge dataStore
}

//Name returns the command name.
func (this *Set) Name() string {
	return "SET"
}

//Help returns information about the command. Description, usage and etc.
func (this *Set) Help() string {
	return `Usage: SET key value [Set the given key value.]`
}

//Execute executes the command with the given arguments.
func (this *Set) Execute(args ...string) Reply {
	var reply Reply

	reply = checkExpcetArgs(2, args...)
	if _, ok := reply.(*ErrReply); ok {
		return reply
	}
	if err := this.stge.Set(args[0], args[1]); err != nil {
		reply = &ErrReply{Message: err}
	}
	return reply
}
