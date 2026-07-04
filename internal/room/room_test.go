package room

import (
	"testing"

	"github.com/pion/rtp"
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

func TestRoomPublishSubscribesOtherParticipants(t *testing.T) {
	r := newRoom()
	publisher := newTestParticipant(t, "publisher")
	subscriber := newTestParticipant(t, "subscriber")
	r.Join(publisher)
	r.Join(subscriber)

	track := &fakeRemoteTrack{
		id:       "audio",
		streamID: "stream-1",
		codec:    webrtc.RTPCodecParameters{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}},
	}

	subscribed := r.Publish("publisher", track)

	if len(subscribed) != 1 || subscribed[0] != "subscriber" {
		t.Fatalf("Publish returned subscribed = %v, want [subscriber]", subscribed)
	}
	if got := len(subscriber.PC.GetSenders()); got != 1 {
		t.Fatalf("subscriber.PC.GetSenders() = %d, want 1", got)
	}
	if got := len(publisher.PC.GetSenders()); got != 0 {
		t.Fatalf("publisher.PC.GetSenders() = %d, want 0 (publisher must not subscribe to itself)", got)
	}
}

func TestRoomForwardCopiesPacketsToAllSubscribers(t *testing.T) {
	r := newRoom()
	pkt1 := &rtp.Packet{Header: rtp.Header{SequenceNumber: 1}}
	pkt2 := &rtp.Packet{Header: rtp.Header{SequenceNumber: 2}}
	track := &fakeRemoteTrack{packets: []*rtp.Packet{pkt1, pkt2}}
	subA := &fakeLocalTrackWriter{}
	subB := &fakeLocalTrackWriter{}
	pt := &publishedTrack{
		publisherID: "publisher",
		track:       track,
		subscribers: map[string]localTrackWriter{"a": subA, "b": subB},
	}

	// Called directly (not via `go`) so the test is deterministic: fakeRemoteTrack
	// returns io.EOF after 2 packets, so forward returns promptly on its own.
	r.forward(pt)

	for name, sub := range map[string]*fakeLocalTrackWriter{"a": subA, "b": subB} {
		got := sub.Written()
		if len(got) != 2 || got[0].SequenceNumber != 1 || got[1].SequenceNumber != 2 {
			t.Fatalf("subscriber %s received %+v, want [seq 1, seq 2]", name, got)
		}
	}
}
