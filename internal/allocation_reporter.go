package internal

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"math"
	"runtime"
	"runtime/pprof"

	"github.com/darshanman/profile-agent/internal/pprof/profile"
)

const (
	goexitTag = "runtime.goexit"
)

type recordSorter []runtime.MemProfileRecord

func (x recordSorter) Len() int {
	return len(x)
}

func (x recordSorter) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

func (x recordSorter) Less(i, j int) bool {
	return x[i].InUseBytes() > x[j].InUseBytes()
}

func readMemAlloc() float64 {
	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)
	return float64(memStats.Alloc)
}

//AllocationReporter ...
type AllocationReporter struct {
	agent             Agent
	profilerScheduler *ProfilerScheduler
}

func newAllocationReporter(agent Agent) *AllocationReporter {
	ar := &AllocationReporter{
		agent:             agent,
		profilerScheduler: nil,
	}

	ar.profilerScheduler = newProfilerScheduler(agent, 0, 0, 120000, nil,
		func() {
			ar.report()
		},
	)

	return ar
}

func (ar *AllocationReporter) start() {
	ar.profilerScheduler.start()
}

func (ar *AllocationReporter) report() {
	if ar.agent.GetConfig().isProfilingDisabled() {
		return
	}

	ar.agent.Log("Reading heap profile...")
	p, e := ar.readHeapProfile()
	if e != nil {
		ar.agent.error(e)
		return
	}
	if p == nil {
		return
	}
	ar.agent.log("Done.")

	// allocated size
	if callGraph, err := ar.createAllocationCallGraph(p); err != nil {
		ar.agent.error(err)
	} else {
		// filter calls with lower than 10KB
		callGraph.filter(2, 10000, math.Inf(0))

		metric := newMetric(ar.agent, TypeProfile, CategoryMemoryProfile, NameHeapAllocation, UnitByte)
		metric.createMeasurement(TriggerTimer, callGraph.measurement, 0, callGraph)
		ar.agent.messageQueue.addMessage("metric", metric.toMap())
	}
}

func (ar *AllocationReporter) createAllocationCallGraph(p *profile.Profile) (*BreakdownNode, error) {
	// find "inuse_space" type index
	inuseSpaceTypeIndex := -1
	for i, s := range p.SampleType {
		if s.Type == "inuse_space" {
			inuseSpaceTypeIndex = i
			break
		}
	}

	// find "inuse_space" type index
	inuseObjectsTypeIndex := -1
	for i, s := range p.SampleType {
		if s.Type == "inuse_objects" {
			inuseObjectsTypeIndex = i
			break
		}
	}

	if inuseSpaceTypeIndex == -1 || inuseObjectsTypeIndex == -1 {
		return nil, errors.New("Unrecognized profile data")
	}

	// build call graph
	rootNode := newBreakdownNode("root")

	for _, s := range p.Sample {
		if !ar.agent.ProfileAgent && isAgentStack(s) {
			continue
		}

		value := s.Value[inuseSpaceTypeIndex]
		count := s.Value[inuseObjectsTypeIndex]
		if value == 0 {
			continue
		}
		rootNode.increment(float64(value), int64(count))

		currentNode := rootNode
		for i := len(s.Location) - 1; i >= 0; i-- {
			l := s.Location[i]
			funcName, fileName, fileLine := readFuncInfo(l)

			if funcName == goexitTag {
				continue
			}

			frameName := fmt.Sprintf("%v (%v:%v)", funcName, fileName, fileLine)
			currentNode = currentNode.findOrAddChild(frameName)
			currentNode.increment(float64(value), int64(count))
		}
	}

	return rootNode, nil
}

func (ar *AllocationReporter) readHeapProfile() (*profile.Profile, error) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	err := pprof.WriteHeapProfile(w)
	if err != nil {
		return nil, err
	}

	w.Flush()
	r := bufio.NewReader(&buf)
	var p *profile.Profile
	var perr error
	if p, perr = profile.Parse(r); perr != nil {
		return nil, perr
	}
	if serr := symbolizeProfile(p); serr != nil {
		return nil, serr
	}

	if verr := p.CheckValid(); verr != nil {
		return nil, verr
	}

	return p, nil

}
