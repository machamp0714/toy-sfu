package room

import "testing"

func TestManagerGetOrCreateReturnsSameRoomForSameName(t *testing.T) {
	m := NewManager()

	a := m.GetOrCreate("room-1")
	b := m.GetOrCreate("room-1")

	if a != b {
		t.Fatalf("GetOrCreate returned different rooms for the same name")
	}
}

func TestManagerGetOrCreateReturnsDifferentRoomsForDifferentNames(t *testing.T) {
	m := NewManager()

	a := m.GetOrCreate("room-1")
	b := m.GetOrCreate("room-2")

	if a == b {
		t.Fatalf("GetOrCreate returned the same room for different names")
	}
}

func TestManagerLeaveRemovesEmptyRoom(t *testing.T) {
	m := NewManager()
	original := m.GetOrCreate("room-1")
	original.Join(newTestParticipant(t, "p1"))

	m.Leave("room-1", "p1")

	again := m.GetOrCreate("room-1")
	if again == original {
		t.Fatalf("room-1 should have been removed after emptying, got the same instance back")
	}
}

func TestManagerLeaveKeepsNonEmptyRoom(t *testing.T) {
	m := NewManager()
	original := m.GetOrCreate("room-1")
	original.Join(newTestParticipant(t, "p1"))
	original.Join(newTestParticipant(t, "p2"))

	m.Leave("room-1", "p1")

	again := m.GetOrCreate("room-1")
	if again != original {
		t.Fatalf("room-1 should still exist with p2 remaining, got a different instance")
	}
}
