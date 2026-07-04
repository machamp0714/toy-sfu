package signaling

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/machamp0714/toy-sfu/internal/room"
)

func TestHandlerSendsOfferAfterJoin(t *testing.T) {
	manager := room.NewManager()
	server := httptest.NewServer(NewHandler(manager))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(Message{Type: TypeJoin, Room: "room-1"}); err != nil {
		t.Fatalf("WriteJSON(join): %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	var msg Message
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}

	if msg.Type != TypeOffer || msg.SDP == "" {
		t.Fatalf("got %+v, want an offer with a non-empty SDP", msg)
	}
}
