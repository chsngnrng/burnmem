/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * THE ORIGINAL CODE WAS MODIFIED IN THIS PROJECT
 * IN ORDER TO PORT IT TO WINDOWS
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/chaosblade-io/chaosblade-spec-go/util"

	"github.com/sirupsen/logrus"
)

const ErrPrefix = "Error:"

var (
	memPercent, memReserve, memRate, timeSeconds int
	ExitMessageForTesting                        string
	includeSwap                                  bool
)

var ExitFunc = os.Exit

var cwd, _ = os.Getwd()
var burnMemBin = cwd + "\\" + "burnmem.exe"

func main() {
	flag.IntVar(&memPercent, "mem-percent", 0, "percent of burn memory")
	flag.IntVar(&memReserve, "reserve", 0, "reserve to burn memory, unit is M")
	flag.IntVar(&memRate, "rate", 100, "burn memory rate, unit is M/S")
	flag.IntVar(&timeSeconds, "time", 0, "duration of work, seconds")
	flag.BoolVar(&includeSwap, "swap", true, "include swap in memory model")

	ParseFlagAndInitLog()
	checkIfBinaryExists()
	runBurnMem()
	PrintOutputAndExit("burnmem_watchdog finished gracefully")
}

func runBurnMem() {
	for timeSeconds > 0 {
		startTime := time.Now()
		arg := []string{"--mem-percent", strconv.Itoa(memPercent), "--reserve", strconv.Itoa(memReserve),
			"--rate", strconv.Itoa(memRate), "--time", strconv.Itoa(timeSeconds), "--swap", strconv.FormatBool(includeSwap)}
		cmd := exec.Command(burnMemBin, arg...)
		logrus.Debugf("Starting chaos_burnmem.exe %v", cmd.Args)
		err := cmd.Run()
		if err != nil {
			if os.IsNotExist(err) {
			}
			logrus.Debugf("burnmem exited with " + err.Error())
		}
		nowTime := time.Now()
		workTime := int(nowTime.Unix() - startTime.Unix())
		timeSeconds -= workTime
	}
}

func checkIfBinaryExists() {
	if _, err := os.Stat(burnMemBin); err != nil {
		logrus.Debugf(err.Error())
		PrintAndExitWithErrPrefix("Burnmem binary not found, exiting")
	}
}
func PrintAndExitWithErrPrefix(message string) {
	ExitMessageForTesting = fmt.Sprintf("%s %s", ErrPrefix, message)
	fmt.Fprint(os.Stderr, fmt.Sprintf("%s %s", ErrPrefix, message))
	ExitFunc(1)
}

func PrintErrAndExit(message string) {
	ExitMessageForTesting = message
	fmt.Fprint(os.Stderr, message)
	ExitFunc(1)
}

func PrintErrRespAndExit(response *spec.Response) {
	PrintErrAndExit(response.Print())
}

func PrintOutputAndExit(message string) {
	ExitMessageForTesting = message
	fmt.Fprintf(os.Stdout, message)
	ExitFunc(0)
}

func ParseFlagAndInitLog() {
	util.AddDebugFlag()
	flag.Parse()
	util.InitLog(util.Bin)
}
