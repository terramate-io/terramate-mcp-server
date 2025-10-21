package tools

import (
	"testing"

	"github.com/terramate-io/terramate-mcp-server/sdk/terramate"
)

func TestNew(t *testing.T) {
	c, err := terramate.NewClient("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	th := New(c)
	if th == nil {
		t.Fatal("expected non-nil ToolHandlers")
		return
	}
	if th.tmcClient == nil {
		t.Fatal("expected tmcClient to be set")
	}
}

func TestTools(t *testing.T) {
	c, err := terramate.NewClient("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	th := New(c)
	tools := th.Tools()
	if len(tools) == 0 {
		t.Fatal("expected at least one tool")
	}
	// Verify authentication tool is registered
	found := false
	for _, tool := range tools {
		if tool.Tool.Name == "tmc_authenticate" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected tmc_authenticate tool to be registered")
	}
}
