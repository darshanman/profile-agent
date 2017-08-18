package internal

import (
	"fmt"
	"math"
	"sync/atomic"
	"time"
	"unsafe"
)

//TypeState ...
const TypeState string = "state"

//TypeCounter ...
const TypeCounter string = "counter"

//TypeProfile ...
const TypeProfile string = "profile"

//TypeTrace ...
const TypeTrace string = "trace"

//CategoryCPU ...
const CategoryCPU string = "cpu"

//CategoryMemory ...
const CategoryMemory string = "memory"

//CategoryGC ...
const CategoryGC string = "gc"

//CategoryRuntime ...
const CategoryRuntime string = "runtime"

//CategoryCPUProfile ...
const CategoryCPUProfile string = "cpu-profile"

//CategoryMemoryProfile ...
const CategoryMemoryProfile string = "memory-profile"

//CategoryBlockProfile ...
const CategoryBlockProfile string = "block-profile"

//CategoryLockProfile ...
const CategoryLockProfile string = "lock-profile"

//CategoryHTTPTrace ...
const CategoryHTTPTrace string = "http-trace"

//CategorySegmentTrace ...
const CategorySegmentTrace string = "segment-trace"

//CategoryErrorProfile ...
const CategoryErrorProfile string = "error-profile"

//NameCPUTime ...
const NameCPUTime string = "CPU time"

//NameCPUUsage ...
const NameCPUUsage string = "CPU usage"

//NameMaxRSS ...
const NameMaxRSS string = "Max RSS"

//NameCurrentRSS ...
const NameCurrentRSS string = "Current RSS"

//NameVMSize ...
const NameVMSize string = "VM Size"

//NameNumGoroutines ...
const NameNumGoroutines string = "Number of goroutines"

//NameNumCgoCalls ...
const NameNumCgoCalls string = "Number of cgo calls"

//NameAllocated ....
const NameAllocated string = "Allocated memory"

//NameLookups ...
const NameLookups string = "Lookups"

//NameMallocs ...
const NameMallocs string = "Mallocs"

//NameFrees ...
const NameFrees string = "Frees"

//NameHeapSys ...
const NameHeapSys string = "Heap obtained"

//NameHeapIdle ....
const NameHeapIdle string = "Heap idle"

//NameHeapInuse ...
const NameHeapInuse string = "Heap non-idle"

//NameHeapReleased ...
const NameHeapReleased string = "Heap released"

//NameHeapObjects ...
const NameHeapObjects string = "Heap objects"

//NameGCTotalPause ...
const NameGCTotalPause string = "GC total pause"
const NameNumGC string = "Number of GCs"
const NameGCCPUFraction string = "GC CPU fraction"
const NameHeapAllocation string = "Heap allocation"
const NameBlockingCallTimes string = "Blocking call times"
const NameHTTPTransactionBreakdown string = "HTTP transaction breakdown"

const UnitNone string = ""
const UnitMillisecond string = "millisecond"
const UnitMicrosecond string = "microsecond"
const UnitNanosecond string = "nanosecond"
const UnitByte string = "byte"
const UnitKilobyte string = "kilobyte"
const UnitPercent string = "percent"

const TriggerTimer string = "timer"
const TriggerAnomaly string = "anomaly"

const ReservoirSize int = 1000

type filterFuncType func(name string) bool

//Measurement ...
type Measurement struct {
	id        string
	trigger   string
	value     float64
	duration  int64
	breakdown *BreakdownNode
	timestamp int64
}

//Metric ...
type Metric struct {
	agent        *Agent
	id           string
	typ          string
	category     string
	name         string
	unit         string
	measurement  *Measurement
	hasLastValue bool
	lastValue    float64
}

func newMetric(agent *Agent, typ string, category string, name string, unit string) *Metric {
	metricID := sha1String(agent.AppName + agent.AppEnvironment + agent.HostName + typ + category + name + unit)

	m := &Metric{
		agent:        agent,
		id:           metricID,
		typ:          typ,
		category:     category,
		name:         name,
		unit:         unit,
		measurement:  nil,
		hasLastValue: false,
		lastValue:    0,
	}

	return m
}

func (m *Metric) hasMeasurement() bool {
	return m.measurement != nil
}

func (m *Metric) createMeasurement(trigger string, value float64, duration int64, breakdown *BreakdownNode) {
	ready := true

	if m.typ == TypeCounter {
		if !m.hasLastValue {
			ready = false
			m.hasLastValue = true
			m.lastValue = value
		} else {
			tmpValue := value
			value = value - m.lastValue
			m.lastValue = tmpValue
		}
	}

	if ready {
		m.measurement = &Measurement{
			id:        m.agent.uuid(),
			trigger:   trigger,
			value:     value,
			duration:  duration,
			breakdown: breakdown,
			timestamp: time.Now().Unix(),
		}
	}
}

func (m *Metric) toStringArray() []string {
	sArray := make([]string, 0)
	sArray = append(
		sArray,
		// m.typ,
		// m.name,
		fmt.Sprintf("%v%v", m.measurement.value, m.unit),
		fmt.Sprint(m.measurement.duration),
		fmt.Sprint(m.measurement.timestamp),
		fmt.Sprint(m.measurement.breakdown),
	)
	return sArray
}

//TODO: REFACTOR - @Darshan
func (m *Metric) toMap() map[string]interface{} {
	var measurementMap map[string]interface{}
	if m.measurement != nil {
		var breakdownMap map[string]interface{}
		if m.measurement.breakdown != nil {
			breakdownMap = m.measurement.breakdown.toMap()
		}

		measurementMap = map[string]interface{}{
			"id":        m.measurement.id,
			"trigger":   m.measurement.trigger,
			"value":     m.measurement.value,
			"duration":  m.measurement.duration,
			"breakdown": breakdownMap,
			"timestamp": m.measurement.timestamp,
		}
	}

	metricMap := map[string]interface{}{
		"id":          m.id,
		"type":        m.typ,
		"category":    m.category,
		"name":        m.name,
		"unit":        m.unit,
		"measurement": measurementMap,
	}

	return metricMap
}

//AddFloat64 ....
func AddFloat64(addr *float64, val float64) (new float64) {
	for {
		old := LoadFloat64(addr)
		new = old + val
		if atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(addr)),
			math.Float64bits(old),
			math.Float64bits(new),
		) {
			break
		}
	}

	return
}

func StoreFloat64(addr *float64, val float64) {
	atomic.StoreUint64((*uint64)(unsafe.Pointer(addr)), math.Float64bits(val))
}

func LoadFloat64(addr *float64) float64 {
	return math.Float64frombits(atomic.LoadUint64((*uint64)(unsafe.Pointer(addr))))
}
