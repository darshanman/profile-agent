package internal

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

//APIRequest ...
type APIRequest struct {
	agent *Agent
	histo *prometheus.HistogramVec //http.Handler
}

func newAPIRequest(a *Agent, histo *prometheus.HistogramVec) *APIRequest {
	ar := &APIRequest{
		agent: a,
		histo: histo,
	}

	return ar
}

func (ar *APIRequest) push(vecType string, payload []string) (map[string]interface{}, error) {
	// switch vecType {
	// case "histogram":
	if len(payload) < 1 {
		log.Println("returning... len(payload) is, ", len(payload))
		return nil, nil
	}
	now := time.Now()
	log.Println("pushing to hist")
	log.Println("len(payload): ", len(payload))
	log.Println("payload: ", payload)
	// values := make([]string, len(payload))
	// i := 0
	// var value string
	// for _, pl := range payload {
	// 	switch p := pl.(type) {
	// 	case map[string]interface{}:
	// 		value = fmt.Sprint(p["content"])
	// 	default:
	// 		value = fmt.Sprint(p)
	// 		log.Println(value)
	// 		values[i] = value
	// 	}
	//
	// 	i++
	// }
	time.Sleep(1000 * time.Millisecond)
	if obs, err := ar.histo.GetMetricWithLabelValues(payload...); err != nil {
		log.Println("ERR: ", err)

	} else {
		obs.Observe(time.Since(now).Seconds())
	}

	// ar.histo.WithLabelValues(payload...).Observe(time.Since(now).Seconds())
	// }
	return nil, nil
}

func (ar *APIRequest) post(endpoint string, payload map[string]interface{}) (map[string]interface{}, error) {
	log.Println("returning from APIRrequest.Post")
	return nil, nil
	reqBody := map[string]interface{}{
		"runtime_type":    "go",
		"runtime_version": runtime.Version(),
		"agent_version":   AgentVersion,
		"app_name":        ar.agent.AppName,
		"app_version":     ar.agent.AppVersion,
		"app_environment": ar.agent.AppEnvironment,
		"host_name":       ar.agent.HostName,
		"process_id":      strconv.Itoa(os.Getpid()),
		"build_id":        ar.agent.buildID,
		"run_id":          ar.agent.runID,
		"run_ts":          ar.agent.runTs,
		"sent_at":         time.Now().Unix(),
		"payload":         payload,
	}

	reqbodyJSON, _ := json.Marshal(reqBody)

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(reqbodyJSON)
	w.Close()

	u := ar.agent.PromethRoute + "/agent/v1/" + endpoint
	req, err := http.NewRequest("POST", u, &buf)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(ar.agent.AgentKey, "")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	ar.agent.log("Posting API request to %v", u)

	var httpClient *http.Client
	if ar.agent.ProxyAddress != "" {
		var proxyURL *url.URL
		if proxyURL, err = url.Parse(ar.agent.ProxyAddress); err != nil {
			return nil, err
		}

		httpClient = &http.Client{
			Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
			Timeout:   time.Second * 20,
		}
	} else {
		httpClient = &http.Client{
			Timeout: time.Second * 20,
		}
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	resbodyJSON, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Received %v: %v", res.StatusCode, string(resbodyJSON))
	}

	var resBody map[string]interface{}
	if err := json.Unmarshal(resbodyJSON, &resBody); err != nil {
		return nil, fmt.Errorf("Cannot parse response body %v", string(resbodyJSON))
	}

	return resBody, nil

}
