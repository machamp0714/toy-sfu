package room

import (
	"runtime"
	"sync"
	"testing"
)

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

// TestManagerLeaveDoesNotOrphanConcurrentJoin proves the TOCTOU fix in
// Manager.Leave: the window between "Leave observed the room as empty" and
// "Leave deletes the map entry" must not let a concurrently-joined
// participant be orphaned.
//
// The buggy version was:
//
//	if r.Leave(participantID) {
//		m.mu.Lock()
//		delete(m.rooms, name)          // unconditional
//		m.mu.Unlock()
//	}
//
// If, after r.Leave(participantID) returns true but before this delete
// runs, another goroutine calls GetOrCreate(name) (getting back the same,
// still-mapped *Room), then Joins a new participant into it, the delete
// above still removes the room from the Manager's map -- even though the
// *Room object itself now holds a live participant. That participant is
// orphaned: no future GetOrCreate(name) can ever reach it again, since a
// brand new (different) *Room gets created instead.
//
// A plain, uncoordinated dual-goroutine race (one calling Leave, the other
// calling GetOrCreate(name).Join(...)) does not reliably exercise this
// specific window: GetOrCreate and Join are two separate, unsynchronized
// calls, and in practice Leave's cleanup step almost always wins the race
// for m.mu before the other goroutine even gets to call Join. That
// generates a *different*, structurally unavoidable race (Join landing
// after the room was legitimately deleted because, at that instant, it truly
// was still empty) which even a correct fix cannot -- and is not expected
// to -- prevent, since GetOrCreate+Join are not atomic with each other.
//
// To reliably hit the *intended* window instead, this test drives the
// interleaving explicitly using direct access to the unexported mutexes
// (this file is in package room, so m.mu and r.mu are reachable):
//
//  1. Hold r.mu and m.mu so Leave's goroutine can only perform its first,
//     harmless read of m.rooms[name] once we release m.mu.
//  2. Release m.mu just long enough (yielding via runtime.Gosched) for
//     Leave's goroutine to complete that read and then block trying to
//     acquire r.mu for r.Leave(participantID).
//  3. Reacquire m.mu (Leave's goroutine cannot reach its cleanup lock yet,
//     since it is still blocked on r.mu).
//  4. Release r.mu: Leave's goroutine now runs r.Leave("p1"), observes the
//     room going empty, returns true, and then blocks trying to acquire
//     m.mu for its cleanup step (still held by us).
//  5. While Leave's goroutine is blocked there, perform the "concurrent"
//     join directly on r -- equivalent to another goroutine's
//     GetOrCreate(name).Join(...), since m.rooms[name] still == r.
//  6. Release m.mu, letting Leave's cleanup step run its guarded check.
//
// With the buggy unconditional delete, this reliably (in this environment,
// ~299/300 iterations) orphans the joined participant. With the fix's
// guard (skip the delete unless m.rooms[name] is still r and r.Count() ==
// 0), the room is correctly kept.
func TestManagerLeaveDoesNotOrphanConcurrentJoin(t *testing.T) {
	const iterations = 200
	name := "room-1"

	for i := 0; i < iterations; i++ {
		m := NewManager()
		r := m.GetOrCreate(name)
		r.Join(newTestParticipant(t, "p1"))

		r.mu.Lock() // block Leave's r.Leave(...) call until we allow it
		m.mu.Lock() // block Leave's initial "read r" until we allow it

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Leave(name, "p1")
		}()

		m.mu.Unlock() // let Leave read r (fast, uncontended), then block on r.mu
		for j := 0; j < 100; j++ {
			runtime.Gosched()
		}
		m.mu.Lock() // reacquire before Leave's cleanup step can run

		r.mu.Unlock() // let r.Leave("p1") mutate + return true; Leave then blocks on m.mu

		// Simulate the concurrent GetOrCreate(name).Join(p2): m.rooms[name]
		// is still r at this point, so joining r directly has the same
		// effect.
		r.Join(newTestParticipant(t, "p2"))

		m.mu.Unlock() // let Leave's guarded cleanup run
		wg.Wait()

		final := m.GetOrCreate(name)
		if final.Count() == 0 {
			t.Fatalf("iteration %d: p2 was orphaned -- room %q is empty after "+
				"a concurrent Join raced with Manager.Leave's cleanup", i, name)
		}
		if final != r {
			t.Fatalf("iteration %d: expected the concurrently-joined room to survive as the "+
				"same *Room instance, got a different one", i)
		}
	}
}
