package internal

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

//AgentVersion ...
const AgentVersion = "2.0.1"

//SAASDashboardAddress ....
const SAASDashboardAddress = "http://localhost:8080/metrics"

var agentStarted bool

//Agent ...
type Agent interface {
	Start()
	RecordSegment(string, float64)
	RecordError(group string, msg interface{}, skipFrames int)
	Log(format string, values ...interface{})
	Error(error)
	//Getters
	GetAPIRequest() *APIRequest
	GetConfig() *Config
	GetConfigLoader() *ConfigLoader
	GetMessageQueue() *MessageQueue
	GetProcessReporter() *ProcessReporter
	GetCPUReporter() *CPUReporter
	GetAllocationReporter() *AllocationReporter
	GetBlockReporter() *BlockReporter
	GetSegmentReporter() *SegmentReporter
	GetErrorReporter() *ErrorReporter
}
type agent struct {
	nextID  int64
	buildID string
	runID   string
	runTs   int64

	apiRequest         *APIRequest
	config             *Config
	configLoader       *ConfigLoader
	messageQueue       *MessageQueue
	processReporter    *ProcessReporter
	cpuReporter        *CPUReporter
	allocationReporter *AllocationReporter
	blockReporter      *BlockReporter
	segmentReporter    *SegmentReporter
	errorReporter      *ErrorReporter

	profilerLock *sync.Mutex

	// Options
	DashboardAddress string
	ProxyAddress     string
	AgentKey         string
	AppName          string
	AppVersion       string
	AppEnvironment   string
	HostName         string
	Debug            bool
	ProfileAgent     bool
}

//NewAgent ...
func NewAgent() Agent {
	a := &agent{
		nextID:  0,
		runID:   "",
		buildID: "",
		runTs:   time.Now().Unix(),

		apiRequest:         nil,
		config:             nil,
		configLoader:       nil,
		messageQueue:       nil,
		processReporter:    nil,
		cpuReporter:        nil,
		allocationReporter: nil,
		blockReporter:      nil,
		segmentReporter:    nil,
		errorReporter:      nil,

		profilerLock: &sync.Mutex{},

		DashboardAddress: SAASDashboardAddress,
		ProxyAddress:     "",
		AgentKey:         "",
		AppName:          "",
		AppVersion:       "",
		AppEnvironment:   "",
		HostName:         "",
		Debug:            false,
		ProfileAgent:     false,
	}
	a.buildID = a.calculateProgramSHA1()
	a.runID = a.uuid()

	a.apiRequest = newAPIRequest(a)
	a.config = newConfig(a)
	a.configLoader = newConfigLoader(a)
	a.messageQueue = newMessageQueue(a)
	a.processReporter = newProcessReporter(a)
	a.cpuReporter = newCPUReporter(a)
	a.allocationReporter = newAllocationReporter(a)
	a.blockReporter = newBlockReporter(a)
	a.segmentReporter = newSegmentReporter(a)
	a.errorReporter = newErrorReporter(a)

	return a
}

//Start ...
func (a *agent) Start() {
	if agentStarted {
		return
	}
	agentStarted = true

	if a.HostName == "" {
		hostName, err := os.Hostname()
		if err != nil {
			a.Error(err)
		}
		a.HostName = hostName
	}

	a.configLoader.start()
	a.messageQueue.start()
	a.processReporter.start()
	a.cpuReporter.start()
	a.allocationReporter.start()
	a.blockReporter.start()
	a.segmentReporter.start()
	a.errorReporter.start()

	a.Log("Agent started.")

	return
}

func (a *agent) calculateProgramSHA1() string {
	file, err := os.Open(os.Args[0])
	if err != nil {
		a.Error(err)
		return ""
	}
	defer file.Close()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		a.Error(err)
		return ""
	}

	return hex.EncodeToString(hash.Sum(nil))
}

//RecordSegment ...
func (a *agent) RecordSegment(name string, duration float64) {
	if !agentStarted {
		return
	}

	a.segmentReporter.recordSegment(name, duration)
}

//RecordError ...
func (a *agent) RecordError(group string, msg interface{}, skipFrames int) {
	if !agentStarted {
		return
	}

	var err error
	switch v := msg.(type) {
	case error:
		err = v
	default:
		err = fmt.Errorf("%v", v)
	}

	a.errorReporter.recordError(group, err, skipFrames+1)
}

func (a *agent) Log(format string, values ...interface{}) {
	if a.Debug {
		fmt.Printf("["+time.Now().Format(time.StampMilli)+"]"+
			" StackImpact "+AgentVersion+": "+
			format+"\n", values...)
	}
}

func (a *agent) Error(err error) {
	if a.Debug {
		fmt.Println("[" + time.Now().Format(time.StampMilli) + "]" +
			" StackImpact " + AgentVersion + ": Error")
		fmt.Println(err)
	}
}

func (a *agent) recoverAndLog() {
	if err := recover(); err != nil {
		a.Log("Recovered from panic in agent: %v", err)
	}
}

func (a *agent) uuid() string {
	n := atomic.AddInt64(&a.nextID, 1)

	uuid :=
		strconv.FormatInt(time.Now().Unix(), 10) +
			strconv.Itoa(rand.Intn(1000000000)) +
			strconv.FormatInt(n, 10)

	return sha1String(uuid)
}

func sha1String(s string) string {
	sha1 := sha1.New()
	sha1.Write([]byte(s))

	return hex.EncodeToString(sha1.Sum(nil))
}

//Getters

func (a *agent) GetAPIRequest() *APIRequest {
	return a.apiRequest
}

func (a *agent) GetConfig() *Config {
	return a.config
}
func (a *agent) GetConfigLoader() *ConfigLoader {
	return a.configLoader
}
func (a *agent) GetMessageQueue() *MessageQueue {
	return a.messageQueue
}
func (a *agent) GetProcessReporter() *ProcessReporter {
	return a.processReporter
}
func (a *agent) GetCPUReporter() *CPUReporter {
	return a.cpuReporter
}
func (a *agent) GetAllocationReporter() *AllocationReporter {
	return a.allocationReporter
}

func (a *agent) GetBlockReporter() *BlockReporter {
	return a.blockReporter
}

func (a *agent) GetSegmentReporter() *SegmentReporter {
	return a.segmentReporter
}
func (a *agent) GetErrorReporter() *ErrorReporter {
	return a.errorReporter
}
