package main

import (
	"fmt"
	"os"

	"github.com/clownware/go-performance-starter/internal/performance"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <binary-path>\n", os.Args[0])
		os.Exit(1)
	}

	binaryPath := os.Args[1]

	fmt.Printf("Checking binary size: %s\n", binaryPath)

	if err := performance.CheckBinarySize(binaryPath); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	// Get file info to print success message
	info, err := os.Stat(binaryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading binary: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Binary size check passed: %d bytes (%.2f MB)\n",
		info.Size(),
		float64(info.Size())/1024/1024)
}
