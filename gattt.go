package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"time"

	log "github.com/romana/rlog"
)

var (
	CommandCodes = map[string]string{
		"write": "00",
		"read":  "01",
		"reset": "02", // "020000",
		"save":  "03", // "030000",
	}
)

var (
	OpCodes = map[string]string{
		"curr-time":       "00",
		"dry-time":        "01",
		"water-time":      "02",
		"day-time":        "03",
		"night-time":      "04",
		"last-water-time": "05",
		"white-level":     "06",
		"red-level":       "07",
		"green-level":     "08",
		"blue-level":      "09",
	}

	OpLength = map[string]string{
		"":                "00",
		"curr-time":       "04",
		"dry-time":        "04",
		"water-time":      "04",
		"day-time":        "04",
		"night-time":      "04",
		"last-water-time": "04",
		"white-level":     "01",
		"red-level":       "01",
		"green-level":     "01",
		"blue-level":      "01",
	}
)

func getMapKeys(m map[string]string) string {
	var res string
	for k, _ := range m {
		res += fmt.Sprintf("|%s", k)
	}
	return res
}

const (
	BtListen = "--listen"
)

type Gattt struct {
	Interface  string
	Handle     string
	Command    string
	Op1        string
	Length     string
	Op2        string
	GatttolBin string
	Mac        string
}

func (g Gattt) Write(timeout time.Duration) error {
	args := []string{
		"-i", g.Interface,
		fmt.Sprintf("--device=%s", g.Mac),
		"--char-write-req",
		fmt.Sprintf("--handle=%s", g.Handle),
		fmt.Sprintf("--value=%s%s%s%s", g.Command, g.Op1, g.Length, g.Op2), // "010004"),
	}

	log.Infof("Executing %s %v", g.GatttolBin, args)

	cmd := exec.Command(g.GatttolBin, args...)

	_, err := TimedGatttExec(cmd, timeout, GatttWriteSuccess)
	return err
}

func (g Gattt) Read(timeout time.Duration) ([]string, error) {
	args := []string{
		"-i", g.Interface,
		fmt.Sprintf("--device=%s", g.Mac),
		"--char-write-req",
		fmt.Sprintf("--handle=%s", g.Handle),
		fmt.Sprintf("--value=%s%s%s%s", g.Command, g.Op1, g.Length, g.Op2),
		BtListen}

	log.Infof("Executing %s %v", g.GatttolBin, args)

	cmd := exec.Command(g.GatttolBin, args...)

	res, err := TimedGatttExec(cmd, timeout, GatttNotificationRead)
	return res, err
}

func (g Gattt) Save(timeout time.Duration) error {
	args := []string{
		"-i", g.Interface,
		fmt.Sprintf("--device=%s", g.Mac),
		"--char-write-req",
		fmt.Sprintf("--handle=%s", g.Handle),
		fmt.Sprintf("--value=%s%s%s%s", g.Command, g.Op1, g.Length, g.Op2), // "010004"),
	}

	log.Infof("Executing %s %v", g.GatttolBin, args)

	cmd := exec.Command(g.GatttolBin, args...)

	_, err := TimedGatttExec(cmd, timeout, GatttNotificationRead)
	return err
}

func (g Gattt) Reset(timeout time.Duration) error {
	args := []string{
		"-i", g.Interface,
		fmt.Sprintf("--device=%s", g.Mac),
		"--char-write-req",
		fmt.Sprintf("--handle=%s", g.Handle),
		fmt.Sprintf("--value=%s%s%s%s", g.Command, g.Op1, g.Length, g.Op2), // "010004"),
	}

	log.Infof("Executing %s %v", g.GatttolBin, args)

	cmd := exec.Command(g.GatttolBin, args...)

	_, err := TimedGatttExec(cmd, timeout, GatttNotificationRead)
	return err
}

var GatttTimeout = time.Duration(1 * time.Second)

const (
	GatttWriteSuccess     = "Characteristic value was written successfully"
	GatttNotificationRead = "Notification handle"
)

func TimedGatttExec(cmd *exec.Cmd, timeout time.Duration, waitFor string) ([]string, error) {
	done := make(chan struct{})

	var result []string

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return result, err
	}

	err = cmd.Start()
	if err != nil {
		return result, err
	}

	scanner := bufio.NewScanner(stdOut)
	go func() {
		for scanner.Scan() {
			text := scanner.Text()
			result = append(result, text)

			if strings.Contains(text, waitFor) {
				log.Infof("Stopping gatttool upon catching target string %s", text)
				cmd.Process.Kill()
				close(done)
				return
			}
		}
	}()

	select {
	case <-time.After(timeout):
		log.Infof("Stopping gatttool upon timeout")
		cmd.Process.Kill()
		return result, GatttTimeoutError{}

	case <-done:
		return result, nil
	}

	return result, nil
}

type GatttTimeoutError struct{}

func (GatttTimeoutError) Error() string {
	return fmt.Sprintf("Timeout reached while waiting for gatttool output")
}
