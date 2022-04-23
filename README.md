# gorountinesConcurrency
Make parallel requests using goroutine constructs like channels and waitgroups

To run,

AUTH={Basic Auth Header} go run main.go

You can mention the concurrency limit and total number of requests in main.go file

We launch number of goroutines specified by concurrency which read from a channel for making the http requests

Uncomment lines inorder to use pooled vs non-pooled connection
