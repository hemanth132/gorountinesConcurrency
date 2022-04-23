package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"
)

const evaluateUrl = "https://razorx.dev.razorpay.in/v1/evaluate"

var nonPooledClient *http.Client
var pooledClient *http.Client

func init() {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DisableKeepAlives = true
	nonPooledClient = &http.Client{Timeout: time.Duration(2) * time.Second, Transport: t}

	pooledClient = &http.Client{Timeout: time.Duration(2) * time.Second, Transport: http.DefaultTransport}
}

func makeRazorxRequest(client *http.Client, result chan<- bool) {
	request, err := http.NewRequest("GET", evaluateUrl, nil)
	if err != nil {
		fmt.Println("error creating request", err.Error())
		result <- false
		return
	}
	auth := os.Getenv("Auth")
	request.Header.Set("Authorization", auth)
	query := request.URL.Query()
	query.Set("id", "IY9DXu40I53Ocw")
	query.Set("feature_flag", "pp_payment_required_amount_quantity_check")
	query.Set("environment", "stage")
	query.Set("mode", "live")
	request.URL.RawQuery = query.Encode()

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
		fmt.Println("got non 200 response", response.StatusCode ,string(body))
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
	start := time.Now()
	//makeConcurrentRequests(10000, 50, nonPooledClient)
	//fmt.Println("total time taken without pool: ", time.Since(start).Seconds())

	makeConcurrentRequests(100, 2, pooledClient)
	fmt.Println("total time taken with pool: ", time.Since(start).Seconds())
}
