package room

import (
	"sync"

	"github.com/pion/webrtc/v4"
)

// Participant represents one connected browser client within a Room. PC is
// the single PeerConnection used both to receive this participant's own
// published track(s) and to send tracks forwarded from other participants.
type Participant struct {
	ID string
	PC *webrtc.PeerConnection
}

// Room holds the participants currently in a single call.
type Room struct {
	mu           sync.Mutex
	participants map[string]*Participant
}

func newRoom() *Room {
	return &Room{participants: make(map[string]*Participant)}
}

// Join adds p to the room.
func (r *Room) Join(p *Participant) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.participants[p.ID] = p
}

// Leave removes participantID from the room. It returns true if the room
// has no participants left.
func (r *Room) Leave(participantID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.participants, participantID)
	return len(r.participants) == 0
}

// Count returns the number of participants currently in the room.
func (r *Room) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.participants)
}
