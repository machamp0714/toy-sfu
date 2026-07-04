package signaling

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"

	"github.com/machamp0714/toy-sfu/internal/room"
)

var upgrader = websocket.Upgrader{
	// This is a local learning project only, not a production service:
	// any origin is accepted so it's easy to open the test client from
	// multiple browser tabs/windows without extra configuration.
	CheckOrigin: func(r *http.Request) bool { return true },
}

var iceServers = []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}}

var participantSeq atomic.Int64

func newParticipantID() string {
	return fmt.Sprintf("p-%d", participantSeq.Add(1))
}

// safeConn serializes writes to a *websocket.Conn. gorilla/websocket allows
// only one concurrent writer, but Pion's OnICECandidate and
// OnNegotiationNeeded callbacks fire on their own goroutines, so writes
// triggered by those and by the read loop below must be locked against
// each other.
type safeConn struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func (c *safeConn) writeJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}

// Handler upgrades incoming HTTP requests to WebSocket connections and
// drives one Participant's signaling for the lifetime of that connection.
type Handler struct {
	manager *room.Manager
}

// NewHandler returns a Handler backed by manager.
func NewHandler(manager *room.Manager) *Handler {
	return &Handler{manager: manager}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}
	defer wsConn.Close()

	var joinMsg Message
	if err := wsConn.ReadJSON(&joinMsg); err != nil {
		log.Printf("read join message: %v", err)
		return
	}
	if joinMsg.Type != TypeJoin || joinMsg.Room == "" {
		log.Printf("first message was not a valid join: %+v", joinMsg)
		return
	}

	participantID := newParticipantID()
	rm := h.manager.GetOrCreate(joinMsg.Room)
	conn := &safeConn{conn: wsConn}

	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{ICEServers: iceServers})
	if err != nil {
		log.Printf("participant %s: NewPeerConnection: %v", participantID, err)
		return
	}
	defer pc.Close()

	if _, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
		log.Printf("participant %s: AddTransceiverFromKind(audio): %v", participantID, err)
		return
	}
	if _, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
		log.Printf("participant %s: AddTransceiverFromKind(video): %v", participantID, err)
		return
	}

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return // nil marks end-of-gathering; nothing to send
		}
		init := c.ToJSON()
		err := conn.writeJSON(Message{Type: TypeICECandidate, Candidate: &ICECandidate{
			Candidate:        init.Candidate,
			SDPMid:           init.SDPMid,
			SDPMLineIndex:    init.SDPMLineIndex,
			UsernameFragment: init.UsernameFragment,
		}})
		if err != nil {
			log.Printf("participant %s: send ice-candidate: %v", participantID, err)
		}
	})

	pc.OnNegotiationNeeded(func() {
		offer, err := pc.CreateOffer(nil)
		if err != nil {
			log.Printf("participant %s: CreateOffer: %v", participantID, err)
			return
		}
		if err := pc.SetLocalDescription(offer); err != nil {
			log.Printf("participant %s: SetLocalDescription(offer): %v", participantID, err)
			return
		}
		if err := conn.writeJSON(Message{Type: TypeOffer, SDP: offer.SDP}); err != nil {
			log.Printf("participant %s: send offer: %v", participantID, err)
		}
	})

	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		rm.Publish(participantID, track)
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("participant %s: connection state = %s", participantID, state)
	})

	rm.Join(&room.Participant{ID: participantID, PC: pc})
	defer h.manager.Leave(joinMsg.Room, participantID)

	for {
		var msg Message
		if err := wsConn.ReadJSON(&msg); err != nil {
			log.Printf("participant %s: read message: %v", participantID, err)
			return
		}

		switch msg.Type {
		case TypeAnswer:
			err := pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: msg.SDP})
			if err != nil {
				log.Printf("participant %s: SetRemoteDescription(answer): %v", participantID, err)
			}
		case TypeICECandidate:
			if msg.Candidate == nil {
				continue
			}
			err := pc.AddICECandidate(webrtc.ICECandidateInit{
				Candidate:        msg.Candidate.Candidate,
				SDPMid:           msg.Candidate.SDPMid,
				SDPMLineIndex:    msg.Candidate.SDPMLineIndex,
				UsernameFragment: msg.Candidate.UsernameFragment,
			})
			if err != nil {
				log.Printf("participant %s: AddICECandidate: %v", participantID, err)
			}
		default:
			log.Printf("participant %s: unexpected message type after join: %q", participantID, msg.Type)
		}
	}
}
