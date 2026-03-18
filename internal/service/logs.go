package service

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	logReadChunkSize     = 64 * 1024
	defaultLogsLimit     = 200
	defaultLogsDirection = "older"
)

type rawLogEntry struct {
	Level   string          `json:"level"`
	Time    time.Time       `json:"time"`
	Message json.RawMessage `json:"message"`
}

type lineWithPos struct {
	content  string
	startPos int64
}

func readTaskLogsFromDisk(filePath string, cursor int64, limit int, direction string) (*TaskLogsResponse, error) {
	logPath, err := resolveLogFilePath(filePath)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("open log file %s: %w", logPath, err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat log file %s: %w", logPath, err)
	}
	fileSize := stat.Size()

	if limit <= 0 {
		limit = defaultLogsLimit
	}

	dir := strings.ToLower(strings.TrimSpace(direction))
	if dir != "newer" {
		dir = defaultLogsDirection
	}

	if cursor > fileSize {
		cursor = fileSize
	}

	resp := &TaskLogsResponse{}

	if dir == "older" {
		start := cursor
		if start < 0 {
			start = fileSize
		}
		lines, newOffset, hasMore, err := readLinesBackward(f, start, limit, fileSize)
		if err != nil {
			return nil, err
		}
		resp.Logs = parseLogLines(lines)
		resp.OlderCursor = newOffset
		resp.NewerCursor = start
		resp.HasMoreOlder = hasMore
		resp.HasMoreNewer = resp.NewerCursor < fileSize
		return resp, nil
	}

	lines, newOffset, hasMore, err := readLinesForward(f, cursor, limit, fileSize)
	if err != nil {
		return nil, err
	}
	resp.Logs = parseLogLines(lines)
	resp.NewerCursor = newOffset
	resp.OlderCursor = cursor
	resp.HasMoreNewer = hasMore
	resp.HasMoreOlder = resp.OlderCursor > 0
	return resp, nil
}

func resolveLogFilePath(filePath string) (string, error) {
	if strings.TrimSpace(filePath) == "" {
		return "", fmt.Errorf("log file path is required")
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(filePath)))
	baseDir := filepath.Join(DefaultConfigDir, hash)
	if _, err := os.Stat(baseDir); err != nil {
		return "", fmt.Errorf("logs directory not found: %s: %w", baseDir, err)
	}

	logsDir := filepath.Join(baseDir, "logs")
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return "", fmt.Errorf("read logs dir %s: %w", logsDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "sync_") {
			return filepath.Join(logsDir, entry.Name(), "olake.log"), nil
		}
	}
	return "", fmt.Errorf("no sync_* log folders found in %s", logsDir)
}

func isValidLogLine(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}

	var entry rawLogEntry
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		return false
	}
	if strings.EqualFold(entry.Level, "debug") {
		return false
	}
	return true
}

func readLinesBackward(f *os.File, startOffset int64, limit int, fileSize int64) ([]string, int64, bool, error) {
	if limit <= 0 {
		return nil, 0, false, fmt.Errorf("limit must be greater than 0")
	}

	if startOffset > fileSize {
		startOffset = fileSize
	}
	if startOffset <= 0 {
		return []string{}, 0, false, nil
	}

	offset := startOffset
	var tail []byte
	found := make([]lineWithPos, 0, limit)

	for offset > 0 && len(found) < limit {
		toRead := offset
		maxChunk := int64(logReadChunkSize)
		if toRead > maxChunk {
			toRead = maxChunk
		}
		readPos := offset - toRead

		chunk := make([]byte, int(toRead))
		n, err := f.ReadAt(chunk, readPos)
		if err != nil && err != io.EOF {
			return nil, 0, false, err
		}
		chunk = chunk[:n]

		data := make([]byte, 0, len(chunk)+len(tail))
		data = append(data, chunk...)
		data = append(data, tail...)

		for len(found) < limit {
			idx := bytes.LastIndexByte(data, '\n')
			if idx == -1 {
				tail = data
				break
			}

			lineBytes := data[idx+1:]
			line := string(lineBytes)
			linePos := readPos + int64(idx) + 1
			if isValidLogLine(line) {
				found = append(found, lineWithPos{content: line, startPos: linePos})
			}
			data = data[:idx]
		}

		offset = readPos
		if offset == 0 {
			if len(tail) > 0 && len(found) < limit {
				line := string(tail)
				if isValidLogLine(line) {
					found = append(found, lineWithPos{content: line, startPos: 0})
				}
			}
			break
		}
	}

	if len(found) == 0 {
		return []string{}, 0, false, nil
	}

	lines := make([]string, len(found))
	for i, line := range found {
		lines[len(found)-1-i] = line.content
	}

	oldestPos := found[len(found)-1].startPos
	hasMore := oldestPos > 0 && len(found) == limit
	if !hasMore {
		oldestPos = 0
	}

	return lines, oldestPos, hasMore, nil
}

func readLinesForward(f *os.File, startOffset int64, limit int, fileSize int64) ([]string, int64, bool, error) {
	if limit <= 0 {
		return nil, 0, false, fmt.Errorf("limit must be greater than 0")
	}
	if startOffset < 0 {
		startOffset = 0
	}
	if startOffset >= fileSize {
		return []string{}, fileSize, false, nil
	}

	if _, err := f.Seek(startOffset, io.SeekStart); err != nil {
		return nil, 0, false, err
	}

	reader := bufio.NewReader(f)
	lines := make([]string, 0, limit)
	current := startOffset

	for len(lines) < limit {
		bytesLine, err := reader.ReadBytes('\n')
		if len(bytesLine) > 0 {
			current += int64(len(bytesLine))
			line := strings.TrimRight(string(bytesLine), "\r\n")
			if isValidLogLine(line) {
				lines = append(lines, line)
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, 0, false, err
		}
	}

	hasMore := current < fileSize && len(lines) == limit
	if !hasMore {
		current = fileSize
	}

	return lines, current, hasMore, nil
}

func parseLogLines(lines []string) []LogEntry {
	entries := make([]LogEntry, 0, len(lines))
	for _, line := range lines {
		var raw rawLogEntry
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		entry := LogEntry{
			Level:   strings.ToLower(raw.Level),
			Time:    raw.Time.UTC().Format(time.RFC3339),
			Message: formatLogMessage(raw.Message),
		}
		entries = append(entries, entry)
	}
	return entries
}

func formatLogMessage(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	switch val := v.(type) {
	case string:
		return val
	default:
		data, err := json.Marshal(val)
		if err != nil {
			return string(raw)
		}
		return string(data)
	}
}
