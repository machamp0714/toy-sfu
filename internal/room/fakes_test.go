package room

import (
	"io"
	"sync"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

// fakeRemoteTrack implements remoteTrack for tests, without requiring a real
// PeerConnection/ICE handshake. It yields the configured packets in order,
// then returns io.EOF forever (mirroring what a closed PeerConnection's
// TrackRemote.ReadRTP does).
type fakeRemoteTrack struct {
	id       string
	streamID string
	codec    webrtc.RTPCodecParameters
	packets  []*rtp.Packet

	mu   sync.Mutex
	next int
}

func (f *fakeRemoteTrack) ID() string                       { return f.id }
func (f *fakeRemoteTrack) StreamID() string                 { return f.streamID }
func (f *fakeRemoteTrack) Codec() webrtc.RTPCodecParameters { return f.codec }

func (f *fakeRemoteTrack) ReadRTP() (*rtp.Packet, interceptor.Attributes, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.next >= len(f.packets) {
		return nil, nil, io.EOF
	}
	pkt := f.packets[f.next]
	f.next++
	return pkt, nil, nil
}

// fakeLocalTrackWriter implements localTrackWriter for tests, recording
// every packet it receives.
type fakeLocalTrackWriter struct {
	mu      sync.Mutex
	written []*rtp.Packet
}

func (f *fakeLocalTrackWriter) WriteRTP(p *rtp.Packet) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.written = append(f.written, p)
	return nil
}

func (f *fakeLocalTrackWriter) Written() []*rtp.Packet {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*rtp.Packet, len(f.written))
	copy(out, f.written)
	return out
}
