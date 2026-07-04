package main

import (
	"log"
	"net/http"
)

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthzHandler)
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
