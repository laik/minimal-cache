package command

import (
	"bytes"
	"errors"
	"strings"
	"unicode"
)

//ErrCommandNotFound means that command could not be parsed.
var ErrCommandNotFound = errors.New("command: not found")

//Parser is a parser that parses user input and creates the appropriate command.
type Parser struct {
	stge dataStore
}

//NewParser creates a new parser
func NewParser(stge dataStore) *Parser {
	return &Parser{stge: stge}
}

//Parse parses string to Command with args
func (p *Parser) Parse(str string) (Command, []string, error) {
	var cmd Command
	args := p.extractArgs(str)

	switch strings.ToUpper(args[0]) {
	case "HELP":
		cmd = &Help{parser: p}
	case "DEL":
		cmd = &Del{stge: p.stge}
	case "GET":
		cmd = &Get{stge: p.stge}
	case "SET":
		cmd = &Set{stge: p.stge}
	case "KEYS":
		cmd = &Keys{stge: p.stge}
	case "PING":
		cmd = &Ping{}
	default:
		return nil, nil, ErrCommandNotFound
	}

	return cmd, args[1:], nil
}

func (p *Parser) extractArgs(val string) []string {
	args := make([]string, 0)
	var inQuote bool
	var buf bytes.Buffer
	for _, r := range val {
		switch {
		case r == '"':
			inQuote = !inQuote
		case unicode.IsSpace(r):
			if !inQuote && buf.Len() > 0 {
				args = append(args, buf.String())
				buf.Reset()
			} else {
				buf.WriteRune(r)
			}
		default:
			buf.WriteRune(r)
		}
	}
	if buf.Len() > 0 {
		args = append(args, buf.String())
	}
	return args
}
