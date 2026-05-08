package workbench

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultCommandTimeout = 30 * time.Second
	maxCommandTimeout     = 2 * time.Minute
	maxCommandOutputBytes = 120000
)

type CommandRequest struct {
	Command        string   `json:"command"`
	Args           []string `json:"args"`
	Approved       bool     `json:"approved"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

type CommandResult struct {
	Command    string    `json:"command"`
	Args       []string  `json:"args"`
	Directory  string    `json:"directory"`
	ExitCode   int       `json:"exit_code"`
	Stdout     string    `json:"stdout"`
	Stderr     string    `json:"stderr"`
	DurationMS int64     `json:"duration_ms"`
	StartedAt  time.Time `json:"started_at"`
	TimedOut   bool      `json:"timed_out"`
	Truncated  bool      `json:"truncated"`
}

func RunApprovedCommand(root string, req CommandRequest) (CommandResult, error) {
	if !req.Approved {
		return CommandResult{}, errors.New("command approval is required")
	}
	if err := validateCommand(req.Command, req.Args); err != nil {
		return CommandResult{}, err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return CommandResult{}, err
	}

	timeout := commandTimeout(req.TimeoutSeconds)
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, req.Command, req.Args...)
	cmd.Dir = rootAbs
	var stdout, stderr cappedBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	result := CommandResult{
		Command:    req.Command,
		Args:       req.Args,
		Directory:  rootAbs,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		DurationMS: time.Since(started).Milliseconds(),
		StartedAt:  started,
		TimedOut:   ctx.Err() == context.DeadlineExceeded,
		Truncated:  stdout.Truncated() || stderr.Truncated(),
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if result.TimedOut {
		return result, nil
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return result, nil
		}
		return result, err
	}
	return result, nil
}

func validateCommand(command string, args []string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return errors.New("command is required")
	}
	for _, arg := range args {
		if strings.ContainsAny(arg, "\r\n") {
			return errors.New("command arguments cannot contain new lines")
		}
	}
	key := commandKey(command, args)
	for _, allowed := range allowedCommands() {
		if key == allowed {
			return nil
		}
	}
	return fmt.Errorf("command is not allowlisted: %s", key)
}

func commandKey(command string, args []string) string {
	parts := append([]string{strings.ToLower(filepath.Base(command))}, args...)
	return strings.Join(parts, " ")
}

func allowedCommands() []string {
	return []string{
		"git status",
		"git status --short",
		"git diff --stat",
		"git diff --name-only",
		"go test",
		"go test ./...",
	}
}

func commandTimeout(seconds int) time.Duration {
	if seconds <= 0 {
		return defaultCommandTimeout
	}
	timeout := time.Duration(seconds) * time.Second
	if timeout > maxCommandTimeout {
		return maxCommandTimeout
	}
	return timeout
}

type cappedBuffer struct {
	buffer    bytes.Buffer
	truncated bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	remaining := maxCommandOutputBytes - b.buffer.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.truncated = true
		_, _ = b.buffer.Write(p[:remaining])
		return len(p), nil
	}
	return b.buffer.Write(p)
}

func (b *cappedBuffer) String() string {
	return b.buffer.String()
}

func (b *cappedBuffer) Truncated() bool {
	return b.truncated
}
