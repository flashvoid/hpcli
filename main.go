package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/romana/rlog"
)

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there")
}

func getSecondOp(command, firstOp, secondOp string) string {
	if command == "write" && firstOp == "curr-time" {
		t := time.Now().Unix()
		log.Infof("Writing current time %d -> %x", t, t)
		return fmt.Sprintf("%x", t)
	}

	return secondOp
}

func timeFromResult(lines []string) (int64, error) {
	if len(lines) != 2 {
		return 0, fmt.Errorf("not enough lines in result, expect exactly 2 lines")
	}

	lastLineArr := strings.Split(lines[1], ":")
	if len(lastLineArr) != 2 {
		return 0, fmt.Errorf("failed to parse time out of line %v", lines)
	}

	return strconv.ParseInt(strings.Replace(lastLineArr[1], " ", "", -1), 16, 64)
}

func main() {
	btInterface := flag.String("interface", "hci0", "Bluetooth interface to use)")
	btHandle := flag.String("handle", "0x0025", "Have no idea what it is")
	gtBin := flag.String("gtBin", "/usr/local/bin/gatttool", "Full path to gatttool")
	targetMac := flag.String("mac", "00:15:83:40:72:75", "Mac address of the target node")
	command := flag.String("command", "", "One of read|write|save|reset")
	firstOp := flag.String("op1", "",
		fmt.Sprintf("Operand for read|write commands\n\t%s", getMapKeys(OpCodes)))
	secondOp := flag.String("op2", "", "Value for write command (auto for write curr-time)")
	timeout := flag.Int("timeout", 5, "Timeout for bluetooth call")

	flag.Parse()

	g := Gattt{
		Interface:  *btInterface,
		Handle:     *btHandle,
		Mac:        *targetMac,
		GatttolBin: *gtBin,
	}

	if cmd, ok := CommandCodes[*command]; !ok {
		log.Infof("Unkonwn command %s", *command)
		flag.Usage()
		os.Exit(1)
	} else {
		g.Command = cmd
	}

	if opc, ok := OpCodes[*firstOp]; !ok {
		if *command == "write" || *command == "read" {
			log.Infof("Unkonwn operand %s", *firstOp)
			flag.Usage()
			os.Exit(1)
		}

		g.Op1 = "00"
	} else {
		g.Op1 = opc
	}

	g.Length = OpLength[*firstOp]
	g.Op2 = getSecondOp(*command, *firstOp, *secondOp)

	var result []string
	var err error

	switch *command {
	case "read":
		result, err = g.Read(time.Duration(*timeout) * time.Second)
	case "write":
		err = g.Write(time.Duration(*timeout) * time.Second)
	case "save":
		err = g.Save(time.Duration(*timeout) * time.Second)
	case "reset":
		err = g.Reset(time.Duration(*timeout) * time.Second)
	}
	if err != nil {
		log.Infof("Error %s", err.Error())
		os.Exit(2)
	}

	if *command == "read" && strings.Contains(*firstOp, "time") {
		readTime, err := timeFromResult(result)
		if err != nil {
			log.Errorf("Failed to detect time marker in %s", result)
		}
		fmt.Printf("Time received %d\n", readTime)
	}
	log.Infof("Result %v", result)
}
