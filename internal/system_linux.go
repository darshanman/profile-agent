// +build linux

package internal

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"syscall"
)

//VMSizeRe ...
var VMSizeRe = regexp.MustCompile(`VmSize:\s+(\d+)\s+kB`)

//VMRSSRe ...
var VMRSSRe = regexp.MustCompile(`VmRSS:\s+(\d+)\s+kB`)

func readCPUTime() (int64, error) {
	rusage := new(syscall.Rusage)
	if err := syscall.Getrusage(0, rusage); err != nil {
		return 0, err
	}

	var cpuTimeNanos int64
	cpuTimeNanos =
		int64(rusage.Utime.Sec*1e9) +
			int64(rusage.Utime.Usec) +
			int64(rusage.Stime.Sec*1e9) +
			int64(rusage.Stime.Usec)

	return cpuTimeNanos, nil
}

func readMaxRSS() (int64, error) {
	rusage := new(syscall.Rusage)
	if err := syscall.Getrusage(0, rusage); err != nil {
		return 0, err
	}

	var maxRSS int64
	maxRSS = int64(rusage.Maxrss)

	if runtime.GOOS == "darwin" {
		maxRSS = maxRSS / 1000
	}

	return maxRSS, nil
}

func readCurrentRSS() (int64, error) {
	pid := os.Getpid()

	output, err := ioutil.ReadFile(fmt.Sprintf("/proc/%v/status", pid))
	if err != nil {
		return 0, err
	}

	results := VMRSSRe.FindStringSubmatch(string(output))
	var v int64
	var e error
	if len(results) >= 2 {

		if v, e = strconv.ParseInt(results[1], 10, 64); e == nil {
			return v, nil
		}
		return 0, e

	}
	return 0, errors.New("Unable to read current RSS")

}

func readVMSize() (int64, error) {
	pid := os.Getpid()

	output, err := ioutil.ReadFile(fmt.Sprintf("/proc/%v/status", pid))
	if err != nil {
		return 0, err
	}

	results := VMSizeRe.FindStringSubmatch(string(output))

	if len(results) >= 2 {
		var v int64
		var e error
		if v, e = strconv.ParseInt(results[1], 10, 64); e != nil {
			return v, nil
		}

		return 0, e

	}
	return 0, errors.New("Unable to read VM size")

}
