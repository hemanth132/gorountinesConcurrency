package main

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

const evaluateUrl = "http://localhost:9400/twirp/rzp.splitz.evaluate.v1.EvaluateAPI/EvaluateBulk"

//const evaluateUrl = "https://api-web.dev.razorpay.in/v1/service/razorx"

const (
	// base62 character set
	base string = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	// Random integer ceil value
	maxRandomIntCeil int64 = 9999999999999

	// UUID size
	expectedIDSize int = 14

	// Timestamp of 1st Jan 2014, in nanosecond precision
	firstJan2014EpochTs int64 = 1388534400 * 1000 * 1000 * 1000

	timeout = 5 * time.Second
)

var nonPooledClient *http.Client
var pooledClient *http.Client

func init() {
	rand.Seed(int64(randUint32()))

	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DisableKeepAlives = true
	nonPooledClient = &http.Client{Timeout: timeout, Transport: t}

	pooledClient = &http.Client{Timeout: timeout, Transport: http.DefaultTransport}
}

func base62Encode(num int64) string {
	index := base
	res := ""

	for {
		res = string(index[num%62]) + res
		num = int64(num / 62)
		if num == 0 {
			break
		}
	}
	return res
}

// randUint32 returns a random uint32 using crypto/rand which should in turn be
// used for seeding math/rand.
func randUint32() uint32 {
	buf := make([]byte, 4)
	// This panic is very very unlikely(refer crypto/rand). Anyway this
	// function should not be used regularly but for one time seeding etc.
	if _, err := cryptorand.Reader.Read(buf); err != nil {
		panic(fmt.Errorf("failed to read random bytes: %v;", err))
	}
	// Using BigEndian or LittleEndian does not matter here.
	return binary.BigEndian.Uint32(buf)
}

func uniqueId() (string, error) {
	nanotime := time.Now().UnixNano()

	random := rand.Int63n(maxRandomIntCeil)
	base62Rand := base62Encode(random)

	// We need exactly 4 chars. If greater than 4, strip and use the last 4 chars
	if len(base62Rand) > 4 {
		base62Rand = base62Rand[len(base62Rand)-4:]
	}

	// If less than 4, left pad with '0'
	base62Rand = fmt.Sprintf("%04s", base62Rand)

	b62 := base62Encode(nanotime - firstJan2014EpochTs)
	id := b62 + base62Rand

	if len(id) != expectedIDSize {
		return id, fmt.Errorf("length mismatch when generating a new id: %s", id)
	}

	return id, nil
}

func makeRazorxRequest(client *http.Client, result chan<- bool) {
	request, err := http.NewRequest("POST", evaluateUrl, bytes.NewBuffer([]byte(`{
    "bulk_evaluate": [
        {
            "id": "FMaYlTExdA4BC5",
            "experiment_id": "L68rF6Mfo0O1Vx",
            "request_data": "{\"attempts\": 5, \"merchant_id\": \"FMaYlTExdA4BC5\"}"
        },
        {
            "id": "FMaYlTExdA4BC5",
            "experiment_id": "L9B4AnIzOyq72u",
            "request_data": "{\"attempts\": 5, \"merchant_id\": \"FMaYlTExdA4BC5\"}"
        },
        {
            "id": "FMaYlTExdA4BC5",
            "experiment_id": "L9B8ohUXQW9A03",
            "request_data": "{\"attempts\": 5, \"merchant_id\": \"FMaYlTExdA4BC5\"}"
        },
        {
            "id": "FMaYlTExdA4BC5",
            "experiment_id": "L9BdlVmHAmJ3E4",
            "request_data": "{\"attempts\": 5, \"merchant_id\": \"FMaYlTExdA4BC5\"}"
        },
        {
            "id": "FMaYlTExdA4BC5",
            "experiment_id": "L9BODCF8OWDsm1",
            "request_data": "{\"attempts\": 5, \"merchant_id\": \"FMaYlTExdA4BC5\"}"
        }
    ]
}`)))
	if err != nil {
		fmt.Println("error creating request", err.Error())
		result <- false
		return
	}
	auth := os.Getenv("Auth")
	request.Header.Set("Authorization", auth)
	request.Header.Set("Content-Type", "application/json")
	//request.Header.Set("X-Admin-Token", "L7ZmfGST2BU5UML7ZmfGST2BU5UM")
	//request.Header.Set("X-Org-Id", "org_100000razorpay")
	//request.Header.Set("rzpctx-dev-serve-user", "hemanth132")
	//query := request.URL.Query()
	//id, _ := uniqueId()
	//query.Set("service_path", fmt.Sprintf("evaluate?id=%s&feature_flag=disputes_dual_write&environment=dev&mode=live", id))
	//request.URL.RawQuery = query.Encode()

	response, err := client.Do(request)
	if err != nil {
		fmt.Println("error calling evaluate", err.Error())
		result <- false
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Println("error reading response body", err.Error())
		}
		fmt.Println("got non 200 response", response.StatusCode, string(body))
		result <- false
		return
	}
	//body, err := ioutil.ReadAll(response.Body)
	//if err != nil {
	//	fmt.Println("error reading response body", err.Error())
	//}
	//fmt.Println("success", string(body))
	result <- true
}

func makeConcurrentRequests(totalRequests int, concurrent int, client *http.Client) {
	ch := make(chan int, concurrent)
	result := make(chan bool, concurrent)
	var waitGroup sync.WaitGroup

	waitGroup.Add(concurrent)

	go func() {
		waitGroup.Wait()
		close(result)
	}()

	for i := 0; i < concurrent; i++ {
		// launch goroutines
		go func(requestNumber int, ch <-chan int) {
			defer waitGroup.Done()
			for _ = range ch {
				makeRazorxRequest(client, result)
			}
		}(i, ch)
	}

	go func() {
		for i := 0; i < totalRequests; i++ {
			ch <- i
		}
		close(ch)
	}()

	errorCount := 0
	for r := range result {
		if !r {
			errorCount++
		}
	}
	fmt.Println("total error count: ", errorCount)
}

func main() {
	//start := time.Now()
	//makeConcurrentRequests(10000, 50, nonPooledClient)
	//fmt.Println("total time taken without pool: ", time.Since(start).Seconds())

	//makeConcurrentRequests(10000, 10, pooledClient)
	//fmt.Println("total time taken with pool: ", time.Since(start).Seconds())

	projectName = "razorpay"
	repositoryName = "razorx"
	deleteArtifactsConcurrently(10, pooledClient)
}
