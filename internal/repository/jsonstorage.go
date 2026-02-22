package repository

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type Appender struct {
	file *os.File
}

func NewAppender(filename string) (*Appender, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return &Appender{
		file: file,
	}, nil
}

func (p *Appender) Write(kv *kvRecord) error {
	data, err := json.Marshal(kv)
	if err != nil {
		return err
	}

	if _, err := p.file.Write(append(data, '\n')); err != nil {
		return err
	}

	return err
}

type Loader struct {
	file    *os.File
	scanner *bufio.Scanner
}

func NewLoader(filename string) (*Loader, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	return &Loader{
		file:    file,
		scanner: bufio.NewScanner(file),
	}, nil
}

type kvRecord struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

func (c *Loader) Load() (map[string]string, error) {
	m := make(map[string]string)

	for c.scanner.Scan() {
		line := c.scanner.Bytes()

		var rec kvRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			fmt.Fprintf(os.Stderr,
				"warning: could not parse line %q: %v\n", line, err)
			continue
		}
		m[rec.Key] = rec.Val
	}

	return m, c.scanner.Err()
}
