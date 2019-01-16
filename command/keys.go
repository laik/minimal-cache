package command

import (
	"fmt"
	"os"
	"regexp"
)

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
	var (
		err     error
		pattern *regexp.Regexp
	)
	if len(args) > 1 {
		return &ErrReply{Message: ErrWrongArgsNumber}
	}
	if len(args) == 1 {
		pattern, err = regexp.CompilePOSIX(args[0])
		if err != nil {
			return &ErrReply{Message: err}
		}
	}

	keysIter, err := this.stge.Keys()
	if err != nil {
		return &ErrReply{Message: err}
	}
	keysName := make([]string, 0, len(keysIter))
	for _, k := range keysIter {
		if pattern != nil {
			if !pattern.MatchString(k) {
				fmt.Fprintf(os.Stdout, "[DEBUG] regexp compile posix is not match %s %v\n", args[0], pattern)
				continue
			}
		}
		keysName = append(keysName, k)
	}
	return &SliceReply{Message: keysName}
}
