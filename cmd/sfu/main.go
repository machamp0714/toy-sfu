package main

import (
	"log"
	"net/http"

	"github.com/machamp0714/toy-sfu/internal/room"
	"github.com/machamp0714/toy-sfu/internal/signaling"
)

func newMux() *http.ServeMux {
	manager := room.NewManager()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthzHandler)
	mux.Handle("/ws", signaling.NewHandler(manager))
	mux.Handle("/", http.FileServer(http.Dir("web")))
	return mux
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func main() {
	addr := ":8080"
	log.Printf("toy-sfu listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, newMux()))
}
