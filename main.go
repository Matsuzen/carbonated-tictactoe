package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	lobby := newLobby()
	go lobby.run()

	r := makeMuxHandler(lobby)

	server := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:8090",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(server.ListenAndServe())

}

func makeMuxHandler(lobby *Lobby) http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", serveHome)
	muxRouter.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(lobby, w, r)
	})
	http.Handle("/", muxRouter)

	return muxRouter
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}
