package integration

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const ping_timeout_sec = 1

var latency_pattern = regexp.MustCompile("time=(.*) *ms")

// This works for both mac and linux output, not sure if for windows too...
func parseResults(cmd *exec.Cmd, name, address string, pattern *regexp.Regexp) (string, error) {
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("event='ping_cmd_error' name='%s' addresss='%s' error='%s'\n", name, address, err)
	}
	if len(output) > 0 {
		for _, line := range strings.Split(string(output), "\n") {
			if matches := pattern.FindStringSubmatch(line); matches != nil && len(matches) >= 2 {
				return fmt.Sprintf("event='ping_latency' name='%s' addresss='%s' latency_ms='%s'\n", name, address, strings.TrimSpace(matches[1])), nil
			}
		}
	}
	// guess we never found a ping latency in our response data
	return "", fmt.Errorf("event='missed_ping_latency' name='%s' addresss='%s'\n", name, address)
}

func pingLinux(address, name string, timeoutSec int, pattern *regexp.Regexp) (string, error) {
	// -c 1 --> send one packet -w <sec> deadline/timeout in seconds before giving up
	cmd := exec.Command("ping", "-c", "1", "-w", strconv.Itoa(timeoutSec), address)
	return parseResults(cmd, name, address, pattern)
}

func pingMac(address, name string, timeoutSec int, pattern *regexp.Regexp) (string, error) {
	// -c 1 --> send one packet -t <sec> timeout in sec before ping exits
	// regardless of packets received
	cmd := exec.Command("ping", "-c", "1", "-t", strconv.Itoa(timeoutSec), address)
	return parseResults(cmd, name, address, pattern)
}

func pingWindows(address, name string, timeoutSec int, pattern *regexp.Regexp) (string, error) {
	// -n 1 --> send one packet/echo -w <miliseconds> wait up to this many ms for
	// each reply (only one reply in this case...).  Note the * 1000 since we're
	// configured with seconds and this arg takes miliseconds.
	cmd := exec.Command("ping", "-n", "1", "-w", strconv.Itoa(timeoutSec*1000), address)
	return parseResults(cmd, name, address, pattern)
}

func Ping(address, name string) (string, error) {
	switch os := runtime.GOOS; os {
	case "darwin":
		return pingMac(address, name, ping_timeout_sec, latency_pattern)
	case "linux":
		return pingLinux(address, name, ping_timeout_sec, latency_pattern)
	case "windows":
		return pingWindows(address, name, ping_timeout_sec, latency_pattern)
	default:
		return "", fmt.Errorf("Unsupported OS type: %s.  Can't establish ping cmd args", os)
	}
}
