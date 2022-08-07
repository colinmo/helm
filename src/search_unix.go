//go:build !windows

package main

import (
	"os"
)

/**
 * This code searches text files in Linux mode
 */

func searchFiles(directory string, lookfor string) ([]string, error) {
	cmdVariables := []string{"find", `\(`, "-name", "*.markdown", "-o", "-name", "*.md", `\)`, "-exec", "grep", "-li", lookfor, "{}", `\;`}
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
		fmt.Printf("Failed with %s\n", bob)
	}

	return strings.Split(strings.Trim(string(so.savedOutput), "\r\n"), "\r\n"), nil
}
