package installer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// NopReporter — all methods must be callable without panic.
// ---------------------------------------------------------------------------

func TestNopReporter_Header(t *testing.T) {
	var r NopReporter
	assert.NotPanics(t, func() { r.Header("header text") })
}

func TestNopReporter_Info(t *testing.T) {
	var r NopReporter
	assert.NotPanics(t, func() { r.Info("info text") })
}

func TestNopReporter_Success(t *testing.T) {
	var r NopReporter
	assert.NotPanics(t, func() { r.Success("success text") })
}

func TestNopReporter_Warn(t *testing.T) {
	var r NopReporter
	assert.NotPanics(t, func() { r.Warn("warn text") })
}

func TestNopReporter_Error(t *testing.T) {
	var r NopReporter
	assert.NotPanics(t, func() { r.Error("error text") })
}

func TestNopReporter_Muted(t *testing.T) {
	var r NopReporter
	assert.NotPanics(t, func() { r.Muted("muted text") })
}

func TestNopReporter_EmptyString(t *testing.T) {
	var r NopReporter
	assert.NotPanics(t, func() {
		r.Header("")
		r.Info("")
		r.Success("")
		r.Warn("")
		r.Error("")
		r.Muted("")
	})
}

func TestNopReporter_ImplementsReporterInterface(t *testing.T) {
	var _ Reporter = NopReporter{}
}

// ---------------------------------------------------------------------------
// ConsoleReporter — verify it satisfies the Reporter interface and that
// each method delegates to the ui package without panicking.
// ConsoleReporter writes to the terminal via the ui package; in test
// environments there is no TTY so ui output is a no-op / safe to call.
// ---------------------------------------------------------------------------

func TestConsoleReporter_ImplementsReporterInterface(t *testing.T) {
	var _ Reporter = ConsoleReporter{}
}

func TestConsoleReporter_Header_NoPanic(t *testing.T) {
	r := ConsoleReporter{}
	assert.NotPanics(t, func() { r.Header("header text") })
}

func TestConsoleReporter_Info_NoPanic(t *testing.T) {
	r := ConsoleReporter{}
	assert.NotPanics(t, func() { r.Info("info text") })
}

func TestConsoleReporter_Success_NoPanic(t *testing.T) {
	r := ConsoleReporter{}
	assert.NotPanics(t, func() { r.Success("success text") })
}

func TestConsoleReporter_Warn_NoPanic(t *testing.T) {
	r := ConsoleReporter{}
	assert.NotPanics(t, func() { r.Warn("warn text") })
}

func TestConsoleReporter_Error_NoPanic(t *testing.T) {
	r := ConsoleReporter{}
	assert.NotPanics(t, func() { r.Error("error text") })
}

func TestConsoleReporter_Muted_NoPanic(t *testing.T) {
	r := ConsoleReporter{}
	assert.NotPanics(t, func() { r.Muted("muted text") })
}

func TestConsoleReporter_MultipleMessages_NoPanic(t *testing.T) {
	r := ConsoleReporter{}
	messages := []string{"", "short", "a longer message with spaces and special chars: @#$%"}
	assert.NotPanics(t, func() {
		for _, msg := range messages {
			r.Header(msg)
			r.Info(msg)
			r.Success(msg)
			r.Warn(msg)
			r.Error(msg)
			r.Muted(msg)
		}
	})
}
