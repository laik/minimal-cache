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
	if len(args) < 2 {
		return &ErrReply{Message: ErrWrongArgsNumber}
	}
	err := this.stge.Set(args[0], args[1])
	if err != nil {
		return &ErrReply{Message: err}
	}
	return &OkReply{}
}
