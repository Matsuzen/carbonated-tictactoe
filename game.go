package main

import (
	"fmt"
	"math/rand"
)

type Game struct {
	players     []*Client
	boardState  [3][3]int
	currentTurn *Client
	gameStarted bool
	gameOver    bool
	winner      *Client
	joinGame    chan *Client
	leaveGame   chan *Client
	startGame   chan bool
	updateState chan Message
}

func newGame() *Game {
	return &Game{
		players:   make([]*Client, 2),
		joinGame:  make(chan *Client),
		leaveGame: make(chan *Client),
	}
}

func (game *Game) runGame() {
	for {
		select {
		case player := <-game.joinGame:
			if len(game.players) < 2 {
				game.players = append(game.players, player)
				return
			} else {

			}

			// Start the game if it has 2 players
			game.startGame <- true
		case gameStarted := <-game.startGame:
			for _, player := range game.players {
				startTurn := randomizeStartTurn(game.players)

				game.gameStarted = gameStarted

				player.send <- Message{
					StartTurn:    startTurn,
					StartMessage: fmt.Sprintf("The game has started! It is %s's turn", startTurn.user.username),
				}
			}
		}
	}
}

func contains(clients []*Client, client *Client) bool {
	for _, c := range clients {
		if c == client {
			return true
		}
	}

	return false
}

func randomizeStartTurn(clients []*Client) *Client {
	return clients[rand.Intn(len(clients))]
}
