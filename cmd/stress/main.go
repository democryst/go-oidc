package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"
)

type result struct {
	duration time.Duration
	err      error
}

func main() {
	target := flag.String("u", "http://localhost:8080/token", "Target URL")
	concurrency := flag.Int("c", 50, "Concurrency level")
	duration := flag.Duration("d", 10*time.Second, "Test duration")
	flag.Parse()

	fmt.Printf("Starting stress test: %s\n", *target)
	fmt.Printf("Concurrency: %d, Duration: %v\n", *concurrency, *duration)

	results := make(chan result, 100000)
	var wg sync.WaitGroup

	start := time.Now()
	deadline := start.Add(*duration)

	// Launch workers
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{
				Timeout: 2 * time.Second,
			}
			for time.Now().Before(deadline) {
				reqStart := time.Now()
				
				// Mock a token request (auth_code flow)
				data := url.Values{}
				data.Set("grant_type", "authorization_code")
				data.Set("code", "dummy-code")
				data.Set("client_id", "dummy-client")
				data.Set("code_verifier", "dummy-verifier")

				req, _ := http.NewRequest("POST", *target, bytes.NewBufferString(data.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Header.Set("X-Request-ID", fmt.Sprintf("stress-%d", time.Now().UnixNano()))

				resp, err := client.Do(req)
				if err != nil {
					results <- result{err: err}
					continue
				}
				
				// Read body and close to ensure connection reuse
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()

				if resp.StatusCode >= 400 && resp.StatusCode != 401 && resp.StatusCode != 429 {
					// We expect 401 (unauthorized) or 429 (rate limited) if no real data is provided,
					// but 500 or 404 would be failures in the stress tool context.
					results <- result{err: fmt.Errorf("HTTP %d", resp.StatusCode)}
				} else {
					results <- result{duration: time.Since(reqStart)}
				}
			}
		}()
	}

	// Wait for completion and close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Analyze results
	var durations []time.Duration
	var errors []error
	var successCount int

	for res := range results {
		if res.err != nil {
			errors = append(errors, res.err)
		} else {
			durations = append(durations, res.duration)
			successCount++
		}
	}

	elapsed := time.Since(start)
	rps := float64(successCount) / elapsed.Seconds()

	fmt.Println("\n--- Test Results ---")
	fmt.Printf("Elapsed:     %.2fs\n", elapsed.Seconds())
	fmt.Printf("Requests:    %d (Success: %d, Fail: %d)\n", successCount+len(errors), successCount, len(errors))
	fmt.Printf("RPS:         %.2f\n", rps)

	if len(durations) > 0 {
		sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
		fmt.Printf("Min Trace:   %v\n", durations[0])
		fmt.Printf("P50 Trace:   %v\n", durations[len(durations)/2])
		fmt.Printf("P95 Trace:   %v\n", durations[int(float64(len(durations))*0.95)])
		fmt.Printf("P99 Trace:   %v\n", durations[int(float64(len(durations))*0.99)])
		fmt.Printf("Max Trace:   %v\n", durations[len(durations)-1])
	} else if len(errors) > 0 {
		fmt.Printf("Sample Error: %v\n", errors[0])
	}
}
