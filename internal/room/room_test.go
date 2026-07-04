package room

import (
	"testing"

	"github.com/pion/webrtc/v4"
)

func newTestParticipant(t *testing.T, id string) *Participant {
	t.Helper()
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatalf("NewPeerConnection: %v", err)
	}
	t.Cleanup(func() { _ = pc.Close() })
	return &Participant{ID: id, PC: pc}
}

func TestRoomJoinAddsParticipant(t *testing.T) {
	r := newRoom()
	p := newTestParticipant(t, "p1")

	r.Join(p)

	if got := r.Count(); got != 1 {
		t.Fatalf("Count() = %d, want 1", got)
	}
}

func TestRoomLeaveRemovesParticipant(t *testing.T) {
	r := newRoom()
	r.Join(newTestParticipant(t, "p1"))
	r.Join(newTestParticipant(t, "p2"))

	empty := r.Leave("p1")

	if empty {
		t.Fatalf(`Leave("p1") reported room empty, want not empty (p2 remains)`)
	}
	if got := r.Count(); got != 1 {
		t.Fatalf("Count() = %d, want 1", got)
	}
}

func TestRoomLeaveLastParticipantReportsEmpty(t *testing.T) {
	r := newRoom()
	r.Join(newTestParticipant(t, "p1"))

	empty := r.Leave("p1")

	if !empty {
		t.Fatalf(`Leave("p1") reported room not empty, want empty`)
	}
}
