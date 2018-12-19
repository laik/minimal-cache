package command

type Keys struct {
	stge dataStore
}

//Name returns the command name.
func (this *Keys) Name() string {
	return "KEYS"
}

//Help returns information about the command. Description, usage and etc.
func (this *Keys) Help() string {
	return `Usage: Keys`
}

//Execute executes the command with the given arguments.
func (this *Keys) Execute(args ...string) Reply {
	var reply Reply

	reply = checkExpcetArgs(0, args...)
	if _, ok := reply.(*ErrReply); ok {
		return reply
	}
	keysIter, err := this.stge.Keys()
	if err != nil {
		reply = &ErrReply{Message: err}
	}

	keysName := make([]string, 0, len(keysIter))
	for _, k := range keysIter {
		keysName = append(keysName, k)
	}
	return &SliceReply{Message: keysName}
}
