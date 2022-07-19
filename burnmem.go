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
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"path"
	"strings"
	"time"

	"github.com/chaosblade-io/chaosblade-spec-go/channel"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
	"github.com/shirou/gopsutil/mem"

	"github.com/sirupsen/logrus"
)

const PageCounterMax uint64 = 9223372036854770000

const ErrPrefix = "Error:"

// 128K
type Block [32 * 1024]int32

var (
	burnMemStart, burnMemStop, burnMemNohup, includeBufferCache      bool
	memPercent, memReserve, memRate                                  int
	ExitMessageForTesting                                            string
)

func main() {
	flag.BoolVar(&burnMemStart, "start", false, "start burn memory")
	flag.BoolVar(&burnMemStop, "stop", false, "stop burn memory")
	flag.BoolVar(&burnMemNohup, "nohup", false, "nohup to run burn memory")
	flag.BoolVar(&includeBufferCache, "include-buffer-cache", false, "ram model mem-percent is exclude buffer/cache")
	flag.IntVar(&memPercent, "mem-percent", 0, "percent of burn memory")
	flag.IntVar(&memReserve, "reserve", 0, "reserve to burn memory, unit is M")
	flag.IntVar(&memRate, "rate", 100, "burn memory rate, unit is M/S, only support for ram mode")
	ParseFlagAndInitLog()

	if burnMemStart {
		startBurnMem()
	} else if burnMemStop {
		if success, errs := stopBurnMem(); !success {
			PrintErrAndExit(errs)
		}
	} else if burnMemNohup {
	    burnMemWithRam()
	} else {
		PrintAndExitWithErrPrefix("less --start or --stop flag")
	}

}

var dirName = "burnmem_tmpfs"

var fileName = "file"

var fileCount = 1

var ExitFunc = os.Exit

func burnMemWithRam() {
	tick := time.Tick(time.Second)
	var cache = make(map[int][]Block, 1)
	var count = 1
	cache[count] = make([]Block, 0)
	if memRate <= 0 {
		memRate = 100
	}
	for range tick {
		_, expectMem, err := calculateMemSize(memPercent, memReserve)
		if err != nil {
			stopBurnMemFunc()
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
			logrus.Debugf("count: %d, len(buf): %d, cap(buf): %d, expect mem: %d, fill size: %d",
				count, len(buf), cap(buf), expectMem, fillSize)
			cache[count] = append(buf, make([]Block, fillSize)...)
		}
	}
}

var burnMemBin = "chaos_burnmem"

var cl = channel.NewLocalChannel()

var stopBurnMemFunc = stopBurnMem

var runBurnMemFunc = runBurnMem

func startBurnMem() {
	ctx := context.Background()
	runBurnMemFunc(ctx, memPercent, memReserve, memRate, includeBufferCache)
}

func runBurnMem(ctx context.Context, memPercent, memReserve, memRate int, includeBufferCache bool) {
	args := fmt.Sprintf(`%s --nohup --mem-percent %d --reserve %d --rate %d --mode %s --include-buffer-cache=%t`,
		path.Join(util.GetProgramPath(), burnMemBin), memPercent, memReserve, memRate, includeBufferCache)
	args = fmt.Sprintf(`%s > /dev/null 2>&1 &`, args)
	response := cl.Run(ctx, "nohup", args)
	if !response.Success {
		stopBurnMemFunc()
		PrintErrAndExit(response.Err)
	}
	// check pid
	newCtx := context.WithValue(context.Background(), channel.ProcessKey, "--nohup")
	pids, err := cl.GetPidsByProcessName(burnMemBin, newCtx)
	if err != nil {
		stopBurnMemFunc()
		PrintErrAndExit(fmt.Sprintf("run burn memory failed, cannot get the burning program pid, %v",
			err))
	}
	if len(pids) == 0 {
		stopBurnMemFunc()
		PrintErrAndExit(fmt.Sprintf("run burn memory failed, cannot find the burning program pid"))
	}
}

func stopBurnMem() (success bool, errs string) {
	ctx := context.WithValue(context.Background(), channel.ProcessKey, "nohup")
	ctx = context.WithValue(ctx, channel.ExcludeProcessKey, "stop")
	pids, _ := cl.GetPidsByProcessName(burnMemBin, ctx)
	var response *spec.Response
	if pids != nil && len(pids) != 0 {
		response = cl.Run(ctx, "kill", fmt.Sprintf(`-9 %s`, strings.Join(pids, " ")))
		if !response.Success {
			return false, response.Err
		}
	}
	return true, errs
}

func calculateMemSize(percent, reserve int) (int64, int64, error) {
	total := int64(0)
	available := int64(0)
	virtualMemory, err := mem.VirtualMemory()
	if err != nil {
		return 0, 0, err
	}
	total = int64(virtualMemory.Total)
	available = int64(virtualMemory.Free)
	if !includeBufferCache {
		available = available + int64(virtualMemory.Buffers+virtualMemory.Cached)
	}
	reserved := int64(0)
	if percent != 0 {
		reserved = (total * int64(100-percent) / 100) / 1024 / 1024
	} else {
		reserved = int64(reserve)
	}
	expectSize := available/1024/1024 - reserved

	logrus.Debugf("available: %d, percent: %d, reserved: %d, expectSize: %d",
		available/1024/1024, percent, reserved, expectSize)

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