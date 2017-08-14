package internal

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestCreateCallGraph(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true
	agent.ProfileAgent = true

	done := make(chan bool)

	go func() {
		// cpu
		//start := time.Now().UnixNano()
		for i := 0; i < 5000000; i++ {
			str := "str" + strconv.Itoa(i)
			_ = str + "a"
		}
		//took := time.Now().UnixNano() - start
		//fmt.Printf("TOOK: %v\n", took)

		done <- true
	}()

	agent.cpuReporter.reset()
	p, _ := agent.cpuReporter.readCPUProfile(1000)
	//fmt.Printf("PROFILE: %v\n", p.String())
	err := agent.cpuReporter.updateCPUProfile(p)
	if err != nil {
		t.Error(err)
		return
	}
	callGraph := agent.cpuReporter.profile
	callGraph.convertToPercentage(float64(1000 * 1e6 * runtime.NumCPU()))

	if false {
		fmt.Printf("CPU USAGE: %v\n", callGraph.measurement)
		fmt.Printf("CALL GRAPH: %v\n", callGraph.printLevel(0))
	}
	if callGraph.measurement < 2 {
		t.Errorf("CPU usage is too low: %v", callGraph.measurement)
	}
	if callGraph.numSamples < 1 {
		t.Error("Number of samples should be > 0")
	}
	if !strings.Contains(fmt.Sprintf("%v", callGraph.toMap()), "TestCreateCallGraph") {
		t.Error("The test function is not found in the profile")
	}

	<-done
}
