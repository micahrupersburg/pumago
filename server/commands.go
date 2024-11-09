package server

import (
	"fmt"
	"regexp"
	"strings"
)

type Command int

const (
	None Command = iota
	Raw  Command = iota
	Watch
	Query
)

func (c Command) String() string {
	return [...]string{"none", "raw", "watch", "query"}[c]
}
func ParseCommand(input string) (Command, error) {
	input = strings.ToLower(input)
	switch input {
	case "raw":
		return Raw, nil
	case "watch":
		return Watch, nil
	case "query":
		return Query, nil
	default:
		return -1, fmt.Errorf("invalid command: %s", input)
	}
}

var CommandRegex = regexp.MustCompile(`^/(\w+)\s*(.*)$`)

func command(input string) (Command, string) {
	matches := CommandRegex.FindStringSubmatch(input)
	if matches == nil {
		return None, input
	}
	parseCommand, err := ParseCommand(matches[1])
	if err != nil {
		return None, input
	}

	return parseCommand, matches[2]
}
