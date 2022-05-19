package main

import (
	"fmt"
	"math/rand"
)

type Game struct {
	gameId      int
	players     []*Client
	gameCreator *Client
	gameName    string
	gameState   [3][3]int
	currentTurn *Client
	gameStarted bool
	gameOver    bool
	winner      *Client
	joinGame    chan *Client
	leaveGame   chan *Client
	startGame   chan bool
	endGame     chan bool
	updateState chan PlayerAction
	moveCount   int
}

// Add game to list if player count == 1, remove if player count == 2
type GameLobbyUpdate struct {
	Action      string  `json:"action"`
	GameId      int     `json:"gameId"`
	GameCreator string  `json:"gameCreator,omitempty"`
	GameName    string  `json:"gameName,omitempty"`
	PlayerCount int     `json:"playerCount"`
	Client      *Client `json:"-"`
}

type ToggleGameTab struct {
	Action        string `json:"action"`
	ToggleGameTab bool   `json:"toggleGameTab"`
	GameName      string `json:"gameName"`
}

type GameSync struct {
	Action      string    `json:"action"`
	GameState   [3][3]int `json:"gameState"`
	CurrentTurn string    `json:"currentTurn,omitempty"`
	Winner      string    `json:"winner,omitempty"`
	Draw        bool      `json:"draw,omitempty"`
}

type GameList struct {
	Action   string         `json:"action"`
	GameList []GameListItem `json:"gameList"`
}

type GameListItem struct {
	Id      int    `json:"gameId"`
	Name    string `json:"gameName"`
	Creator string `json:"gameCreator"`
}

type PlayerAction struct {
	Action string  `json:"action"`
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Client *Client `json:"-"`
}

func createGame(gameCreator *Client, gameName string, gameId int) *Game {
	return &Game{
		players:     []*Client{},
		gameId:      gameId,
		gameCreator: gameCreator,
		gameName:    gameName,
		joinGame:    make(chan *Client),
		leaveGame:   make(chan *Client),
		startGame:   make(chan bool),
		endGame:     make(chan bool),
		updateState: make(chan PlayerAction),
	}
}

func (game *Game) runGame() {
	for {
		select {
		// Join a player
		case player := <-game.joinGame:
			if len(game.players) < 2 {
				game.players = append(game.players, player)
				player.game = game

				player.toggleGameTab <- ToggleGameTab{
					Action:        "toggleGameTab",
					ToggleGameTab: true,
					GameName:      game.gameName,
				}
			}

			if len(game.players) == 2 {
				startGame(game, true)
			}

		// Update the game state for every player in the game
		case playerAction := <-game.updateState:
			if validPlacement(game, playerAction) {
				gameSync := placePiece(game, playerAction)

				for _, player := range game.players {
					player.gameSync <- gameSync
				}

				// Delete games from the existing ones when it's over and stop go routine
				if game.gameOver {
					playerAction.Client.lobby.deleteGame <- game
					return
				}
			}
		case <-game.endGame:
			return
		}
	}
}

func startGame(game *Game, gameStarted bool) {
	game.currentTurn = randomizeStartTurn(game.players)
	game.gameStarted = gameStarted

	for _, player := range game.players {

		// Assign piece to each player
		if player == game.currentTurn {
			player.piece = 1
		} else {
			player.piece = 2
		}

		player.gameSync <- GameSync{
			Action:      "turnStart",
			CurrentTurn: game.currentTurn.user.username,
			GameState:   game.gameState,
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
	return clients[rand.Intn(len(clients)-1)]
}

func validPlacement(game *Game, playerAction PlayerAction) bool {

	// Player not actually part of the game
	if !contains(game.players, playerAction.Client) {
		return false
	}

	// Not the player's turn
	fmt.Println(game.currentTurn.user.username)
	if playerAction.Client != game.currentTurn {
		return false
	}

	x := playerAction.X
	y := playerAction.Y

	// Move outside the board
	if x > len(game.gameState) || y > len(game.gameState) || x < 0 || y < 0 {
		return false
	}

	// Already a piece at given coords
	if game.gameState[y][x] != 0 {
		return false
	}

	return true
}

func placePiece(game *Game, playerAction PlayerAction) GameSync {
	x := playerAction.X
	y := playerAction.Y

	piece := playerAction.Client.piece

	game.gameState[y][x] = piece
	game.moveCount++

	nextPlayer := findNextPlayer(game)

	// Check winning conditions
	len := len(game.gameState)

	for i := 0; i < len; i++ {
		if game.gameState[y][i] != piece {
			break
		}
		if i == len-1 {
			game.winner = playerAction.Client
		}
	}

	for i := 0; i < len; i++ {
		if game.gameState[i][x] != piece {
			break
		}
		if i == len-1 {
			game.winner = playerAction.Client
		}
	}

	if x == y {
		for i := 0; i < len; i++ {
			if game.gameState[i][i] != piece {
				break
			}
			if i == len-1 {
				game.winner = playerAction.Client
			}
		}
	}

	if x+y == len-1 {
		for i := 0; i < len; i++ {
			if game.gameState[i][(len-1)-i] != piece {
				break
			}
			if i == len-1 {
				game.winner = playerAction.Client
			}
		}
	}

	res := GameSync{
		Action:    "turnStart",
		GameState: game.gameState,
	}

	// Draw
	if game.moveCount == 9 {
		game.gameOver = true
		res.Draw = true
		res.Action = "draw"
	}

	// Set winner
	if game.winner != nil {
		game.gameOver = true
		res.Winner = playerAction.Client.user.username
		res.Action = "gameOver"
	}

	// Set the next player's turn
	if !game.gameOver {
		game.currentTurn = nextPlayer
		res.CurrentTurn = game.currentTurn.user.username
	}

	return res
}

func findNextPlayer(game *Game) *Client {
	for _, player := range game.players {
		if player != game.currentTurn {
			return player
		}
	}
	return game.currentTurn
}
