//go:build windows

package main

import (
	"fmt"
	"os/exec"
)

/**
* This code searches text files, in Windows or Linux mode
 */

func searchFiles(directory string, lookfor string) []string {
	cmd := exec.Command("findstr", "/s", fmt.Sprintf("/c:%s", lookfor))
	cmd.Dir = directory
	bob := cmd.Run()

	fmt.Printf("Bob is %v\n", bob)
}
