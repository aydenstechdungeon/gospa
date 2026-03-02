package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"sync"
	"time"
)

// RequestResult holds timing data for a single request
type RequestResult struct {
	StatusCode int
	Duration   time.Duration
	Error      error
}

// BenchmarkStats holds aggregated benchmark statistics
type BenchmarkStats struct {
	TotalRequests      int
	SuccessfulRequests int
	FailedRequests     int
	TotalDuration      time.Duration
	MinLatency         time.Duration
	MaxLatency         time.Duration
	AvgLatency         time.Duration
	MedianLatency      time.Duration
	P95Latency         time.Duration
	P99Latency         time.Duration
	RequestsPerSecond  float64
	BytesReceived      int64
	StatusCodes        map[int]int
	Errors             []string
	Concurrency        int
}

func main() {
	// Get absolute path to server binary
	cwd, _ := os.Getwd()
	serverPath := cwd + "/../website/dist/server"
	serverDir := cwd + "/../website"

	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		fmt.Printf("Server binary not found at: %s\n", serverPath)
		fmt.Println("Please build the counter example first")
		os.Exit(1)
	}

	fmt.Println("╔══════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              GOSPA COUNTER SERVER EXTENDED BENCHMARK TEST             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Start the server
	fmt.Println("🚀 Starting server...")
	cmd := exec.Command(serverPath)
	cmd.Dir = serverDir

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}

	// Capture server output in background (suppress for cleaner output)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			// Suppress server output for cleaner benchmark results
			_ = scanner.Text()
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			// Suppress server output for cleaner benchmark results
			_ = scanner.Text()
		}
	}()

	// Wait for server to be ready
	fmt.Println("⏳ Waiting for server to be ready...")
	baseURL := "http://localhost:3000"

	var ready bool
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		resp, err := http.Get(baseURL + "/")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == 200 {
				ready = true
				break
			}
		}
	}

	if !ready {
		fmt.Println("❌ Server failed to start within 3 seconds")
		_ = cmd.Process.Kill()
		os.Exit(1)
	}
	fmt.Println("✅ Server is ready!")
	fmt.Println()

	// Extended benchmark configurations with gradual increase
	configs := []struct {
		name       string
		endpoint   string
		concurrent int
		requests   int
	}{
		{"Warmup", "/", 1, 100},
		{"Low Load", "/", 5, 2500},
		{"Medium-Low Load", "/", 10, 5000},
		{"Medium Load", "/", 25, 12500},
		{"Medium-High Load", "/", 50, 25000},
		{"High Load", "/", 75, 37500},
		{"Very High Load", "/", 100, 50000},
		{"Extreme Load", "/", 150, 75000},
		{"Maximum Load", "/", 200, 100000},
	}

	var allStats []struct {
		name  string
		stats BenchmarkStats
	}

	totalRequests := 0
	startTime := time.Now()

	for i, config := range configs {
		fmt.Printf("📊 [%d/%d] Running benchmark: %s\n", i+1, len(configs), config.name)
		fmt.Printf("   Concurrency: %3d | Requests: %6d | ", config.concurrent, config.requests)

		stats := runBenchmark(baseURL+config.endpoint, config.concurrent, config.requests)
		stats.Concurrency = config.concurrent
		allStats = append(allStats, struct {
			name  string
			stats BenchmarkStats
		}{config.name, stats})

		totalRequests += config.requests

		fmt.Printf("RPS: %8.2f | Avg: %6.2fms | P95: %6.2fms\n",
			stats.RequestsPerSecond,
			float64(stats.AvgLatency.Microseconds())/1000,
			float64(stats.P95Latency.Microseconds())/1000)
	}

	totalDuration := time.Since(startTime)

	// Shutdown server
	fmt.Println()
	fmt.Println("🛑 Shutting down server...")
	_ = cmd.Process.Kill()
	_ = cmd.Wait()

	// Print detailed results table
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                              DETAILED BENCHMARK RESULTS                                              ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ Total Test Duration: %-78s ║\n", totalDuration.Round(time.Millisecond))
	fmt.Printf("║ Total Requests: %-82d ║\n", totalRequests)
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════════════════╣")
	fmt.Println("║ Test Name          │ Concurrency │ Requests │   RPS    │ Avg Latency │ P95 Latency │ Success Rate ║")
	fmt.Println("╠════════════════════╪═════════════╪══════════╪══════════╪═════════════╪═════════════╪══════════════╣")

	for _, s := range allStats {
		successRate := float64(s.stats.SuccessfulRequests) / float64(s.stats.TotalRequests) * 100
		fmt.Printf("║ %-18s │ %11d │ %8d │ %8.2f │ %10.2fms │ %10.2fms │ %11.2f%% ║\n",
			s.name,
			s.stats.Concurrency,
			s.stats.TotalRequests,
			s.stats.RequestsPerSecond,
			float64(s.stats.AvgLatency.Microseconds())/1000,
			float64(s.stats.P95Latency.Microseconds())/1000,
			successRate)
	}
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════════════════════════╝")

	// Print ASCII Graph
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                         REQUESTS PER SECOND vs CONCURRENCY GRAPH                                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Find max RPS for scaling
	maxRPS := 0.0
	for _, s := range allStats {
		if s.stats.RequestsPerSecond > maxRPS {
			maxRPS = s.stats.RequestsPerSecond
		}
	}

	// Print graph
	graphWidth := 60
	fmt.Printf("RPS\n")
	fmt.Printf("↑\n")

	// Y-axis labels and bars
	ySteps := 10
	for y := ySteps; y >= 0; y-- {
		threshold := float64(y) / float64(ySteps) * maxRPS
		fmt.Printf("%10.0f │ ", threshold)

		for _, s := range allStats {
			if s.name == "Warmup" {
				continue // Skip warmup in graph
			}
			barHeight := int(s.stats.RequestsPerSecond / maxRPS * float64(graphWidth))
			thresholdHeight := int(threshold / maxRPS * float64(graphWidth))
			if barHeight >= thresholdHeight && thresholdHeight > 0 {
				fmt.Print("█")
			} else if thresholdHeight > 0 {
				fmt.Print(" ")
			}
		}
		fmt.Println()
	}

	// X-axis
	fmt.Printf("%10s └", "")
	for _, s := range allStats {
		if s.name == "Warmup" {
			continue
		}
		fmt.Print("─")
	}
	fmt.Println()

	// X-axis labels
	fmt.Printf("%12s ", "Concurrency:")
	for _, s := range allStats {
		if s.name == "Warmup" {
			continue
		}
		fmt.Printf("%d ", s.stats.Concurrency)
	}
	fmt.Println()

	// Print latency graph
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                         LATENCY TRENDS (Avg, P95, P99) vs CONCURRENCY                                ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Find max latency for scaling
	maxLatency := 0.0
	for _, s := range allStats {
		latMs := float64(s.stats.P99Latency.Microseconds()) / 1000
		if latMs > maxLatency {
			maxLatency = latMs
		}
	}

	fmt.Printf("Latency(ms)\n")
	fmt.Printf("↑\n")

	// Y-axis labels and bars for latency
	for y := ySteps; y >= 0; y-- {
		threshold := float64(y) / float64(ySteps) * maxLatency
		fmt.Printf("%8.1f │ ", threshold)

		for _, s := range allStats {
			if s.name == "Warmup" {
				continue
			}
			avgHeight := int(float64(s.stats.AvgLatency.Microseconds()) / 1000 / maxLatency * float64(graphWidth))
			thresholdHeight := int(threshold / maxLatency * float64(graphWidth))

			if avgHeight >= thresholdHeight && thresholdHeight > 0 {
				fmt.Print("▓") // Avg latency
			} else if thresholdHeight > 0 {
				fmt.Print(" ")
			}
		}
		fmt.Println()
	}

	// X-axis
	fmt.Printf("%9s └", "")
	for _, s := range allStats {
		if s.name == "Warmup" {
			continue
		}
		fmt.Print("─")
	}
	fmt.Println()

	// X-axis labels
	fmt.Printf("%11s ", "Concurrency:")
	for _, s := range allStats {
		if s.name == "Warmup" {
			continue
		}
		fmt.Printf("%d ", s.stats.Concurrency)
	}
	fmt.Println()
	fmt.Println()
	fmt.Println("Legend: ▓ = Average Latency")

	// Print summary statistics
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                         SUMMARY STATISTICS                                           ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Find best performing configuration
	bestRPS := allStats[0]
	lowestLatency := allStats[0]
	for _, s := range allStats {
		if s.stats.RequestsPerSecond > bestRPS.stats.RequestsPerSecond {
			bestRPS = s
		}
		if s.stats.AvgLatency < lowestLatency.stats.AvgLatency {
			lowestLatency = s
		}
	}

	fmt.Printf("┌─────────────────────────────────────────────────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│ %-35s %60s │\n", "Peak Performance:", "")
	fmt.Printf("│   %-32s %15.2f %34s │\n", "Maximum Requests Per Second:", bestRPS.stats.RequestsPerSecond, "")
	fmt.Printf("│   %-32s %15d %34s │\n", "Achieved at concurrency:", bestRPS.stats.Concurrency, "")
	fmt.Printf("│   %-32s %15s %34s │\n", "Average latency at peak:", bestRPS.stats.AvgLatency.Round(time.Microsecond), "")
	fmt.Printf("├─────────────────────────────────────────────────────────────────────────────────────────────────────┤\n")
	fmt.Printf("│ %-35s %60s │\n", "Best Latency:", "")
	fmt.Printf("│   %-32s %15s %34s │\n", "Lowest Average Latency:", lowestLatency.stats.AvgLatency.Round(time.Microsecond), "")
	fmt.Printf("│   %-32s %15d %34s │\n", "At concurrency:", lowestLatency.stats.Concurrency, "")
	fmt.Printf("│   %-32s %15.2f %34s │\n", "RPS at lowest latency:", lowestLatency.stats.RequestsPerSecond, "")
	fmt.Printf("├─────────────────────────────────────────────────────────────────────────────────────────────────────┤\n")

	// Calculate overall success rate
	totalSuccessful := 0
	totalFailed := 0
	for _, s := range allStats {
		totalSuccessful += s.stats.SuccessfulRequests
		totalFailed += s.stats.FailedRequests
	}
	overallSuccessRate := float64(totalSuccessful) / float64(totalSuccessful+totalFailed) * 100

	fmt.Printf("│ %-35s %60s │\n", "Overall Statistics:", "")
	fmt.Printf("│   %-32s %15d %34s │\n", "Total Successful Requests:", totalSuccessful, "")
	fmt.Printf("│   %-32s %15d %34s │\n", "Total Failed Requests:", totalFailed, "")
	fmt.Printf("│   %-32s %14.2f%% %34s │\n", "Overall Success Rate:", overallSuccessRate, "")
	fmt.Printf("│   %-32s %15s %34s │\n", "Total Test Duration:", totalDuration.Round(time.Millisecond), "")
	fmt.Printf("└─────────────────────────────────────────────────────────────────────────────────────────────────────┘\n")

	// Performance classification
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                    PERFORMANCE CLASSIFICATION                                        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Classify based on peak RPS
	var performanceClass string
	var performanceEmoji string
	if bestRPS.stats.RequestsPerSecond >= 20000 {
		performanceClass = "EXCELLENT"
		performanceEmoji = "🚀🚀🚀"
	} else if bestRPS.stats.RequestsPerSecond >= 10000 {
		performanceClass = "VERY GOOD"
		performanceEmoji = "🚀🚀"
	} else if bestRPS.stats.RequestsPerSecond >= 5000 {
		performanceClass = "GOOD"
		performanceEmoji = "🚀"
	} else if bestRPS.stats.RequestsPerSecond >= 1000 {
		performanceClass = "ACCEPTABLE"
		performanceEmoji = "✓"
	} else {
		performanceClass = "NEEDS IMPROVEMENT"
		performanceEmoji = "⚠️"
	}

	fmt.Printf("┌─────────────────────────────────────────────────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│  %s %-20s Peak RPS: %10.2f                                              │\n", performanceEmoji, performanceClass, bestRPS.stats.RequestsPerSecond)
	fmt.Printf("├─────────────────────────────────────────────────────────────────────────────────────────────────────┤\n")
	fmt.Printf("│  Server: GoSPA v0.1.3 with Fiber v2.52.11                                                          │\n")
	fmt.Printf("│  Endpoint: GET / (HTML Page with Server-Side Rendering)                                            │\n")
	fmt.Printf("│  Test Configuration: Gradual load increase from 1 to 200 concurrent connections                    │\n")
	fmt.Printf("└─────────────────────────────────────────────────────────────────────────────────────────────────────┘\n")
}

func runBenchmark(url string, concurrent int, totalRequests int) BenchmarkStats {
	results := make(chan RequestResult, totalRequests)
	var wg sync.WaitGroup

	// Distribute requests across workers
	requestsPerWorker := totalRequests / concurrent
	extraRequests := totalRequests % concurrent

	start := time.Now()

	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		workerRequests := requestsPerWorker
		if i < extraRequests {
			workerRequests++
		}

		go func(count int) {
			defer wg.Done()
			for j := 0; j < count; j++ {
				makeRequest(url, results)
			}
		}(workerRequests)
	}

	// Wait for all requests to complete
	wg.Wait()
	close(results)

	totalDuration := time.Since(start)

	// Collect results
	var durations []time.Duration
	var bytesReceived int64
	statusCodes := make(map[int]int)
	var errors []string
	var successfulRequests int
	var failedRequests int
	minLatency := time.Hour
	maxLatency := time.Nanosecond

	for result := range results {
		if result.Error != nil {
			failedRequests++
			errors = append(errors, result.Error.Error())
			continue
		}

		successfulRequests++
		statusCodes[result.StatusCode]++
		durations = append(durations, result.Duration)
		bytesReceived += int64(result.StatusCode) // Approximate

		if result.Duration < minLatency {
			minLatency = result.Duration
		}
		if result.Duration > maxLatency {
			maxLatency = result.Duration
		}
	}

	// Calculate statistics
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	var avgLatency, medianLatency, p95Latency, p99Latency time.Duration
	if len(durations) > 0 {
		var total time.Duration
		for _, d := range durations {
			total += d
		}
		avgLatency = total / time.Duration(len(durations))
		medianLatency = durations[len(durations)/2]
		p95Latency = durations[int(float64(len(durations))*0.95)]
		p99Latency = durations[int(float64(len(durations))*0.99)]
	}

	rps := float64(successfulRequests) / totalDuration.Seconds()

	return BenchmarkStats{
		TotalRequests:      totalRequests,
		SuccessfulRequests: successfulRequests,
		FailedRequests:     failedRequests,
		TotalDuration:      totalDuration,
		MinLatency:         minLatency,
		MaxLatency:         maxLatency,
		AvgLatency:         avgLatency,
		MedianLatency:      medianLatency,
		P95Latency:         p95Latency,
		P99Latency:         p99Latency,
		RequestsPerSecond:  rps,
		BytesReceived:      bytesReceived,
		StatusCodes:        statusCodes,
		Errors:             errors,
	}
}

func makeRequest(url string, results chan<- RequestResult) {
	start := time.Now()

	resp, err := http.Get(url)
	if err != nil {
		results <- RequestResult{
			StatusCode: 0,
			Duration:   time.Since(start),
			Error:      err,
		}
		return
	}
	defer func() { _ = resp.Body.Close() }()

	// Read body to ensure complete request
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		results <- RequestResult{
			StatusCode: resp.StatusCode,
			Duration:   time.Since(start),
			Error:      err,
		}
		return
	}

	_ = body // We don't need the body content for benchmarking

	results <- RequestResult{
		StatusCode: resp.StatusCode,
		Duration:   time.Since(start),
		Error:      nil,
	}
}
