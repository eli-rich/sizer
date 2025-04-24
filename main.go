package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"
)

type Entry struct {
	Name string
	Size int64
	Type string // "file" or "dir"
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func main() {
	// Parse arguments manually to allow flexible ordering
	var target string
	var showAll bool

	// Look for the -a flag anywhere in arguments
	for _, arg := range os.Args[1:] {
		if arg == "-a" {
			showAll = true
		} else if len(target) == 0 {
			// First non-flag argument is the target
			target = arg
		}
	}

	// Check if we have a target
	if target == "" {
		target = "."
	}

	// Check if target exists and what type it is
	fileInfo, err := os.Stat(target)
	if err != nil {
		// Error if target doesn't exist or can't be accessed
		log.Fatalf("error accessing target: %v\n", err)
	}

	// If target is a file, simply output the size of the file and exit
	if !fileInfo.IsDir() {
		fileSize := fileInfo.Size()
		// Get absolute path for display
		absPath, err := filepath.Abs(target)
		if err != nil {
			absPath = target // Fallback to target if we can't get absolute path
		}
		fmt.Printf("\nFile: %s\n", absPath)
		fmt.Printf("Size: %s\n", formatBytes(fileSize))
		return
	}

	// If target is a directory, proceed with normal directory analysis
	rootEntries, totalSize, totalCount, err := listRootWithSizes(target, showAll)
	if err != nil {
		log.Fatalf("error walking: %v\n", err)
	}

	// Get absolute path for display
	absPath, err := filepath.Abs(target)
	if err != nil {
		absPath = target // Fallback to target if we can't get absolute path
	}

	// Print each root entry with its size
	fmt.Printf("\nContents of: %s\n", absPath)
	fmt.Println("----------------------------------------")
	for _, entry := range rootEntries {
		entryType := "DIR"
		if entry.Type == "file" {
			entryType = "FILE"
		}
		fmt.Printf("%-6s %-15s %s\n", entryType, formatBytes(entry.Size), entry.Name)
	}
	fmt.Println("----------------------------------------")
	fmt.Printf("TOTAL: %s (%d files)\n", formatBytes(totalSize), totalCount)
}

func listRootWithSizes(root string, showAll bool) ([]Entry, int64, int, error) {
	var entries []Entry
	var totalSize int64 = 0
	var totalCount int = 0

	// Read the root directory
	dirEntries, err := os.ReadDir(root)
	if err != nil {
		return nil, 0, 0, err
	}

	// Filter out dotfiles and prepare directories for parallel processing
	var dirs []os.DirEntry
	for _, dirEntry := range dirEntries {
		// Skip dotfiles (hidden files/directories) unless showAll is true
		if !showAll && len(dirEntry.Name()) > 0 && dirEntry.Name()[0] == '.' {
			continue
		}

		if dirEntry.IsDir() {
			dirs = append(dirs, dirEntry)
		} else {
			// Process files immediately
			info, err := dirEntry.Info()
			if err != nil {
				continue // Skip files we can't get info for
			}

			entries = append(entries, Entry{
				Name: dirEntry.Name(),
				Size: info.Size(),
				Type: "file",
			})

			totalSize += info.Size()
			totalCount++
		}
	}

	// Process directories in parallel for large directory sets
	if len(dirs) > 0 {
		// Create a result channel
		type dirResult struct {
			entry Entry
			size  int64
			count int
			err   error
		}

		// Use number of CPUs but cap at a reasonable maximum
		numWorkers := runtime.NumCPU()
		if numWorkers > 8 {
			numWorkers = 8
		}

		// Create channels for work distribution
		results := make(chan dirResult, len(dirs))
		var wg sync.WaitGroup

		// Start the worker pool
		dirCh := make(chan os.DirEntry, len(dirs))
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for dirEntry := range dirCh {
					entryPath := filepath.Join(root, dirEntry.Name())

					// Show scanning progress
					fmt.Printf("\rScanning: %-30s", dirEntry.Name())

					// Calculate directory size
					size, count, err := sizeDir(entryPath, showAll)

					results <- dirResult{
						entry: Entry{
							Name: dirEntry.Name(),
							Size: size,
							Type: "dir",
						},
						size:  size,
						count: count,
						err:   err,
					}
				}
			}()
		}

		// Feed directories to workers
		for _, dir := range dirs {
			dirCh <- dir
		}
		close(dirCh)

		// Wait for all workers to complete
		go func() {
			wg.Wait()
			close(results)
		}()

		// Process results
		var lastUpdate time.Time
		resultsProcessed := 0
		for result := range results {
			resultsProcessed++

			// Update progress every 100ms
			now := time.Now()
			if now.Sub(lastUpdate) > 100*time.Millisecond {
				fmt.Printf("\rScanned %d/%d directories...", resultsProcessed, len(dirs))
				lastUpdate = now
			}

			if result.err != nil {
				continue
			}

			entries = append(entries, result.entry)
			totalSize += result.size
			totalCount += result.count
		}
	}

	// Sort entries by size (largest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Size > entries[j].Size
	})

	// Clear the "Scanning" line
	fmt.Print("\033[2K\r")

	return entries, totalSize, totalCount, nil
}

func sizeDir(root string, showAll bool) (int64, int, error) {
	var total int64 = 0
	count := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Skip dotfiles/directories unless showAll is true
		name := filepath.Base(path)
		if !showAll && len(name) > 0 && name[0] == '.' {
			if d.IsDir() {
				return filepath.SkipDir // Skip entire directory
			}
			return nil // Skip file
		}

		if d.IsDir() {
			return nil
		}

		if d.Type()&os.ModeSymlink != 0 {
			count++
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		total += info.Size()
		count++
		return nil
	})
	return total, count, err
}
