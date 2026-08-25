package notifier

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendError(t *testing.T) {
	dir := t.TempDir()
	AppendError(dir, "boom")
	// pattern YYYY/MM/DD/error.log
	matches, _ := filepath.Glob(filepath.Join(dir, "*", "*", "*", "error.log"))
	if len(matches) != 1 {
		t.Fatalf("expected 1 log file, got %v", matches)
	}
	data, _ := os.ReadFile(matches[0])
	if !strings.Contains(string(data), "boom") {
		t.Fatalf("log missing message: %s", data)
	}
}
