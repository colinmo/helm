//go:build !windows

package main

import (
	"errors"
	"os/exec"
	"strings"
)

/**
 * This code searches text files in Linux mode
 */
type saveOutput struct {
	savedOutput []byte
}

func (so *saveOutput) Write(p []byte) (n int, err error) {
	so.savedOutput = append(so.savedOutput, p...)
	return len(p), nil
}
func searchFiles(directory string, lookfor string) ([]string, error) {
	cmdVariables := []string{`/bin/sh`, "-c", `find . \( -name '*.markdown' -o -name '*.md' \) -exec grep -li '` + lookfor + `' {} \;`}
	cmd := exec.Command(cmdVariables[0], cmdVariables[1:]...)
	cmd.Dir = directory

	var so saveOutput
	cmd.Stdout = &so
	cmd.Stderr = &so
	bob := cmd.Run()
	if bob != nil {
		if bob.Error() == "exit status 1" {
			// No results
			return []string{}, nil
		}
		if bob.Error() == "exit status 2" {
			// Problem with input to executable
			return []string{}, errors.New("bad string format")
		}
	}
	output := strings.Split(strings.Trim(string(so.savedOutput), "./\n"), "\n")
	for i, x := range output {
		output[i] = strings.Trim(x, "./")
	}
	return output, nil
}
