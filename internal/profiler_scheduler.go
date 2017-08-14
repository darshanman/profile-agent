package internal

import (
	"math/rand"
	"time"
)

type recordFuncType func(duration int64)
type reportFuncType func()

//ProfilerScheduler ...
type ProfilerScheduler struct {
	agent          *Agent
	randSource     *rand.Rand
	recordInterval int64
	recordDuration int64
	reportInterval int64
	recordFunc     recordFuncType
	reportFunc     reportFuncType
}

func newProfilerScheduler(
	agent *Agent,
	recordInterval int64,
	recordDuration int64,
	reportInterval int64,
	recordFunc recordFuncType,
	reportFunc reportFuncType) *ProfilerScheduler {

	ps := &ProfilerScheduler{
		agent:          agent,
		randSource:     rand.New(rand.NewSource(time.Now().UnixNano())),
		recordInterval: recordInterval,
		recordDuration: recordDuration,
		reportInterval: reportInterval,
		recordFunc:     recordFunc,
		reportFunc:     reportFunc,
	}

	return ps
}

func (ps *ProfilerScheduler) start() {
	if ps.recordFunc != nil {
		maxDelay := int64(float64(ps.recordInterval - ps.recordDuration))

		recordIntervalTicker := time.NewTicker(time.Duration(ps.recordInterval) * time.Millisecond)
		go func() {
			defer ps.agent.recoverAndLog()

			for {
				select {
				case <-recordIntervalTicker.C:
					randomTimer := time.NewTimer(time.Duration(ps.randSource.Int63n(maxDelay)) * time.Millisecond)
					<-randomTimer.C

					go ps.executeRecord()
				}
			}
		}()
	}

	reportIntervalTicker := time.NewTicker(time.Duration(ps.reportInterval) * time.Millisecond)
	go func() {
		defer ps.agent.recoverAndLog()

		for {
			select {
			case <-reportIntervalTicker.C:
				go ps.executeReport()
			}
		}
	}()
}

func (ps *ProfilerScheduler) executeRecord() {
	defer ps.agent.recoverAndLog()

	ps.agent.profilerLock.Lock()
	defer ps.agent.profilerLock.Unlock()

	ps.recordFunc(ps.recordDuration)
}

func (ps *ProfilerScheduler) executeReport() {
	defer ps.agent.recoverAndLog()

	ps.agent.profilerLock.Lock()
	defer ps.agent.profilerLock.Unlock()

	ps.reportFunc()
}
