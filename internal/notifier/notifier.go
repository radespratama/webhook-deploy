package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func Discord(url, msg, gifURL string) {
	if url == "" {
		return
	}

	payload := map[string]interface{}{
		"content": msg,
	}

	if gifURL != "" {
		payload["embeds"] = []map[string]interface{}{
			{
				"image": map[string]string{"url": gifURL},
			},
		}
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("discord notify: marshal error: %v", err)
		return
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("discord notify error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("discord notify: unexpected status %s", resp.Status)
	}
}

// AppendError writes one timestamped line to <logDir>/YYYY/MM/DD/error.log,
// e.g. logs/2026/08/15/error.log. Creates directories as needed.
func AppendError(logDir, msg string) {
	path := filepath.Join(logDir, time.Now().Format("2006/01/02"), "error.log")

	// ponytail: no file locking — single webhook server, one deploy goroutine at a time in practice
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("error log: mkdir %s: %v", path, err)
		return
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("error log: open %s: %v", path, err)
		return
	}
	defer f.Close()

	line := fmt.Sprintf("%s %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
	if _, err := f.WriteString(line); err != nil {
		log.Printf("error log: write %s: %v", path, err)
	}
}
