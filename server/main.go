package main

import (
	"encoding/json"
	"log"
	"net/http"

	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Message struct {
	User string `json:"user"`
	Body string `json:"body"`
}

// Server is http.Handler
type Server struct {
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	go func() {
		tick := time.Tick(5 * time.Second)
		for {
			<-tick
			pong := Message{User: "server", Body: "pong"}
			data, err := json.Marshal(&pong)
			if err != nil {
				log.Print("failed to marshal:", err)
				break
			}
			err = c.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				log.Print("failed to write:", err)
				break
			}
		}
	}()

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)

		var recvMsg Message
		err = json.Unmarshal(message, &recvMsg)
		if err != nil {
			log.Println("failed to parse:", err)
			break
		}

		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}

}

func main() {
	g := &Server{}
	http.Handle("/echo", g)
	fs := assetFS()
	fs.Prefix = "assets"
	http.Handle("/", http.FileServer(fs))
	err := http.ListenAndServe(":12345", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
