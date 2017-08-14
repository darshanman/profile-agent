package stackimpact

import (
	"fmt"
	"net/http"

	"github.com/stackimpact/stackimpact-go/internal"
)

//ErrorGroupRecoveredPanics ...
const ErrorGroupRecoveredPanics string = "Recovered panics"

//ErrorGroupUnrecoveredPanics ...
const ErrorGroupUnrecoveredPanics string = "Unrecovered panics"

//ErrorGroupHandledExceptions ...
const ErrorGroupHandledExceptions string = "Handled exceptions"

//Options ...
type Options struct {
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

//Agent ...
type Agent struct {
	internalAgent *internal.Agent

	// compatibility < 1.2.0
	DashboardAddress string
	AgentKey         string
	AppName          string
	HostName         string
	Debug            bool
}

//NewAgent DEPRECATED. Kept for compatibility with <1.4.3.
func NewAgent() *Agent {
	a := &Agent{
		internalAgent: internal.NewAgent(),
	}

	return a
}

// Agent instance
var _agent *Agent

//Start - Starts the agent with configuration options.
// Required options are AgentKey and AppName.
func Start(options Options) *Agent {
	if _agent == nil {
		_agent = &Agent{
			internalAgent: internal.NewAgent(),
		}
	}

	_agent.Start(options)

	return _agent
}

//Start -  Starts the agent with configuration options.
// Required options are AgentKey and AppName.
func (a *Agent) Start(options Options) {
	a.internalAgent.AgentKey = options.AgentKey
	a.internalAgent.AppName = options.AppName

	if options.AppVersion != "" {
		a.internalAgent.AppVersion = options.AppVersion
	}

	if options.AppEnvironment != "" {
		a.internalAgent.AppEnvironment = options.AppEnvironment
	}

	if options.HostName != "" {
		a.internalAgent.HostName = options.HostName
	}

	if options.DashboardAddress != "" {
		a.internalAgent.DashboardAddress = options.DashboardAddress
	}

	if options.ProxyAddress != "" {
		a.internalAgent.ProxyAddress = options.ProxyAddress
	}

	if options.Debug {
		a.internalAgent.Debug = options.Debug
	}

	if options.ProfileAgent {
		a.internalAgent.ProfileAgent = options.ProfileAgent
	}

	a.internalAgent.Start()
}

//Configure - DEPRECATED. Kept for compatibility with <1.2.0.
func (a *Agent) Configure(agentKey string, appName string) {
	a.Start(Options{
		AgentKey:         agentKey,
		AppName:          appName,
		HostName:         a.HostName,
		DashboardAddress: a.DashboardAddress,
		Debug:            a.Debug,
	})
}

//MeasureSegment  Starts measurement of execution time of a code segment.
// To stop measurement call Stop on returned Segment object.
// After calling Stop the segment is recorded, aggregated and
// reported with regular intervals.
func (a *Agent) MeasureSegment(segmentName string) *Segment {
	s := newSegment(a, segmentName)
	s.start()

	return s
}

//MeasureHandlerFunc - A helper function to measure HTTP handler function execution
// by wrapping http.HandleFunc method parameters.
func (a *Agent) MeasureHandlerFunc(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) (string, func(http.ResponseWriter, *http.Request)) {
	return pattern, func(w http.ResponseWriter, r *http.Request) {
		segment := a.MeasureSegment(fmt.Sprintf("Handler %s", pattern))
		defer segment.Stop()

		handlerFunc(w, r)
	}
}

//MeasureHandler - A helper function to measure HTTP handler execution
// by wrapping http.Handle method parameters.
func (a *Agent) MeasureHandler(pattern string, handler http.Handler) (string, http.Handler) {
	return pattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		segment := a.MeasureSegment(fmt.Sprintf("Handler %s", pattern))
		defer segment.Stop()

		handler.ServeHTTP(w, r)
	})
}

//RecordError - Aggregates and reports errors with regular intervals.
func (a *Agent) RecordError(err interface{}) {
	a.internalAgent.RecordError(ErrorGroupHandledExceptions, err, 1)
}

//RecordPanic - Aggregates and reports panics with regular intervals.
func (a *Agent) RecordPanic() {
	if err := recover(); err != nil {
		a.internalAgent.RecordError(ErrorGroupUnrecoveredPanics, err, 1)

		panic(err)
	}
}

//RecordAndRecoverPanic - Aggregates and reports panics with regular intervals. This function also
// recovers from panics
func (a *Agent) RecordAndRecoverPanic() {
	if err := recover(); err != nil {
		a.internalAgent.RecordError(ErrorGroupRecoveredPanics, err, 1)
	}
}
