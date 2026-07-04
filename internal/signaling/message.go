package signaling

// MessageType identifies the kind of payload carried by a Message.
type MessageType string

const (
	TypeJoin         MessageType = "join"
	TypeOffer        MessageType = "offer"
	TypeAnswer       MessageType = "answer"
	TypeICECandidate MessageType = "ice-candidate"
)

// ICECandidate mirrors the JSON shape of webrtc.ICECandidateInit so this
// package has no dependency on Pion.
type ICECandidate struct {
	Candidate        string  `json:"candidate"`
	SDPMid           *string `json:"sdpMid,omitempty"`
	SDPMLineIndex    *uint16 `json:"sdpMLineIndex,omitempty"`
	UsernameFragment *string `json:"usernameFragment,omitempty"`
}

// Message is the envelope for every WebSocket message exchanged between the
// browser client and the SFU. Only the fields relevant to Type are
// populated; the rest are left at their zero value.
type Message struct {
	Type      MessageType   `json:"type"`
	Room      string        `json:"room,omitempty"`
	SDP       string        `json:"sdp,omitempty"`
	Candidate *ICECandidate `json:"candidate,omitempty"`
}
