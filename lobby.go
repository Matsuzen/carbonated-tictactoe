package main

type Lobby struct {
	clients      map[*Client]bool
	broadcast    chan ChatMessage
	register     chan *Client
	unregister   chan *Client
	createGame   chan GameLobbyUpdate
	joinGame     chan GameLobbyUpdate
	currentGames map[int]*Game
	deleteGame   chan *Game
}

type ChatMessage struct {
	Action   string `json:"action"`
	Message  string `json:"chatMessage"`
	Username string `json:"username"`
}

func newLobby() *Lobby {
	return &Lobby{
		broadcast:    make(chan ChatMessage),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		clients:      make(map[*Client]bool),
		createGame:   make(chan GameLobbyUpdate),
		joinGame:     make(chan GameLobbyUpdate),
		currentGames: make(map[int]*Game),
		deleteGame:   make(chan *Game),
	}
}

func (lobby *Lobby) run() {
	for {
		select {
		case client := <-lobby.register:
			lobby.clients[client] = true

		case client := <-lobby.unregister:
			if _, ok := lobby.clients[client]; ok {
				// Delete the user's game
				if client.game != nil {
					client.game.endGame <- true
					delete(lobby.currentGames, client.game.gameId)
				}
				delete(lobby.clients, client)
			}

		case chatMessage := <-lobby.broadcast:
			for client := range lobby.clients {
				client.chatMessage <- chatMessage
			}

		case newGame := <-lobby.createGame:
			// Create the game and send event to game creator to update his display
			createdGame := createGame(newGame.Client, newGame.GameName, len(lobby.currentGames)+1)
			go createdGame.runGame()

			lobby.currentGames[createdGame.gameId] = createdGame
			newGame.Client.game = createdGame

			// Add the player to the game's current players
			createdGame.joinGame <- newGame.Client
			newGame.PlayerCount = len(createdGame.players)

			// Send event to update lobby for other players
			for client := range lobby.clients {
				if client == newGame.Client {
					continue
				}
				client.gameLobbyUpdate <- newGame
			}

		case joinGame := <-lobby.joinGame:
			// Select the game with the given id then add the player to it
			joinedGame, ok := lobby.currentGames[joinGame.GameId]
			if !ok {
				// The game does not exist
				close(joinGame.Client.gameLobbyUpdate)
			}

			joinedGame.joinGame <- joinGame.Client
			joinGame.PlayerCount = len(joinedGame.players)

			// Send event to other client to remove the game from the list
			for client := range lobby.clients {
				client.gameLobbyUpdate <- joinGame
			}

		case game := <-lobby.deleteGame:
			delete(lobby.currentGames, game.gameId)
		}
	}
}
