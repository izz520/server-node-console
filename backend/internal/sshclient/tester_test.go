package sshclient

import (
	"reflect"
	"testing"
)

func TestLineCallbackWriterEmitsCompleteLines(t *testing.T) {
	var lines []string
	writer := newLineCallbackWriter(func(line string) {
		lines = append(lines, line)
	})

	if _, err := writer.Write([]byte("one\ntwo\r\nthree")); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	writer.Flush()

	want := []string{"one", "two", "three"}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("expected %v, got %v", want, lines)
	}
}

func TestLineCallbackWriterSkipsBlankLines(t *testing.T) {
	var lines []string
	writer := newLineCallbackWriter(func(line string) {
		lines = append(lines, line)
	})

	if _, err := writer.Write([]byte("\n  \nvalue\n")); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	writer.Flush()

	want := []string{"value"}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("expected %v, got %v", want, lines)
	}
}
