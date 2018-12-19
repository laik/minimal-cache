package command

import (
	"errors"
)

var (
	//ErrWrongArgsNumber means that given arguments not acceptable by Command.
	ErrWrongArgsNumber = errors.New("command: wrong args number")
	//ErrWrongTypeOp means that operation is not acceptable for the given key.
	ErrWrongTypeOp = errors.New("command: wrong type operation")
)

//Command represents a command thats server can execute.
type Command interface {
	//Name returns the command name.
	Name() string
	//Help returns information about the command. Description, usage and etc.
	Help() string
	//Execute executes the command with the given arguments.
	Execute(args ...string) Reply
}

type dataStore interface {
	//Set puts a new value at the given Key.
	Set(string, string) error
	//Get gets a value by the given key.
	Get(string) (string, error)
	//Del deletes a value by the given key.
	Del(string) error
	//Keys returns all stored keys.
	Keys() ([]string, error)
}

type CommandParser interface {
	Parse(str string) (cmd Command, args []string, err error)
}

func checkExpcetArgs(expectQuantity int, args ...string) Reply {
	if len(args) != expectQuantity {
		return &ErrReply{Message: ErrWrongArgsNumber}
	}
	return &OkReply{}
}
