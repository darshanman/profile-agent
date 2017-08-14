package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/darshanman/profile-agent"
)

var agent *profileagent.Agent

func useCPU(duration int, usage int) {
	for j := 0; j < duration; j++ {
		go func() {
			for i := 0; i < usage*80000; i++ {
				str := "str" + strconv.Itoa(i)
				_ = str + "a"
			}
		}()

		time.Sleep(1 * time.Second)
	}
}

func simulateCPUUsage() {
	// sumulate CPU usage anomaly - every 45 minutes
	cpuAnomalyTicker := time.NewTicker(45 * time.Minute)
	go func() {
		for {
			select {
			case <-cpuAnomalyTicker.C:
				// for 60 seconds produce generate 50% CPU usage
				useCPU(60, 50)
			}
		}
	}()

	// generate constant ~10% CPU usage
	useCPU(math.MaxInt64, 10)
}

func leakMemory(duration int, size int) {
	mem := make([]string, 0)

	for j := 0; j < duration; j++ {
		go func() {
			for i := 0; i < size; i++ {
				mem = append(mem, string(i))
			}
		}()

		time.Sleep(1 * time.Second)
	}
}

func simulateMemoryLeak() {
	// simulate memory leak - constantly
	constantTicker := time.NewTicker(2 * 3600 * time.Second)
	go func() {
		for {
			select {
			case <-constantTicker.C:
				leakMemory(2*3600, 1000)
			}
		}
	}()

	go leakMemory(2*3600, 1000)
}

func simulateChannelWait() {
	for {
		done := make(chan bool)

		go func() {
			wait := make(chan bool)

			go func() {
				time.Sleep(500 * time.Millisecond)

				wait <- true
			}()

			<-wait

			done <- true
		}()

		<-done

		time.Sleep(1000 * time.Millisecond)
	}
}

func simulateNetworkWait() {
	// start HTTP server
	go func() {
		http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			done := make(chan bool)

			go func() {
				time.Sleep(time.Duration(200+rand.Intn(5)) * time.Millisecond)
				done <- true
			}()
			<-done

			fmt.Fprintf(w, "OK")
		})

		if err := http.ListenAndServe(":5000", nil); err != nil {
			log.Fatal(err)
			return
		}
	}()

	requestTicker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-requestTicker.C:
			res, err := http.Get("http://localhost:5000/test")
			if err == nil {
				res.Body.Close()
			}
		}
	}
}

func simulateSyscallWait() {
	for {
		done := make(chan bool)

		go func() {
			_, err := exec.Command("sleep", "1").Output()
			if err != nil {
				fmt.Println(err)
			}

			done <- true
		}()

		time.Sleep(1 * time.Second)

		<-done
	}
}

func simulateLockWait() {
	for {
		done := make(chan bool)

		lock := &sync.RWMutex{}
		lock.Lock()

		go func() {
			lock.RLock()
			lock.RUnlock()

			done <- true
		}()

		go func() {
			time.Sleep(200 * time.Millisecond)
			lock.Unlock()
		}()

		<-done

		time.Sleep(500 * time.Millisecond)
	}
}

func simulateSegments() {
	for {
		done1 := make(chan bool)

		go func() {
			segment := agent.MeasureSegment("Segment1")
			defer segment.Stop()

			time.Sleep(time.Duration(100+rand.Intn(20)) * time.Millisecond)

			done1 <- true
		}()

		<-done1
	}
}

func simulateHandlerSegments() {
	// start HTTP server
	go func() {
		http.HandleFunc(agent.MeasureHandlerFunc("/some-handler-func", func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Duration(200+rand.Intn(50)) * time.Millisecond)

			fmt.Fprintf(w, "OK")
		}))

		http.Handle(agent.MeasureHandler("/some-handler", http.StripPrefix("/some-handler", http.FileServer(http.Dir("/tmp")))))

		if err := http.ListenAndServe(":5001", nil); err != nil {
			log.Fatal(err)
			return
		}
	}()

	requestTicker := time.NewTicker(1000 * time.Millisecond)
	for {
		select {
		case <-requestTicker.C:
			res, err := http.Get("http://localhost:5001/some-handler-func")
			if err == nil {
				res.Body.Close()
			}

			res, err = http.Get("http://localhost:5001/some-handler")
			if err == nil {
				res.Body.Close()
			}
		}
	}
}

func simulateErrors() {
	go func() {
		for {
			agent.RecordError(fmt.Sprintf("A handled exception %v", rand.Intn(10)))

			time.Sleep(2 * time.Second)
		}
	}()

	go func() {
		for {
			agent.RecordError(fmt.Errorf("A handled exception %v", rand.Intn(10)))

			time.Sleep(10 * time.Second)
		}
	}()

	go func() {
		for {
			go func() {
				defer agent.RecordAndRecoverPanic()

				panic("A recovered panic")
			}()

			time.Sleep(5 * time.Second)
		}
	}()

	go func() {
		for {
			go func() {
				defer func() {
					if err := recover(); err != nil {
						// recover from unrecovered panic
					}
				}()
				defer agent.RecordPanic()

				panic("An unrecovered panic")
			}()

			time.Sleep(7 * time.Second)
		}
	}()

}

func main() {
	// StackImpact initialization
	agent = profileagent.Start(profileagent.Options{
		AgentKey:         os.Getenv("AGENT_KEY"),
		AppName:          "ExampleGoApp",
		AppVersion:       "1.0.0",
		DashboardAddress: os.Getenv("DASHBOARD_ADDRESS"), // test only
		Debug:            false,
	})
	// end StackImpact initialization

	go simulateCPUUsage()
	go simulateMemoryLeak()
	go simulateChannelWait()
	go simulateNetworkWait()
	go simulateSyscallWait()
	go simulateLockWait()
	go simulateSegments()
	go simulateHandlerSegments()
	go simulateErrors()

	select {}
}
