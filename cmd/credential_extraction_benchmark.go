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
