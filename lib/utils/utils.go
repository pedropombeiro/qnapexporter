package utils

import (
	"io/ioutil"
	"os/exec"
	"strings"
)

// ReadFile reads the entire contents of a file as a string
func ReadFile(f string) (string, error) {
	contents, err := ioutil.ReadFile(f)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(contents)), nil
}

// ReadFileLines reads the entire contents of a file as an array of lines
func ReadFileLines(f string) ([]string, error) {
	contents, err := ReadFile(f)
	if err != nil {
		return nil, err
	}

	return strings.Split(contents, "\n"), nil
}

// ExecCommand executes a command and returns the standard output, as well as any error
func ExecCommand(cmd string, args ...string) (string, error) {
	var (
		err    error
		output []byte
	)

	c := exec.Command(cmd, args...)
	if output, err = c.Output(); err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// ExecCommandGetLines executes a command and returns the standard output
// as an array of lines, as well as any error
func ExecCommandGetLines(cmd string, args ...string) ([]string, error) {
	output, err := ExecCommand(cmd, args...)
	if err != nil {
		return nil, err
	}

	return strings.Split(output, "\n"), nil
}
