package room

import (
	"sync"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

// Participant represents one connected browser client within a Room. PC is
// the single PeerConnection used both to receive this participant's own
// published track(s) and to send tracks forwarded from other participants.
type Participant struct {
	ID string
	PC *webrtc.PeerConnection
}

// remoteTrack is the subset of *webrtc.TrackRemote that Room needs to fan a
// published track out to subscribers. Defining it here (instead of
// depending on *webrtc.TrackRemote directly) lets tests substitute a fake
// track without a real PeerConnection/ICE handshake.
type remoteTrack interface {
	ID() string
	StreamID() string
	Codec() webrtc.RTPCodecParameters
	ReadRTP() (*rtp.Packet, interceptor.Attributes, error)
}

// localTrackWriter is the subset of *webrtc.TrackLocalStaticRTP that the
// forwarding loop needs.
type localTrackWriter interface {
	WriteRTP(p *rtp.Packet) error
}

// publishedTrack is one track published by one participant, together with
// the set of other participants currently subscribed to it.
type publishedTrack struct {
	publisherID string
	track       remoteTrack
	subscribers map[string]localTrackWriter
}

// Room holds the participants and published tracks for a single call.
type Room struct {
	mu           sync.Mutex
	participants map[string]*Participant
	published    []*publishedTrack
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

// Leave removes participantID from the room, and from any published
// track's subscriber set. It returns true if the room has no participants
// left.
func (r *Room) Leave(participantID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.participants, participantID)

	remaining := r.published[:0]
	for _, pt := range r.published {
		if pt.publisherID == participantID {
			continue
		}
		delete(pt.subscribers, participantID)
		remaining = append(remaining, pt)
	}
	r.published = remaining

	return len(r.participants) == 0
}

// Count returns the number of participants currently in the room.
func (r *Room) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.participants)
}

// newLocalTrackFor builds a local forwarding track for pt's track and adds it as a
// sender on p's PeerConnection. It does not register p in pt.subscribers or take
// any lock -- callers do that under whatever locking discipline they need.
func newLocalTrackFor(pt *publishedTrack, p *Participant) (localTrackWriter, error) {
	local, err := webrtc.NewTrackLocalStaticRTP(pt.track.Codec().RTPCodecCapability, pt.track.ID(), pt.track.StreamID())
	if err != nil {
		return nil, err
	}
	if _, err := p.PC.AddTrack(local); err != nil {
		return nil, err
	}
	return local, nil
}

// Publish registers track as published by publisherID and subscribes every
// other current participant to it, by adding a forwarding local track to
// their PeerConnection (this alone is enough to make Pion fire its
// OnNegotiationNeeded callback for them). It returns the participant IDs
// that were subscribed — mainly so tests can observe the fan-out target
// selection.
func (r *Room) Publish(publisherID string, track remoteTrack) []string {
	r.mu.Lock()

	pt := &publishedTrack{
		publisherID: publisherID,
		track:       track,
		subscribers: make(map[string]localTrackWriter),
	}

	var subscribed []string
	for id, p := range r.participants {
		if id == publisherID {
			continue
		}
		local, err := newLocalTrackFor(pt, p)
		if err != nil {
			continue
		}
		pt.subscribers[id] = local
		subscribed = append(subscribed, id)
	}
	r.published = append(r.published, pt)

	r.mu.Unlock()

	go r.forward(pt)

	return subscribed
}

// forward reads RTP packets from pt.track and writes them to every current
// subscriber, until the track's ReadRTP call returns an error — which
// happens once the publisher's PeerConnection is closed on Leave.
func (r *Room) forward(pt *publishedTrack) {
	for {
		pkt, _, err := pt.track.ReadRTP()
		if err != nil {
			return
		}

		r.mu.Lock()
		writers := make([]localTrackWriter, 0, len(pt.subscribers))
		for _, w := range pt.subscribers {
			writers = append(writers, w)
		}
		r.mu.Unlock()

		for _, w := range writers {
			_ = w.WriteRTP(pkt) // best-effort: a write error just means that subscriber is going away
		}
	}
}
