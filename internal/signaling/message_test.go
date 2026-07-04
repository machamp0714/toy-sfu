package signaling

import (
	"encoding/json"
	"testing"
)

func TestMessageJoinRoundTrip(t *testing.T) {
	msg := Message{Type: TypeJoin, Room: "room-1"}

	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got Message
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Type != TypeJoin || got.Room != "room-1" {
		t.Fatalf("got %+v, want Type=%q Room=%q", got, TypeJoin, "room-1")
	}
}

func TestMessageOfferRoundTrip(t *testing.T) {
	msg := Message{Type: TypeOffer, SDP: "v=0..."}

	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got Message
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Type != TypeOffer || got.SDP != "v=0..." {
		t.Fatalf("got %+v, want Type=%q SDP=%q", got, TypeOffer, "v=0...")
	}
}

func TestMessageICECandidateRoundTrip(t *testing.T) {
	mid := "0"
	var mlineIndex uint16 = 0
	msg := Message{
		Type: TypeICECandidate,
		Candidate: &ICECandidate{
			Candidate:     "candidate:1 1 UDP 2122260223 10.0.0.1 54400 typ host",
			SDPMid:        &mid,
			SDPMLineIndex: &mlineIndex,
		},
	}

	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got Message
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Candidate == nil {
		t.Fatalf("got Candidate = nil")
	}
	if got.Candidate.Candidate != msg.Candidate.Candidate {
		t.Fatalf("got Candidate.Candidate = %q, want %q", got.Candidate.Candidate, msg.Candidate.Candidate)
	}
	if got.Candidate.SDPMid == nil || *got.Candidate.SDPMid != mid {
		t.Fatalf("got SDPMid = %v, want %q", got.Candidate.SDPMid, mid)
	}
}

func TestMessageUnknownTypeUnmarshals(t *testing.T) {
	var got Message
	if err := json.Unmarshal([]byte(`{"type":"bogus"}`), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Type != MessageType("bogus") {
		t.Fatalf("got Type = %q, want %q", got.Type, "bogus")
	}
}
