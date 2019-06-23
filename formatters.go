package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
)

type Formatter interface {
	io.Closer
	Write(v interface{}, fields ...string) error
}

type CSVFormatter struct {
	w *csv.Writer
}

func NewCSVFormatter(writer io.Writer, headers []string) (*CSVFormatter, error) {
	w := csv.NewWriter(writer)
	err := w.Write(headers)
	if err != nil {
		return nil, err
	}
	return &CSVFormatter{
		w: csv.NewWriter(writer),
	}, nil
}

func (f *CSVFormatter) Write(v interface{}, fields ...string) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to serialize v: %v", err)
	}

	r := append(fields, string(b))
	return f.w.Write(r)

}

func (f *CSVFormatter) Close() error {
	f.w.Flush()
	return nil
}

type JSONFormatter struct {
	w *json.Encoder
}

func NewJSONFormatter(writer io.Writer, headers []string) (*JSONFormatter, error) {
	return nil, fmt.Errorf("not supported")
}

func (f *JSONFormatter) Write(v interface{}, fields ...string) error {
	return fmt.Errorf("not supported")

}

func (f *JSONFormatter) Close() error {
	return nil
}
