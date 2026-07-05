package cmd

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/infrastructure/ai/pyai"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"

	"github.com/spf13/cobra"
)

var (
	credentialExtractionBenchmarkTimeout string
	credentialExtractionBenchmarkSizes   string
	credentialExtractionBenchmarkCount   int
	credentialExtractionBenchmarkOutput  string
	credentialExtractionBenchmarkCPUProf string
	credentialExtractionBenchmarkMemProf string
	credentialExtractionBenchmarkTrace   string
)

var credentialExtractionBenchmarkCmd = &cobra.Command{
	Use:   "credential-extraction-benchmark",
	Short: "Benchmarks the credential extraction pipeline across timeout and file size combinations",
	Long: `Runs the in-memory credential extraction pipeline (encrypt → decrypt → AI extract)
against varying timeout configurations and file sizes, collecting latency, memory, and goroutine metrics.

Pipeline: generate in-memory PDF → encrypt AES-256-GCM → decrypt AES-256-GCM → POST multipart to Python AI → parse response.

Default: 4 timeouts × 4 sizes × 3 reps = 48 pipeline calls (~$0.24-$0.28 API cost).

Examples:
  go run main.go credential-extraction-benchmark
  go run main.go credential-extraction-benchmark --output results.csv
  go run main.go credential-extraction-benchmark --timeout 30,60 --sizes 500,1000 --count 5
  go run main.go credential-extraction-benchmark --cpuprofile cpu.prof --memprofile mem.prof`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := credentialExtractionBenchmark(cmd, args); err != nil {
			log.Fatalf("benchmark failed: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(credentialExtractionBenchmarkCmd)

	credentialExtractionBenchmarkCmd.Flags().StringVar(&credentialExtractionBenchmarkTimeout, "timeout", "30,60,120,240", "Comma-separated timeout values in seconds")
	credentialExtractionBenchmarkCmd.Flags().StringVar(&credentialExtractionBenchmarkSizes, "sizes", "500,1000,2000,5000", "Comma-separated file sizes in KB")
	credentialExtractionBenchmarkCmd.Flags().IntVar(&credentialExtractionBenchmarkCount, "count", 3, "Number of concurrent goroutine runs per combination (minimum 3)")
	credentialExtractionBenchmarkCmd.Flags().StringVar(&credentialExtractionBenchmarkOutput, "output", "", "CSV output file path (empty = terminal only)")
	credentialExtractionBenchmarkCmd.Flags().StringVar(&credentialExtractionBenchmarkCPUProf, "cpuprofile", "", "Write CPU profile to file")
	credentialExtractionBenchmarkCmd.Flags().StringVar(&credentialExtractionBenchmarkMemProf, "memprofile", "", "Write memory profile to file")
	credentialExtractionBenchmarkCmd.Flags().StringVar(&credentialExtractionBenchmarkTrace, "trace", "", "Write execution trace to file")
}

// credentialExtractionBenchmarkRun holds per-goroutine metrics from one pipeline execution.
type credentialExtractionBenchmarkRun struct {
	LatencyMs  float64 // wall-clock time from goroutine start to pipeline return (includes mutex wait)
	AllocMB    float64 // heap bytes allocated per operation (per-run share of batch total)
	Goroutines int     // snapshot of runtime.NumGoroutine() after this run completed
	TimedOut   bool    // true if the pipeline hit the HTTP client timeout or context deadline
}

// credentialExtractionBenchmarkResult holds aggregated metrics for one (timeout, size) combination.
type credentialExtractionBenchmarkResult struct {
	TimeoutSec     int     // HTTP client timeout in seconds for this combination
	SizeKB         int     // in-memory PDF file size in KB
	AvgMs          float64 // mean latency across successful runs (timeouts excluded)
	P50Ms          float64 // 50th percentile latency (median)
	P95Ms          float64 // 95th percentile latency (tail performance)
	P99Ms          float64 // 99th percentile latency (worst-case tail)
	MinMs          float64 // fastest successful run
	MaxMs          float64 // slowest successful run
	OpsPerSec      float64 // throughput: successful_runs / wall_clock_seconds
	AllocMBPerOp   float64 // heap bytes allocated per operation (TotalAlloc delta / count)
	GoroutinesPeak int     // peak runtime.NumGoroutine() sampled during concurrent batch execution
	MutexWaitMs    float64 // latency spread (max - min), approximates mutex queue delay from AI client serialization
	TimeoutPct     float64 // (timed_out_runs / total_runs) × 100; 100.0 → all runs timed out → ETIMEOUT sentinel
}

func credentialExtractionBenchmarkParseInts(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q: %w", p, err)
		}
		result = append(result, v)
	}
	return result, nil
}
