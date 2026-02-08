package run

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
)

type mockRunShowService struct {
	run    *tfe.Run
	runErr error
}

func (m *mockRunShowService) ListRuns(_ context.Context, _ string, _ *tfe.RunListOptions) (*tfe.RunList, error) {
	return nil, nil
}

func (m *mockRunShowService) ReadRun(_ context.Context, _ string) (*tfe.Run, error) {
	if m.runErr != nil {
		return nil, m.runErr
	}
	return m.run, nil
}

func TestRunShow_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	mock := &mockRunShowService{
		run: &tfe.Run{
			ID:               "run-abc123",
			Status:           tfe.RunApplied,
			Message:          "Apply complete",
			TerraformVersion: "1.5.0",
			HasChanges:       true,
			IsDestroy:        false,
			CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
		},
	}

	cmd := newCmdRunShowWith(func() (client.RunService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"run-abc123"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunShow_Table_Output(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	mock := &mockRunShowService{
		run: &tfe.Run{
			ID:               "run-abc123",
			Status:           tfe.RunApplied,
			Message:          "Apply complete",
			TerraformVersion: "1.5.0",
			HasChanges:       true,
			IsDestroy:        false,
			CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunShow(mock, "run-abc123")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"ID:", "run-abc123", "Status:", "applied", "Message:", "Apply complete", "Terraform Version:", "1.5.0", "Has Changes:", "true", "Is Destroy:", "false"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestRunShow_Table_WithTimestamps(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	mock := &mockRunShowService{
		run: &tfe.Run{
			ID:               "run-abc123",
			Status:           tfe.RunApplied,
			Message:          "Apply complete",
			TerraformVersion: "1.5.0",
			HasChanges:       true,
			CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			StatusTimestamps: &tfe.RunStatusTimestamps{
				AppliedAt: time.Date(2024, 3, 15, 12, 5, 0, 0, time.UTC),
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunShow(mock, "run-abc123")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, "Applied At:") {
		t.Errorf("expected 'Applied At:' in output, got:\n%s", got)
	}
}

func TestRunShow_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)

	mock := &mockRunShowService{
		run: &tfe.Run{
			ID:               "run-abc123",
			Status:           tfe.RunApplied,
			Message:          "Apply complete",
			TerraformVersion: "1.5.0",
			HasChanges:       true,
			IsDestroy:        false,
			CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunShow(mock, "run-abc123")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{`"id": "run-abc123"`, `"status": "applied"`, `"has_changes": true`, `"is_destroy": false`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, got)
		}
	}
}

func TestRunShow_ClientError(t *testing.T) {
	viper.Reset()

	cmd := newCmdRunShowWith(func() (client.RunService, error) {
		return nil, fmt.Errorf("token missing")
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"run-abc123"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "token missing") {
		t.Errorf("expected 'token missing' error, got: %v", err)
	}
}

func TestRunShow_ReadError(t *testing.T) {
	viper.Reset()

	mock := &mockRunShowService{
		runErr: fmt.Errorf("run not found"),
	}

	cmd := newCmdRunShowWith(func() (client.RunService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"run-nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "run not found") {
		t.Errorf("expected 'run not found' error, got: %v", err)
	}
}

func TestRunShow_NoArgs(t *testing.T) {
	viper.Reset()

	cmd := newCmdRunShowWith(func() (client.RunService, error) {
		return &mockRunShowService{}, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
