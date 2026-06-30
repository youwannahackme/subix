package output

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/youwannahackme/subix/pkg/types"
)

// Writer handles writing results to various output formats
type Writer struct {
	config    *types.Config
	file      *os.File
	csvWriter *csv.Writer
	mu        sync.Mutex
	jsonArray bool
	jsonFirst bool
}

// NewWriter creates a new output writer
func NewWriter(cfg *types.Config) (*Writer, error) {
	w := &Writer{
		config:    cfg,
		jsonFirst: true,
	}

	if cfg.OutputFile != "" {
		file, err := os.Create(cfg.OutputFile)
		if err != nil {
			return nil, fmt.Errorf("could not create output file: %w", err)
		}
		w.file = file

		if cfg.OutputFormat == "csv" {
			w.csvWriter = csv.NewWriter(file)
			// Write CSV header
			if err := w.csvWriter.Write([]string{"host", "source", "ips"}); err != nil {
				return nil, err
			}
			w.csvWriter.Flush()
		}

		if cfg.OutputFormat == "json" {
			w.jsonArray = true
			_, _ = file.Write([]byte("[\n"))
		}
	}

	return w, nil
}

// Write consumes results from the channel and writes them
func (w *Writer) Write(results <-chan *types.SubdomainResult, ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case result, ok := <-results:
			if !ok {
				return
			}
			w.writeResult(result)
		}
	}
}

// writeResult writes a single result
func (w *Writer) writeResult(r *types.SubdomainResult) {
	w.mu.Lock()
	defer w.mu.Unlock()

	switch w.config.OutputFormat {
	case "json":
		w.writeJSON(r)
	case "csv":
		w.writeCSV(r)
	default:
		w.writePlain(r)
	}
}

// writePlain writes a plain text result
func (w *Writer) writePlain(r *types.SubdomainResult) {
	line := r.Host
	if w.file != nil {
		fmt.Fprintln(w.file, line)
	}
	if w.config.OutputFile == "" {
		fmt.Println(line)
	}
}

// writeJSON writes a JSON result
func (w *Writer) writeJSON(r *types.SubdomainResult) {
	data, err := json.Marshal(r)
	if err != nil {
		return
	}

	if w.jsonFirst {
		w.jsonFirst = false
	} else {
		prefix := ",\n"
		if w.file != nil {
			_, _ = w.file.Write([]byte(prefix))
		}
		if w.config.OutputFile == "" {
			fmt.Print(prefix)
		}
	}

	line := string(data)
	if w.file != nil {
		_, _ = w.file.Write([]byte("  " + line))
	}
	if w.config.OutputFile == "" {
		fmt.Print("  " + line)
	}
}

// writeCSV writes a CSV result
func (w *Writer) writeCSV(r *types.SubdomainResult) {
	ipStr := ""
	if len(r.IPs) > 0 {
		for i, ip := range r.IPs {
			if i > 0 {
				ipStr += " "
			}
			ipStr += ip
		}
	}

	record := []string{r.Host, r.Source, ipStr}
	if w.csvWriter != nil {
		_ = w.csvWriter.Write(record)
		w.csvWriter.Flush()
	}
}

// Close flushes and closes the output file
func (w *Writer) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		if w.config.OutputFormat == "json" && w.jsonArray {
			_, _ = w.file.Write([]byte("\n]\n"))
		}
		if w.csvWriter != nil {
			w.csvWriter.Flush()
		}
		w.file.Close()
	}
}
