package workbench

import (
	"os/exec"
	"testing"
)

func TestRunApprovedCommandRejectsMissingApproval(t *testing.T) {
	_, err := RunApprovedCommand(t.TempDir(), CommandRequest{
		Command: "git",
		Args:    []string{"status", "--short"},
	})
	if err == nil {
		t.Fatal("expected missing approval to be rejected")
	}
}

func TestRunApprovedCommandRejectsUnknownCommand(t *testing.T) {
	_, err := RunApprovedCommand(t.TempDir(), CommandRequest{
		Command:  "git",
		Args:     []string{"reset", "--hard"},
		Approved: true,
	})
	if err == nil {
		t.Fatal("expected non-allowlisted command to be rejected")
	}
}

func TestRunApprovedCommandRunsAllowlistedGitStatus(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	dir := t.TempDir()
	runGit(t, dir, "init")

	result, err := RunApprovedCommand(dir, CommandRequest{
		Command:  "git",
		Args:     []string{"status", "--short"},
		Approved: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected zero exit code, got %d: %s", result.ExitCode, result.Stderr)
	}
	if result.Command != "git" || result.Directory == "" {
		t.Fatalf("unexpected result: %#v", result)
	}
}
