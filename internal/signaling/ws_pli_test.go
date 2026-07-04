package signaling

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/pion/rtcp"
)

// fakeRTCPWriter is a minimal rtcpWriter that records every packet it
// receives and starts erroring once it has recorded stopAfter of them,
// simulating a PeerConnection that has closed.
type fakeRTCPWriter struct {
	mu        sync.Mutex
	written   []rtcp.Packet
	stopAfter int
}

func (f *fakeRTCPWriter) WriteRTCP(pkts []rtcp.Packet) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.written) >= f.stopAfter {
		return errors.New("closed")
	}
	f.written = append(f.written, pkts...)
	return nil
}

func TestSendPeriodicPLI(t *testing.T) {
	const (
		mediaSSRC = uint32(12345)
		stopAfter = 4
	)
	fake := &fakeRTCPWriter{stopAfter: stopAfter}

	// Called directly (not via `go`) with a tiny interval so the test is
	// fast and deterministic: sendPeriodicPLI returns on its own once the
	// fake starts erroring.
	sendPeriodicPLI(fake, mediaSSRC, time.Millisecond)

	fake.mu.Lock()
	defer fake.mu.Unlock()

	if len(fake.written) != stopAfter {
		t.Fatalf("got %d PLI packets, want %d", len(fake.written), stopAfter)
	}
	for i, pkt := range fake.written {
		pli, ok := pkt.(*rtcp.PictureLossIndication)
		if !ok {
			t.Fatalf("packet %d: got %T, want *rtcp.PictureLossIndication", i, pkt)
		}
		if pli.MediaSSRC != mediaSSRC {
			t.Fatalf("packet %d: MediaSSRC = %d, want %d", i, pli.MediaSSRC, mediaSSRC)
		}
	}
}
