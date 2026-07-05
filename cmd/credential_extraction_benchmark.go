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

func credentialExtractionBenchmarkGeneratePDF(sizeKB int) []byte {
	lorem := []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum. ")

	targetBytes := sizeKB * 1024

	var textBuf bytes.Buffer
	for textBuf.Len() < targetBytes {
		textBuf.Write(lorem)
	}
	textBytes := textBuf.Bytes()[:targetBytes]

	escapePDFString := func(b []byte) []byte {
		result := make([]byte, 0, len(b)+16)
		for _, ch := range b {
			switch ch {
			case '\\':
				result = append(result, '\\', '\\')
			case '(':
				result = append(result, '\\', '(')
			case ')':
				result = append(result, '\\', ')')
			default:
				result = append(result, ch)
			}
		}
		return result
	}

	maxStrLen := 60000
	var contentBuf bytes.Buffer
	contentBuf.WriteString("BT\n/F1 12 Tf\n50 750 Td\n")

	offset := 0
	for offset < len(textBytes) {
		chunkLen := len(textBytes) - offset
		if chunkLen > maxStrLen {
			chunkLen = maxStrLen
		}
		chunk := textBytes[offset : offset+chunkLen]
		escaped := escapePDFString(chunk)
		fmt.Fprintf(&contentBuf, "(%s) Tj\n0 -14 Td\n", string(escaped))
		offset += chunkLen
	}
	contentBuf.WriteString("ET\n")

	streamData := contentBuf.Bytes()

	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")

	var offsets [5]int

	offsets[0] = pdf.Len()
	pdf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	offsets[1] = pdf.Len()
	pdf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	offsets[2] = pdf.Len()
	pdf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n")

	offsets[3] = pdf.Len()
	fmt.Fprintf(&pdf, "4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(streamData), string(streamData))

	offsets[4] = pdf.Len()
	pdf.WriteString("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	xrefOffset := pdf.Len()
	pdf.WriteString("xref\n")
	fmt.Fprintf(&pdf, "0 6\n0000000000 65535 f \n")
	for i := 0; i < 5; i++ {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offsets[i])
	}

	fmt.Fprintf(&pdf, "trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefOffset)

	return pdf.Bytes()
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
