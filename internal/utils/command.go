package utils

import (
	"bytes"
	"os/exec"
	"strings"
)

// RunCommand executes a shell command and returns the output
func RunCommand(command string) (string, error) {
	parts := strings.Split(command, " ")
	cmd := exec.Command(parts[0], parts[1:]...)
	
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err != nil {
		return stderr.String(), err
	}
	
	return out.String(), nil
}
