package main

type Lobby struct {
	clients map[*Client]bool
	// Inbound messages from the clients.
	broadcast chan Message

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	currentGames map[*Game]bool
}

func newLobby() *Lobby {
	return &Lobby{
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (lobby *Lobby) run() {
	for {
		select {
		case client := <-lobby.register:
			lobby.clients[client] = true
		case client := <-lobby.unregister:
			if _, ok := lobby.clients[client]; ok {
				delete(lobby.clients, client)
				close(client.send)
			}
		case message := <-lobby.broadcast:
			for client := range lobby.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(lobby.clients, client)
				}
			}
		}
	}
}
