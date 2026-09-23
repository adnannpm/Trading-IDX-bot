package bot

import (
	"testing"
)

func TestLastBroadcastMessageTracking(t *testing.T) {
	channelID := "test-channel-123"

	SetLastBroadcastMessageID(channelID, "")

	if id := GetLastBroadcastMessageID(channelID); id != "" {
		t.Fatalf("expected empty id, got %s", id)
	}

	SetLastBroadcastMessageID(channelID, "msg-001")
	if id := GetLastBroadcastMessageID(channelID); id != "msg-001" {
		t.Fatalf("expected msg-001, got %s", id)
	}

	SetLastBroadcastMessageID(channelID, "msg-002")
	if id := GetLastBroadcastMessageID(channelID); id != "msg-002" {
		t.Fatalf("expected msg-002, got %s", id)
	}

	SetLastBroadcastMessageID(channelID, "")
}
