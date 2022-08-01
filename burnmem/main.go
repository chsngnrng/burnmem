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
	"math"
	"os"
	"time"

	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
	"github.com/shirou/gopsutil/mem"

	"github.com/sirupsen/logrus"
)

// 128K
type Block [32 * 1024]int32

const ErrPrefix = "Error:"

var (
	memPercent, memReserve, memRate, timeSeconds int
	ExitMessageForTesting                        string
	includeSwap                                  bool
)

var ExitFunc = os.Exit

func main() {
	flag.IntVar(&memPercent, "mem-percent", 0, "percent of burn memory")
	flag.IntVar(&memReserve, "reserve", 0, "reserve to burn memory, unit is M")
	flag.IntVar(&memRate, "rate", 100, "burn memory rate, unit is M/S")
	flag.IntVar(&timeSeconds, "time", 0, "duration of work, seconds")
	flag.BoolVar(&includeSwap, "swap", true, "include swap in memory model")

	ParseFlagAndInitLog()
	burnMem()
}

func burnMem() {
	tick := time.Tick(time.Second)
	var cache = make(map[int][]Block, 1)
	var count = 1
	cache[count] = make([]Block, 0)
	if memRate <= 0 {
		memRate = 100
	}
	for range tick {
		if timeSeconds > 0 {
			if timeSeconds > 1 {
				timeSeconds -= 1
			} else {
				logrus.Debug("Done")
				PrintOutputAndExit("Done")
			}
		}
		_, expectMem, err := calculateMemSize(memPercent, memReserve)
		if err != nil {
			PrintErrAndExit(err.Error())
		}
		fillMem := expectMem
		if expectMem > 0 {
			if expectMem > int64(memRate) {
				fillMem = int64(memRate)
			} else {
				fillMem = expectMem / 10
				if fillMem == 0 {
					continue
				}
			}
			fillSize := int(8 * fillMem)
			buf := cache[count]
			if cap(buf)-len(buf) < fillSize &&
				int(math.Floor(float64(cap(buf))*1.25)) >= int(8*expectMem) {
				count += 1
				cache[count] = make([]Block, 0)
				buf = cache[count]
			}
			logrus.Debugf("count: %d, len(buf): %d, cap(buf): %d, expect mem: %d, fill size: %d, time left: %d",
				count, len(buf), cap(buf), expectMem, fillSize, timeSeconds)
			cache[count] = append(buf, make([]Block, fillSize)...)
		}
	}
}

func calculateMemSize(percent, reserve int) (int64, int64, error) {
	total := int64(0)
	available := int64(0)
	reserved := int64(0)
	expectSize := int64(0)
	if percent != 0 {
		reserved = (total * int64(100-percent) / 100) / 1024 / 1024
	} else {
		reserved = int64(reserve)
	}
	virtualMemory, err := mem.VirtualMemory()
	if err != nil {
		return 0, 0, err
	}
	total = int64(virtualMemory.Total)
	available = int64(virtualMemory.Free)

	if includeSwap {
		swapMemory, err := mem.SwapMemory()
		if err != nil {
			return 0, 0, err
		}
		total += int64(swapMemory.Total)
		available += int64(swapMemory.Free)

		expectSize = available/1024/1024 - reserved
		logrus.Debugf("available: %d, swap free: %d, percent: %d, reserved: %d, expectSize: %d",
			available/1024/1024, swapMemory.Free/1024/1024, percent, reserved, expectSize)

	} else {
		expectSize = available/1024/1024 - reserved
		logrus.Debugf("available: %d, percent: %d, reserved: %d, expectSize: %d",
			available/1024/1024, percent, reserved, expectSize)
	}

	return total / 1024 / 1024, expectSize, nil
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
