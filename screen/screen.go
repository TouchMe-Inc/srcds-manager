package screen

import (
	"bytes"
	"fmt"
	"github.com/keyCat/srcds-manager/config"
	"github.com/keyCat/srcds-manager/utils"
	"log"
	"os/exec"
	"strings"
	"time"
)

func GetNameForServer(server config.Server) string {
	return fmt.Sprintf("%s%02d", config.Value.Project, server.Number)
}

func GetIdlerNameForServer(server config.Server) string {
	return fmt.Sprintf("%s-idler", GetNameForServer(server))
}

// IsRunningForServer checks if screen is running for server config
func IsRunningForServer(server config.Server) bool {
	return IsRunning(GetNameForServer(server))
}

// IsNotRunningForServer checks if screen is not running for server config
func IsNotRunningForServer(server config.Server) bool {
	return !IsRunningForServer(server)
}

// IsRunning checks if screen with given name is running
// screen -list | grep %name%
func IsRunning(name string) bool {
	cmd := exec.Command("screen", "-list")
	stdout, _ := cmd.StdoutPipe()
	defer stdout.Close()
	grep := exec.Command("grep", fmt.Sprintf("%s\\s", name))
	grep.Stdin = stdout
	cmd.Start()
	out, _ := grep.Output()
	cmd.Wait()

	return string(out) != ""
}

func CountRunningServers() int {
	count := 0
	for _, server := range config.Value.Servers {
		if IsRunningForServer(server) {
			count++
		}
	}
	return count
}

func StartForServer(server config.Server) error {
	return Start(GetNameForServer(server), utils.GetStartScriptPathForServer(server))
}

func Start(name string, command string) error {
	args := strings.Fields(command)
	cmdArgs := append([]string{"-U", "-m", "-d", "-S", name}, args...)
	cmd := exec.Command("screen", cmdArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return err
}

// KillForServer kills screen for server config
func KillForServer(server config.Server) {
	Kill(GetNameForServer(server))
}

// Kill sends a quite command to a screen with the given name
// screen -r %name% -X quit
func Kill(name string) {
	if !IsRunning(name) {
		return
	}
	cmd := exec.Command("screen", "-S", name, "-X", "quit")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Printf("could not kill screen %q: %v — %v", name, err, stderr.String())
		return
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !IsRunning(name) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	log.Printf("screen %q did not exit within timeout, sending SIGKILL", name)
	// fallback: parse pid from `screen -ls` and kill -9
	_ = exec.Command("pkill", "-9", "-f", fmt.Sprintf("screen.*%s", name)).Run()
}
