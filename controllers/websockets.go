package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

const (
	websocketMessageRegisterPlayer = "RegisterPlayer"
	websocketMessagePlayerRemoved  = "PlayerRemoved"
	websocketMessageEnqueue        = "Enqueue"
	websocketMessageRegister       = "Register"
	websocketMessageNoPlayer       = "NoPlayer"
	websocketMessagePlayerExists   = "PlayerExists"
	websocketMessageError          = "Error"
)

type EnqueuePayload struct {
	ItemIds   []string `json:"itemIds"`
	PodcastId string   `json:"podcastId"`
	TagIds    []string `json:"tagIds"`
}

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var activePlayers = make(map[*websocket.Conn]string)
var allConnections = make(map[*websocket.Conn]string)

var broadcast = make(chan Message) // broadcast channel

type Message struct {
	Identifier  string          `json:"identifier"`
	MessageType string          `json:"messageType"`
	Payload     string          `json:"payload"`
	Connection  *websocket.Conn `json:"-"`
}

func WSHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsupgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Failed to set websocket upgrade: %+v", err)
		return
	}
	defer conn.Close()
	for {
		var mess Message
		err := conn.ReadJSON(&mess)
		if err != nil {
			//	fmt.Println("Socket Error")
			// fmt.Println(err.Error())
			isPlayer := activePlayers[conn] != ""
			if isPlayer {
				delete(activePlayers, conn)
				broadcast <- Message{
					MessageType: "PlayerRemoved",
					Identifier:  mess.Identifier,
				}
			}
			delete(allConnections, conn)
			break
		}
		mess.Connection = conn
		allConnections[conn] = mess.Identifier
		broadcast <- mess
		//	conn.WriteJSON(mess)
	}
}

func HandleWebsocketMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast

		switch msg.MessageType {
		case websocketMessageRegisterPlayer:
			activePlayers[msg.Connection] = msg.Identifier
			for connection := range allConnections {
				connection.WriteJSON(Message{
					Identifier:  msg.Identifier,
					MessageType: websocketMessagePlayerExists,
				})
			}
			fmt.Println("Player Registered")
		case websocketMessagePlayerRemoved:
			for connection := range allConnections {
				connection.WriteJSON(Message{
					Identifier:  msg.Identifier,
					MessageType: websocketMessageNoPlayer,
				})
			}
			fmt.Println("Player Registered")
		case websocketMessageEnqueue:
			var payload EnqueuePayload
			fmt.Println(msg.Payload)
			err := json.Unmarshal([]byte(msg.Payload), &payload)
			if err == nil {

				items, err := getItemsToPlay(payload.ItemIds, payload.PodcastId, payload.TagIds)
				if err != nil {
					for connection := range allConnections {
						connection.WriteJSON(Message{
							Identifier:  msg.Identifier,
							MessageType: websocketMessageError,
						})
					}
					break
				}

				var player *websocket.Conn
				for connection, id := range activePlayers {

					if msg.Identifier == id {
						player = connection
						break
					}
				}
				if player != nil {
					payloadStr, err := json.Marshal(items)
					if err == nil {
						player.WriteJSON(Message{
							Identifier:  msg.Identifier,
							MessageType: websocketMessageEnqueue,
							Payload:     string(payloadStr),
						})
					}
				}
			} else {
				fmt.Println(err.Error())
			}
		case websocketMessageRegister:
			var player *websocket.Conn
			for connection, id := range activePlayers {

				if msg.Identifier == id {
					player = connection
					break
				}
			}

			if player == nil {
				msg.Connection.WriteJSON(Message{
					Identifier:  msg.Identifier,
					MessageType: websocketMessageNoPlayer,
				})
			} else {
				msg.Connection.WriteJSON(Message{
					Identifier:  msg.Identifier,
					MessageType: websocketMessagePlayerExists,
				})
			}
		}
	}
}
