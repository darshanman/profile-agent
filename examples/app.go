package examples

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

	profileagent "github.com/darshanman/profile-agent"
)

var agent2 *profileagent.Agent

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

func SimulateCPUUsage() {
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
	defer log.Println("exiting leakMemory")
	for j := 0; j < duration; j++ {
		go func() {
			for i := 0; i < size; i++ {
				mem = append(mem, string(i))
			}
		}()

		time.Sleep(1 * time.Second)
	}
}

//SimulateMemoryLeak - constantly
func SimulateMemoryLeak() {

	// constantTimer := time.NewTimer(2 * 3600 * time.Second)
	// go func() {
	// 	// for {
	// 	// select {
	// 	// case <-constantTimer.C:
	// 	<-constantTimer.C
	// 	leakMemory(2*3600, 1000)
	// 	// }
	// 	// }
	// }()

	go leakMemory(2*60, 1000)
}

func SimulateChannelWait() {
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

func SimulateNetworkWait() {
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

func SimulateSyscallWait() {
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

func SimulateLockWait() {
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

func SimulateSegments() {
	for {
		done1 := make(chan bool)

		go func() {
			segment := agent2.MeasureSegment("Segment1")
			defer segment.Stop()

			time.Sleep(time.Duration(100+rand.Intn(20)) * time.Millisecond)

			done1 <- true
		}()

		<-done1
	}
}

func SimulateHandlerSegments() {
	// start HTTP server
	go func() {
		http.HandleFunc(agent2.MeasureHandlerFunc("/some-handler-func", func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Duration(200+rand.Intn(50)) * time.Millisecond)

			fmt.Fprintf(w, "OK")
		}))

		http.Handle(agent2.MeasureHandler("/some-handler", http.StripPrefix("/some-handler", http.FileServer(http.Dir("/tmp")))))

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

func SimulateErrors() {
	go func() {
		for {
			agent2.RecordError(fmt.Sprintf("A handled exception %v", rand.Intn(10)))

			time.Sleep(2 * time.Second)
		}
	}()

	go func() {
		for {
			agent2.RecordError(fmt.Errorf("A handled exception %v", rand.Intn(10)))

			time.Sleep(10 * time.Second)
		}
	}()

	go func() {
		for {
			go func() {
				defer agent2.RecordAndRecoverPanic()

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
				defer agent2.RecordPanic()

				panic("An unrecovered panic")
			}()

			time.Sleep(7 * time.Second)
		}
	}()

}

func profilemain() {
	// StackImpact initialization
	agent2 = profileagent.Start(profileagent.Options{
		AgentKey:   os.Getenv("AGENT_KEY"),
		AppName:    "ExampleGoApp",
		AppVersion: "1.0.0",
		// DashboardAddress: os.Getenv("DASHBOARD_ADDRESS"), // test only
		Debug: false,
	})
	// end StackImpact initialization

	go SimulateCPUUsage()
	go SimulateMemoryLeak()
	go SimulateChannelWait()
	go SimulateNetworkWait()
	go SimulateSyscallWait()
	go SimulateLockWait()
	go SimulateSegments()
	go SimulateHandlerSegments()
	go SimulateErrors()

	select {}
}
