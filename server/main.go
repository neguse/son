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

type Client struct {
	conn *websocket.Conn

	sendingCh chan *Message
	recvingCh chan *Message
	err       chan error
}

const (
	RecvBuf = 32
	SendBuf = 32
)

func NewClient(conn *websocket.Conn) *Client {
	c := &Client{
		conn:      conn,
		sendingCh: make(chan *Message, SendBuf),
		recvingCh: make(chan *Message, RecvBuf),
	}
	return c
}

func (c *Client) Send(msg *Message) {
	c.sendingCh <- msg
}

func (c *Client) Recv() <-chan *Message {
	return c.recvingCh
}

func (c *Client) Err() <-chan error {
	return c.err
}

func (c *Client) Close() {
	c.conn.Close()
}

func (c *Client) Main() {
	// sending goroutine
	go func() {
		for {
			msg := <-c.sendingCh
			data, err := json.Marshal(msg)
			if err != nil {
				c.err <- err
				continue
			}
			err = c.conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				c.err <- err
				continue
			}
		}
	}()

	// recving goroutine
	go func() {
		for {
			mt, message, err := c.conn.ReadMessage()
			if err != nil {
				c.err <- err
				continue
			}

			if mt == websocket.TextMessage {
				var recvMsg Message
				err = json.Unmarshal(message, &recvMsg)
				if err != nil {
					c.err <- err
					continue
				}
				c.recvingCh <- &recvMsg
			}

		}
	}()
}

type Server struct {
	messages []Message
	clients  map[*Client]struct{}
}

func NewServer() *Server {
	s := &Server{
		messages: nil,
		clients:  make(map[*Client]struct{}),
	}
	return s
}

func (s *Server) Broadcast(msg *Message) {
	for c, _ := range s.clients {
		c.Send(msg)
	}
}

func (s *Server) Main() {
	for {
		pongTick := time.Tick(1 * time.Second)
		for {
			select {
			case <-pongTick:
				pong := Message{User: "server", Body: "pong"}
				s.Broadcast(&pong)
			}
		}
	}
}

func (s *Server) Close(c *Client) {
	c.Close()
	delete(s.clients, c)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	client := NewClient(c)
	s.clients[client] = struct{}{}
	go client.Main()
	go func() {
		for {
			select {
			case recv := <-client.Recv():
				s.Broadcast(recv)
			case err := <-client.Err():
				log.Println("client error:", client, err)
				s.Close(client)
			}
		}
	}()
}

func main() {
	s := NewServer()
	go s.Main()
	http.Handle("/echo", s)
	fs := assetFS()
	fs.Prefix = "assets"
	http.Handle("/", http.FileServer(fs))
	err := http.ListenAndServe(":12345", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
