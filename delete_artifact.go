package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
)

const deleteArtifactUrl = "https://harbor.razorpay.com/api/v2.0/projects/%s/repositories/%s/artifacts/%s"

var projectName, repositoryName string

type errResponse struct {
	digest string
	err    error
}

func newErrorResponse(digest string, err error) errResponse {
	return errResponse{digest: digest, err: err}
}

func deleteArtifact(client *http.Client, digest string, result chan<- errResponse) {
	deleteUrl := fmt.Sprintf(deleteArtifactUrl, projectName, repositoryName, digest)
	request, err := http.NewRequest("DELETE", deleteUrl, bytes.NewBufferString(`{}`))
	if err != nil {
		result <- newErrorResponse(digest, errors.Wrap(err, "error creating request"))
		return
	}
	auth := os.Getenv("Auth")
	request.Header.Set("Authorization", auth)
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		result <- newErrorResponse(digest, errors.Wrap(err, "error calling delete endpoint"))
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			result <- newErrorResponse(digest, errors.Wrap(err, "error reading response body"))
			return
		}
		result <- newErrorResponse(digest, fmt.Errorf("got non 200 response, code: %d, body: %s", response.StatusCode, string(body)))
		return
	}
	result <- newErrorResponse(digest, nil)
}

func deleteArtifactsConcurrently(concurrent int, client *http.Client) {
	ch := make(chan string, concurrent)
	result := make(chan errResponse, concurrent)
	var waitGroup sync.WaitGroup

	waitGroup.Add(concurrent)

	go func() {
		waitGroup.Wait()
		close(result)
	}()

	for i := 0; i < concurrent; i++ {
		// launch goroutines
		go func() {
			defer waitGroup.Done()
			for digest := range ch {
				deleteArtifact(client, digest, result)
			}
		}()
	}

	go func() {
		defer close(ch)
		file, err := os.Open("digests.txt")
		if err != nil {
			fmt.Printf("error opening file %v \n", err.Error())
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			digest := scanner.Text()
			ch <- digest
		}
		if err := scanner.Err(); err != nil {
			fmt.Printf("error during scanning %v \n", err.Error())
		}
	}()

	errorCount := 0
	for resp := range result {
		if resp.err != nil {
			errorCount++
			fmt.Printf("error when processing digest: %s, error: %v \n", resp.digest, resp.err.Error())
		}
	}
	fmt.Println("total error count: ", errorCount)
}
