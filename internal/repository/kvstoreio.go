package repository

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

type kvStoreWriter struct {
	mu   sync.RWMutex
	file *os.File
}

func NewKVStoreWriter(path string) (*kvStoreWriter, error) {
	f, err := os.OpenFile(path,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0666)
	if err != nil {
		return nil, err
	}
	return &kvStoreWriter{
		mu:   sync.RWMutex{},
		file: f}, nil
}

func (w *kvStoreWriter) Put(e *kvEntry) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err = w.file.Write(append(data, '\n'))
	if err != nil {
		return err
	}

	return w.file.Sync()
}

func (w *kvStoreWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

type kvStoreReader struct {
	mu      sync.RWMutex
	file    *os.File
	scanner *bufio.Scanner
}

func NewKVStoreReader(path string) (*kvStoreReader, error) {
	f, err := os.OpenFile(path,
		os.O_RDONLY|os.O_CREATE,
		0666)
	if err != nil {
		return nil, err
	}
	return &kvStoreReader{
		mu:      sync.RWMutex{},
		file:    f,
		scanner: bufio.NewScanner(f),
	}, nil
}

type kvEntry struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

func (r *kvStoreReader) ReadAll() (map[string]string, error) {
	store := make(map[string]string)

	r.mu.RLock()
	defer r.mu.RUnlock()
	for r.scanner.Scan() {
		line := r.scanner.Bytes()
		var e kvEntry
		if err := json.Unmarshal(line, &e); err != nil {
			slog.Error(fmt.Sprintf("could not parse line %q: %v", line, err))
			continue
		}
		store[e.Key] = e.Val
	}
	return store, r.scanner.Err()
}

func (r *kvStoreReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}
