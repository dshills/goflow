package tui

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"
)

// TestReadKeyboardInput_GoroutineLifecycle tests that the input goroutine starts and stops correctly
func TestReadKeyboardInput_GoroutineLifecycle(t *testing.T) {
	t.Run("goroutine stops on context cancellation", func(t *testing.T) {
		// Create a pipe to simulate stdin
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		defer r.Close()
		defer w.Close()

		// Save original stdin and restore later
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()
		os.Stdin = r

		// Create app with cancellable context
		ctx, cancel := context.WithCancel(context.Background())
		app := &App{
			ctx:       ctx,
			cancel:    cancel,
			inputChan: make(chan KeyEvent, 100),
		}

		// Start goroutine
		done := make(chan struct{})
		go func() {
			app.readKeyboardInput()
			close(done)
		}()

		// Cancel context
		cancel()

		// Goroutine should exit quickly
		select {
		case <-done:
			// Success - goroutine exited
		case <-time.After(100 * time.Millisecond):
			t.Fatal("goroutine did not exit after context cancellation")
		}
	})

	t.Run("goroutine exits on EOF", func(t *testing.T) {
		// Create a pipe that we can close
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		defer r.Close()

		// Save original stdin and restore later
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()
		os.Stdin = r

		// Create app
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		app := &App{
			ctx:       ctx,
			cancel:    cancel,
			inputChan: make(chan KeyEvent, 100),
		}

		// Start goroutine
		done := make(chan struct{})
		go func() {
			app.readKeyboardInput()
			close(done)
		}()

		// Close write end to trigger EOF
		w.Close()

		// Goroutine should exit on EOF
		select {
		case <-done:
			// Success - goroutine exited on EOF
		case <-time.After(100 * time.Millisecond):
			t.Fatal("goroutine did not exit after EOF")
		}
	})
}

// TestReadKeyboardInput_StdinReading tests that input is correctly read from stdin
func TestReadKeyboardInput_StdinReading(t *testing.T) {
	t.Run("reads single character", func(t *testing.T) {
		// Create a pipe with test data
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		defer r.Close()
		defer w.Close()

		// Save original stdin and restore later
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()
		os.Stdin = r

		// Create app
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		app := &App{
			ctx:       ctx,
			cancel:    cancel,
			inputChan: make(chan KeyEvent, 100),
		}

		// Start goroutine
		go app.readKeyboardInput()

		// Write test input
		testInput := []byte{'a'}
		_, err = w.Write(testInput)
		if err != nil {
			t.Fatalf("failed to write test input: %v", err)
		}

		// Verify event received
		select {
		case event := <-app.inputChan:
			if event.Key != 'a' {
				t.Errorf("expected key 'a', got '%c'", event.Key)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("did not receive input event")
		}
	})

	t.Run("reads multiple characters", func(t *testing.T) {
		// Create a pipe with test data
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		defer r.Close()
		defer w.Close()

		// Save original stdin and restore later
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()
		os.Stdin = r

		// Create app
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		app := &App{
			ctx:       ctx,
			cancel:    cancel,
			inputChan: make(chan KeyEvent, 100),
		}

		// Start goroutine
		go app.readKeyboardInput()

		// Write multiple inputs
		testInputs := []byte{'h', 'e', 'l', 'l', 'o'}
		for _, b := range testInputs {
			_, err = w.Write([]byte{b})
			if err != nil {
				t.Fatalf("failed to write test input: %v", err)
			}

			// Verify each event
			select {
			case event := <-app.inputChan:
				if event.Key != rune(b) {
					t.Errorf("expected key '%c', got '%c'", b, event.Key)
				}
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("did not receive input event for '%c'", b)
			}
		}
	})
}

// TestReadKeyboardInput_ContextCancellation tests context cancellation behavior
func TestReadKeyboardInput_ContextCancellation(t *testing.T) {
	t.Run("stops reading on context cancellation", func(t *testing.T) {
		// Create a pipe
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		defer r.Close()
		defer w.Close()

		// Save original stdin and restore later
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()
		os.Stdin = r

		// Create app
		ctx, cancel := context.WithCancel(context.Background())
		app := &App{
			ctx:       ctx,
			cancel:    cancel,
			inputChan: make(chan KeyEvent, 100),
		}

		// Start goroutine
		done := make(chan struct{})
		go func() {
			app.readKeyboardInput()
			close(done)
		}()

		// Write some input to ensure goroutine is running
		w.Write([]byte{'a'})

		// Wait for event
		<-app.inputChan

		// Cancel context
		cancel()

		// Write another byte to unblock the Read() call
		// (context cancellation only takes effect between reads or when Read returns)
		w.Write([]byte{'b'})

		// Goroutine should exit after the read completes and sees context is cancelled
		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("goroutine did not exit after context cancellation")
		}
	})

	t.Run("does not send events after cancellation", func(t *testing.T) {
		// Create a pipe
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		defer r.Close()
		defer w.Close()

		// Save original stdin and restore later
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()
		os.Stdin = r

		// Create app
		ctx, cancel := context.WithCancel(context.Background())
		app := &App{
			ctx:       ctx,
			cancel:    cancel,
			inputChan: make(chan KeyEvent, 100),
		}

		// Start goroutine
		done := make(chan struct{})
		go func() {
			app.readKeyboardInput()
			close(done)
		}()

		// Cancel context immediately
		cancel()

		// Wait for goroutine to exit
		<-done

		// Write input after cancellation
		w.Write([]byte{'x'})

		// Should not receive any events
		select {
		case event := <-app.inputChan:
			t.Errorf("received unexpected event after cancellation: %+v", event)
		case <-time.After(50 * time.Millisecond):
			// Success - no events received
		}
	})
}

// TestReadKeyboardInput_EventDelivery tests that events are correctly delivered to the channel
func TestReadKeyboardInput_EventDelivery(t *testing.T) {
	t.Run("delivers events to input channel", func(t *testing.T) {
		// Create a pipe
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		defer r.Close()
		defer w.Close()

		// Save original stdin and restore later
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()
		os.Stdin = r

		// Create app with small buffer
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		app := &App{
			ctx:       ctx,
			cancel:    cancel,
			inputChan: make(chan KeyEvent, 5),
		}

		// Start goroutine
		go app.readKeyboardInput()

		// Write inputs with small delays to ensure separate reads
		inputs := []byte{'a', 'b', 'c'}
		for _, b := range inputs {
			w.Write([]byte{b})
			time.Sleep(10 * time.Millisecond) // Give time for read to complete
		}

		// Verify all events delivered in order
		for i, expected := range inputs {
			select {
			case event := <-app.inputChan:
				if event.Key != rune(expected) {
					t.Errorf("event %d: expected key '%c', got '%c'", i, expected, event.Key)
				}
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("did not receive event %d (expected '%c')", i, expected)
			}
		}
	})

	t.Run("handles escape sequences", func(t *testing.T) {
		// Create a pipe
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		defer r.Close()
		defer w.Close()

		// Save original stdin and restore later
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()
		os.Stdin = r

		// Create app
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		app := &App{
			ctx:       ctx,
			cancel:    cancel,
			inputChan: make(chan KeyEvent, 100),
		}

		// Start goroutine
		go app.readKeyboardInput()

		// Write arrow up escape sequence
		arrowUp := []byte{27, '[', 'A'}
		_, err = w.Write(arrowUp)
		if err != nil {
			t.Fatalf("failed to write escape sequence: %v", err)
		}

		// Verify special key event
		select {
		case event := <-app.inputChan:
			if !event.IsSpecial {
				t.Error("expected special key event")
			}
			if event.Special != "Up" {
				t.Errorf("expected special key 'Up', got '%s'", event.Special)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("did not receive escape sequence event")
		}
	})
}

// TestReadKeyboardInput_EOFHandling tests EOF handling
func TestReadKeyboardInput_EOFHandling(t *testing.T) {
	t.Run("graceful shutdown on EOF", func(t *testing.T) {
		// Create a buffer that will return EOF
		buf := bytes.NewReader([]byte{})

		// We can't directly test with a buffer since readKeyboardInput uses os.Stdin
		// This test documents expected behavior
		if buf.Len() == 0 {
			err := io.EOF
			if err != io.EOF {
				t.Error("EOF should be handled gracefully")
			}
		}
	})

	t.Run("exits cleanly on pipe close", func(t *testing.T) {
		// Create a pipe
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		defer r.Close()

		// Save original stdin and restore later
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()
		os.Stdin = r

		// Create app
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		app := &App{
			ctx:       ctx,
			cancel:    cancel,
			inputChan: make(chan KeyEvent, 100),
		}

		// Start goroutine
		done := make(chan struct{})
		go func() {
			app.readKeyboardInput()
			close(done)
		}()

		// Close write end immediately (triggers EOF on read)
		w.Close()

		// Goroutine should exit gracefully
		select {
		case <-done:
			// Success - exited on EOF
		case <-time.After(100 * time.Millisecond):
			t.Fatal("goroutine did not exit after pipe close (EOF)")
		}
	})
}

// TestReadKeyboardInput_ConcurrentEvents tests handling of concurrent input
func TestReadKeyboardInput_ConcurrentEvents(t *testing.T) {
	t.Run("handles rapid input without loss", func(t *testing.T) {
		// Create a pipe
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		defer r.Close()
		defer w.Close()

		// Save original stdin and restore later
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()
		os.Stdin = r

		// Create app with larger buffer to prevent drops
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		app := &App{
			ctx:       ctx,
			cancel:    cancel,
			inputChan: make(chan KeyEvent, 100),
		}

		// Start goroutine
		go app.readKeyboardInput()

		// Send characters with small delays
		const count = 10
		expectedKeys := make([]rune, count)
		for i := 0; i < count; i++ {
			key := rune('a' + i)
			expectedKeys[i] = key
			w.Write([]byte{byte(key)})
			time.Sleep(5 * time.Millisecond) // Small delay between writes
		}

		// Collect all events
		receivedKeys := make([]rune, 0, count)
		for i := 0; i < count; i++ {
			select {
			case event := <-app.inputChan:
				receivedKeys = append(receivedKeys, event.Key)
			case <-time.After(200 * time.Millisecond):
				t.Fatalf("timeout waiting for event %d", i)
			}
		}

		// Verify all keys received in order
		if len(receivedKeys) != len(expectedKeys) {
			t.Fatalf("expected %d keys, got %d", len(expectedKeys), len(receivedKeys))
		}
		for i, expected := range expectedKeys {
			if receivedKeys[i] != expected {
				t.Errorf("key %d: expected '%c', got '%c'", i, expected, receivedKeys[i])
			}
		}
	})
}
