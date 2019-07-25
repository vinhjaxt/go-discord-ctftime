package main

import (
	"fmt"
	"strings"
)

const (
	stateStart = iota
	stateQuotes
	stateArg
)

func parseCommandLine(command string) ([]string, error) {
	var args []string
	state := stateArg
	current := ""
	quote := ""
	escapeNext := false
	command = strings.Trim(command, "\r\n\t ")
	for i := 0; i < len(command); i++ {
		c := command[i]

		if state == stateQuotes {
			if string(c) != quote {
				current += string(c)
			} else {
				args = append(args, current)
				current = ""
				state = stateStart
			}
			continue
		}

		if escapeNext {
			current += string(c)
			escapeNext = false
			continue
		}

		if c == '\\' {
			escapeNext = true
			continue
		}

		if c == '"' || c == '\'' {
			state = stateQuotes
			quote = string(c)
			continue
		}

		if state == stateArg {
			if c == ' ' || c == '\t' {
				args = append(args, current)
				current = ""
				state = stateStart
			} else {
				current += string(c)
			}
			continue
		}

		if c != ' ' && c != '\t' {
			state = stateArg
			current += string(c)
		}
	}

	if state == stateQuotes {
		return []string{}, fmt.Errorf("Unclosed quote in command line: %s", command)
	}

	if current != "" {
		args = append(args, current)
	}

	return args, nil
}
