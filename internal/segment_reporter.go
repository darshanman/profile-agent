package internal

import (
	"sync"
	"time"
)

//SegmentReporter ...
type SegmentReporter struct {
	agent            Agent
	segmentNodes     map[string]*BreakdownNode
	segmentDurations map[string]*float64
	recordLock       *sync.RWMutex
}

func newSegmentReporter(agent Agent) *SegmentReporter {
	sr := &SegmentReporter{
		agent:            agent,
		segmentNodes:     make(map[string]*BreakdownNode),
		segmentDurations: make(map[string]*float64),
		recordLock:       &sync.RWMutex{},
	}

	return sr
}

func (sr *SegmentReporter) start() {
	reportTicker := time.NewTicker(60 * time.Second)
	go func() {
		defer sr.agent.recoverAndLog()

		for {
			select {
			case <-reportTicker.C:
				sr.report()
			}
		}
	}()
}

func (sr *SegmentReporter) recordSegment(name string, duration float64) {
	if name == "" {
		sr.agent.log("Empty segment name")
		return
	}

	// Segment exists for the current interval.
	sr.recordLock.RLock()
	node, nExists := sr.segmentNodes[name]
	if nExists {
		node.updateP95(duration)
	}
	sr.recordLock.RUnlock()

	// Segment does not exist yet for the current interval.
	if !nExists {
		sr.recordLock.Lock()
		node, nExists := sr.segmentNodes[name]
		if !nExists {
			// If segment was not created by other recordSegment call between locks, create it.
			node = newBreakdownNode(name)
			sr.segmentNodes[name] = node
		}
		sr.recordLock.Unlock()

		sr.recordLock.RLock()
		node.updateP95(duration)
		sr.recordLock.RUnlock()
	}

	// Save last duration
	sr.recordLock.RLock()
	lastDurationAddr, dExists := sr.segmentDurations[name]
	if dExists {
		StoreFloat64(lastDurationAddr, duration)
	}
	sr.recordLock.RUnlock()

	if !dExists {
		sr.recordLock.Lock()
		d := duration
		sr.segmentDurations[name] = &d
		sr.recordLock.Unlock()
	}
}

func (sr *SegmentReporter) report() {
	sr.recordLock.Lock()
	outgoing := sr.segmentNodes
	sr.segmentNodes = make(map[string]*BreakdownNode)
	sr.recordLock.Unlock()

	for _, segmentNode := range outgoing {
		segmentRoot := newBreakdownNode("root")
		segmentRoot.addChild(segmentNode)
		segmentRoot.evaluateP95()
		segmentRoot.propagate()

		metric := newMetric(sr.agent, TypeTrace, CategorySegmentTrace, segmentNode.name, UnitMillisecond)
		metric.createMeasurement(TriggerTimer, segmentRoot.measurement, 60, segmentRoot)
		sr.agent.messageQueue.addMessage("metric", metric.toMap())
	}
}

func (sr *SegmentReporter) readLastDurations() map[string]float64 {
	sr.recordLock.RLock()
	defer sr.recordLock.RUnlock()

	read := make(map[string]float64)
	for name, durationAddr := range sr.segmentDurations {
		duration := LoadFloat64(durationAddr)
		if duration != -1 {
			read[name] = LoadFloat64(durationAddr)
			StoreFloat64(durationAddr, -1)
		}
	}

	return read
}
