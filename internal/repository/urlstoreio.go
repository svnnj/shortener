package repository

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

type urlStoreWriter struct {
	mu   sync.Mutex
	file *os.File
	path string
}

type urlStoreReader struct {
	mu      sync.RWMutex
	file    *os.File
	scanner *bufio.Scanner
	path    string
}

func NewURLStoreWriter(path string) (*urlStoreWriter, error) {
	f, err := os.OpenFile(path,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0666)
	if err != nil {
		return nil, fmt.Errorf("open writer file %q: %w", path, err)
	}
	return &urlStoreWriter{
		mu:   sync.Mutex{},
		file: f,
		path: path,
	}, nil
}

func (w *urlStoreWriter) Put(rec URLRec) error {
	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal record with token %q: %w", rec.Token, err)
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err = w.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write record with token %q: %w", rec.Token, err)
	}

	return w.file.Sync()
}

func (w *urlStoreWriter) PutBatch(recs []URLRec) error {
	var batch bytes.Buffer
	for _, v := range recs {
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("marshal record (token %q): %w", v.Token, err)
		}
		batch.Write(data)
		batch.WriteByte('\n')
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.file.Write(batch.Bytes()); err != nil {
		return fmt.Errorf("batch write %d records: %w", len(recs), err)
	}

	return w.file.Sync()
}

func (w *urlStoreWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func NewURLStoreReader(path string) (*urlStoreReader, error) {
	f, err := os.OpenFile(path,
		os.O_RDONLY|os.O_CREATE,
		0666)
	if err != nil {
		return nil, fmt.Errorf("open reader file: %w", err)
	}
	return &urlStoreReader{
		mu:      sync.RWMutex{},
		file:    f,
		scanner: bufio.NewScanner(f),
		path:    path,
	}, nil
}

func (r *urlStoreReader) ReadAll() (map[string]string, error) {
	store := make(map[string]string)

	r.mu.RLock()
	defer r.mu.RUnlock()
	for r.scanner.Scan() {
		line := r.scanner.Bytes()
		var rec URLRec
		if err := json.Unmarshal(line, &rec); err != nil {
			slog.Warn(fmt.Sprintf("could not parse line %q: %v", line, err))
			continue
		}
		store[rec.Token] = rec.OriginalURL
	}
	return store, r.scanner.Err()
}

func (r *urlStoreReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}
