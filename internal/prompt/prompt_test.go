package prompt

import (
	"io"
	"os"
	"testing"
)

func TestConfirm_Yes(t *testing.T) {
	// Create a pipe to simulate stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() { _ = r.Close() }()

	// Replace os.Stdin temporarily
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// Write "y\n" to the pipe
	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte("y\n"))
	}()

	result, err := Confirm("Test question")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result {
		t.Error("expected true for 'y' input, got false")
	}
}

func TestConfirm_YesUpperCase(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() { _ = r.Close() }()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte("Y\n"))
	}()

	result, err := Confirm("Test question")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result {
		t.Error("expected true for 'Y' input, got false")
	}
}

func TestConfirm_No(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() { _ = r.Close() }()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte("n\n"))
	}()

	result, err := Confirm("Test question")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result {
		t.Error("expected false for 'n' input, got true")
	}
}

func TestConfirm_EmptyEnter(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() { _ = r.Close() }()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte("\n"))
	}()

	result, err := Confirm("Test question")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result {
		t.Error("expected false (default No) for empty Enter, got true")
	}
}

func TestConfirm_OtherInput(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() { _ = r.Close() }()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte("foo\n"))
	}()

	result, err := Confirm("Test question")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result {
		t.Error("expected false for other input, got true")
	}
}

func TestConfirm_Error(t *testing.T) {
	// Close the read end immediately to simulate an error
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	_ = r.Close()
	_ = w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	_, err = Confirm("Test question")
	if err == nil {
		t.Error("expected error when stdin is closed, got nil")
	}
}

func TestConfirm_MessageFormat(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	}()

	// Capture stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create output pipe: %v", err)
	}

	// Write input data and close the write end
	_, _ = w.Write([]byte("n\n"))
	_ = w.Close()

	// Set stdin and stdout
	os.Stdin = r
	os.Stdout = wOut

	// Run Confirm in a goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = Confirm("Do you want to proceed?")
		_ = wOut.Close()
	}()

	// Read captured output
	outputBytes, err := io.ReadAll(rOut)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// Wait for Confirm to finish
	<-done
	_ = r.Close()

	output := string(outputBytes)
	expectedPrompt := "Do you want to proceed? [y/N]: "
	if output != expectedPrompt {
		t.Errorf("expected prompt %q, got %q", expectedPrompt, output)
	}
}
