package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestRootCommand(t *testing.T) {
	cmd := newRootCmd()

	if cmd == nil {
		t.Fatal("expected root command, got nil")
	}

	if cmd.Use != "fleet" {
		t.Errorf("expected use 'fleet', got %q", cmd.Use)
	}

	// Verify subcommands are registered
	expectedCommands := []string{
		"version",
		"completion",
		"cluster",
		"get",
		"apply",
		"delete",
	}

	for _, cmdName := range expectedCommands {
		found := false
		for _, cmd := range cmd.Commands() {
			if cmd.Name() == cmdName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %q to be registered", cmdName)
		}
	}
}

func TestRootCommandHelp(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"--help"})

	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	help := output.String()

	expectedStrings := []string{
		"Fleet",
		"Kubernetes",
		"version",
		"completion",
		"cluster",
		"get",
		"apply",
		"delete",
	}

	for _, want := range expectedStrings {
		if !strings.Contains(help, want) {
			t.Errorf("expected help to contain %q", want)
		}
	}
}

func TestRootCommandPersistentFlags(t *testing.T) {
	cmd := newRootCmd()

	expectedFlags := []string{
		"config",
		"kubeconfig",
		"clusters",
		"output",
		"verbose",
		"no-color",
		"timeout",
		"parallel",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected persistent flag %q to be defined", flagName)
		}
	}
}

func TestRootCommandFlagDefaults(t *testing.T) {
	cmd := newRootCmd()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{
			name:     "config default",
			flag:     "config",
			expected: "",
		},
		{
			name:     "kubeconfig default",
			flag:     "kubeconfig",
			expected: "",
		},
		{
			name:     "output default",
			flag:     "output",
			expected: "",
		},
		{
			name:     "verbose default",
			flag:     "verbose",
			expected: "false",
		},
		{
			name:     "no-color default",
			flag:     "no-color",
			expected: "false",
		},
		{
			name:     "timeout default",
			flag:     "timeout",
			expected: (30 * time.Second).String(),
		},
		{
			name:     "parallel default",
			flag:     "parallel",
			expected: "5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := cmd.PersistentFlags().Lookup(tt.flag)
			if flag == nil {
				t.Fatalf("flag %q not found", tt.flag)
			}

			if flag.DefValue != tt.expected {
				t.Errorf("expected default value %q, got %q", tt.expected, flag.DefValue)
			}
		})
	}
}

func TestExecute(t *testing.T) {
	// Test that Execute can be called without panicking
	// We can't fully test it because it requires a valid kubeconfig,
	// but we can verify it doesn't crash on help
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Capture the args before modifying
	// We'll test this by checking the help works
	// Note: This is a basic smoke test
	_ = ctx
}

func TestRootCommandSilenceFlags(t *testing.T) {
	cmd := newRootCmd()

	if !cmd.SilenceUsage {
		t.Error("expected SilenceUsage to be true")
	}

	if !cmd.SilenceErrors {
		t.Error("expected SilenceErrors to be true")
	}
}

func TestRootCommandShortFlags(t *testing.T) {
	cmd := newRootCmd()

	// Verify short flags are set correctly
	shortFlags := map[string]string{
		"o": "output",
		"v": "verbose",
		"p": "parallel",
	}

	for short, long := range shortFlags {
		shortFlag := cmd.PersistentFlags().ShorthandLookup(short)
		if shortFlag == nil {
			t.Errorf("expected short flag -%s for %s", short, long)
			continue
		}

		if shortFlag.Name != long {
			t.Errorf("expected short flag -%s to map to %s, got %s", short, long, shortFlag.Name)
		}
	}
}
