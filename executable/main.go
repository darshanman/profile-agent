package main

import (
	"fmt"
	"net/http"

	profileagent "github.com/darshanman/profile-agent"
	"github.com/darshanman/profile-agent/examples"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var agent *profileagent.Agent

var buckets = []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.75, 1}

func main() {
	startFunc()
}

func startFunc() {
	histo := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "profileagent_example",
		Name:      "memory_histo_sample",
		Help:      "memory histogram vec",
		Buckets:   buckets,
	}, []string{
		"measuremntValue",
		"measuremntDuration",
		"measuremntTimestamp",
		"breakdown",
	})
	prometheus.MustRegister(
		histo,
	)
	if agent == nil {
		agent = profileagent.NewAgent(histo)
	}
	agent = profileagent.Start(profileagent.Options{

		AppName:    "ExampleGoApp",
		AppVersion: "1.0.0",
		// DashboardAddress: os.Getenv("DASHBOARD_ADDRESS"), // test only
		Debug: false,
	})

	// start := time.Now()
	// histo.WithLabelValues(
	// 	"m.id",
	// 	"m.measurement.value",
	// 	"m.measurement.duration",
	// 	"m.measurement.timestamp",
	// 	"m.typ",
	// 	"name",
	// ).Observe(time.Since(start).Seconds())

	http.HandleFunc(agent.MeasureHandlerFunc("/measure-func", func(w http.ResponseWriter, r *http.Request) {
		// s := make([]string, 0)
		// for i := 0; i < 1000; i++ {
		// 	// s = append(s, "data")
		// 	appendAll()
		// }
		// examples.SimulateMemoryLeak()
		go examples.SimulateCPUUsage()
		go examples.SimulateMemoryLeak()
		go examples.SimulateChannelWait()
		// go examples.SimulateNetworkWait()
		// go examples.SimulateSyscallWait()
		// go examples.SimulateLockWait()
		// go examples.SimulateSegments()
		// go examples.SimulateHandlerSegments()
		// go examples.SimulateErrors()
		fmt.Fprintf(w, "OK")
	}))
	// http.HandleFunc("/some-func", func(w http.ResponseWriter, r *http.Request) {
	// 	s := make([]string, 0)
	// 	for i := 0; i < 100000; i++ {
	// 		s = append(s, "data")
	// 	}
	// 	fmt.Fprintf(w, "some-func")
	// })
	http.Handle("/metrics", promhttp.Handler())

	http.ListenAndServe(":8081", nil)
}

func appendAll() {
	s := make([]string, 0)
	for i := 0; i < 100000; i++ {
		s = append(s, "data")
	}
}
