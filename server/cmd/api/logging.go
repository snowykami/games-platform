package main

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"sync"
)

const (
	ansiReset  = "\033[0m"
	ansiWhite  = "\033[37m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
)

func configureLogger() {
	output := io.Writer(os.Stderr)
	if os.Getenv("NO_COLOR") == "" {
		output = &levelColorWriter{writer: os.Stderr}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(output, nil)))
}

type levelColorWriter struct {
	mu     sync.Mutex
	writer io.Writer
}

func (w *levelColorWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	colored := colorLogLine(data)
	_, err := w.writer.Write(colored)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func colorLogLine(data []byte) []byte {
	color := logLineColor(data)
	if color == "" {
		return data
	}
	colored := make([]byte, 0, len(color)+len(data)+len(ansiReset))
	colored = append(colored, color...)
	if bytes.HasSuffix(data, []byte("\n")) {
		colored = append(colored, data[:len(data)-1]...)
		colored = append(colored, ansiReset...)
		colored = append(colored, '\n')
		return colored
	}
	colored = append(colored, data...)
	colored = append(colored, ansiReset...)
	return colored
}

func logLineColor(data []byte) string {
	switch {
	case bytes.Contains(data, []byte("level=ERROR")):
		return ansiRed
	case bytes.Contains(data, []byte("level=WARN")):
		return ansiYellow
	case bytes.Contains(data, []byte("level=INFO")):
		return ansiWhite
	default:
		return ""
	}
}
