package llmops

import (
	"encoding/json"
	"testing"

	"github.com/adm/pkg/battle"
)

func TestDecodeJSONStripsFences(t *testing.T) {
	raw := "```json\n{\"payload\":\"x\",\"technique\":\"RT-001\",\"endpoint\":\"chat\",\"reason\":\"r\",\"strategy\":\"s\"}\n```"
	var out MutateResult
	if err := decodeJSON(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.Payload != "x" || out.Technique != "RT-001" {
		t.Fatalf("got %+v", out)
	}
}

func TestFilterAgentTargets(t *testing.T) {
	got := filterAgentTargets([]string{"executor", "gateway", "EXECUTOR", "redis", "planner"})
	if len(got) != 2 || got[0] != "executor" || got[1] != "planner" {
		t.Fatalf("got %v", got)
	}
}

func TestDefaultTriage(t *testing.T) {
	tr := DefaultTriage(battle.Event{
		SessionID: "sess-1",
		Technique: "RT-004",
		Target:    "executor",
		Severity:  5,
	})
	if !tr.Revoke || len(tr.RestartTargets) != 1 || tr.RestartTargets[0] != "executor" {
		t.Fatalf("got %+v", tr)
	}
	if tr.Summary == "" {
		t.Fatal("expected summary")
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	if normalizeEndpoint("tool") != "tool" {
		t.Fatal("tool")
	}
	if normalizeEndpoint("CHAT") != "chat" {
		t.Fatal("chat")
	}
}

func TestMutateResultRoundTrip(t *testing.T) {
	b, _ := json.Marshal(MutateResult{
		Payload: "p", Technique: "RT-002", Endpoint: "tool",
		Target: "executor", Reason: "r", Strategy: "s",
	})
	var out MutateResult
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Endpoint != "tool" {
		t.Fatal(out)
	}
}
