package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	writeWait = 10 * time.Second

	pongWait = 60 * time.Second

	pingPeriod = (pongWait * 9) / 10

	maxMessageSize = 512
)

type User struct {
	username string
}

type Client struct {
	user            *User
	lobby           *Lobby
	game            *Game
	conn            *websocket.Conn
	piece           int
	chatMessage     chan ChatMessage
	gameLobbyUpdate chan GameLobbyUpdate
	toggleGameTab   chan ToggleGameTab
	gameSync        chan GameSync
	syncGameList    chan GameList
}

type Action struct {
	Client       *Client
	Username     string         `json:"setUsername"`
	Action       string         `json:"action"`
	ChatMessage  string         `json:"chatMessage"`
	StartMessage string         `json:"startMessage"`
	StartTurn    *Client        `json:"startTurn"`
	PlayerAction map[string]int `json:"playerAction"`
	CreateGame   bool           `json:"createGame"`
	GameName     string         `json:"gameName"`
	GameCreated  *Game          `json:"gameCreated"`
	JoinGameId   int            `json:"gameId"`
}

func (c *Client) readPump() {
	defer func() {
		c.lobby.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		var action Action
		err := c.conn.ReadJSON(&action)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		fmt.Println(action)

		determineAction(action, c)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case chatMessage, ok := <-c.chatMessage:
			writeMessage(c, chatMessage, ok)

		case gameLobbyUpdate, ok := <-c.gameLobbyUpdate:
			writeMessage(c, gameLobbyUpdate, ok)

		case toggleGameTab, ok := <-c.toggleGameTab:
			writeMessage(c, toggleGameTab, ok)

		case gameSync, ok := <-c.gameSync:
			writeMessage(c, gameSync, ok)

		case gameList, ok := <-c.syncGameList:
			writeMessage(c, gameList, ok)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func serveWs(lobby *Lobby, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		lobby:           lobby,
		conn:            conn,
		chatMessage:     make(chan ChatMessage, 256),
		gameLobbyUpdate: make(chan GameLobbyUpdate, 256),
		toggleGameTab:   make(chan ToggleGameTab, 256),
		gameSync:        make(chan GameSync, 256),
		syncGameList:    make(chan GameList),
	}

	client.lobby.register <- client

	go client.writePump()
	go client.readPump()
}

func determineAction(action Action, c *Client) {
	if action.PlayerAction != nil {
		fmt.Println(c)
		c.game.updateState <- PlayerAction{
			Action: "placePiece",
			X:      action.PlayerAction["x"],
			Y:      action.PlayerAction["y"],
			Client: c,
		}
	} else if action.ChatMessage != "" {
		c.lobby.broadcast <- ChatMessage{
			Action:   "chatMessage",
			Message:  action.ChatMessage,
			Username: c.user.username,
		}
	} else if action.CreateGame {
		c.lobby.createGame <- GameLobbyUpdate{
			Action:      "gameLobbyUpdate",
			GameId:      len(c.lobby.currentGames) + 1,
			GameCreator: c.user.username,
			GameName:    action.GameName,
			Client:      c,
		}
	} else if action.JoinGameId != 0 {
		c.lobby.joinGame <- GameLobbyUpdate{
			Action: "gameLobbyUpdate",
			GameId: action.JoinGameId,
			Client: c,
		}
	} else if action.Username != "" { // Set username & return list of available games for client
		c.user = &User{
			username: action.Username,
		}
		gameList := []GameListItem{}

		for _, game := range c.lobby.currentGames {
			gameList = append(gameList, GameListItem{
				Id:      game.gameId,
				Name:    game.gameName,
				Creator: game.gameCreator.user.username,
			})
		}
		c.syncGameList <- GameList{
			Action:   "displayGameList",
			GameList: gameList,
		}
	}
}

func writeMessage(c *Client, data interface{}, ok bool) {

	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if !ok {
		// The hub closed the channel.
		c.conn.WriteMessage(websocket.CloseMessage, []byte{})
		return
	}

	jsonData, err := json.Marshal(data)

	if err != nil {
		c.conn.WriteMessage(websocket.CloseMessage, []byte{})
	}

	c.conn.WriteMessage(websocket.TextMessage, []byte(jsonData))
}
