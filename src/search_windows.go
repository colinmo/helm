//go:build windows

package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

/**
* This code searches text files, in Windows or Linux mode
 */
type saveOutput struct {
	savedOutput []byte
}

func (so *saveOutput) Write(p []byte) (n int, err error) {
	so.savedOutput = append(so.savedOutput, p...)
	return len(p), nil
}

func searchFiles(directory string, lookfor string) ([]string, error) {
	cmdVariables := []string{"findstr", fmt.Sprintf(`/simc:%s`, lookfor), "*.markdown", "*.md"}
	cmd := exec.Command(cmdVariables[0], cmdVariables[1:]...)
	cmd.Dir = directory
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW

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
		return []string{}, errors.New("unknown error: " + bob.Error())
	}

	return strings.Split(strings.Trim(string(so.savedOutput), "\r\n"), "\r\n"), nil
}
