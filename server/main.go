package main

import (
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"net/http"

	"time"

	"github.com/gorilla/websocket"
)

const (
	W = 320.0
	H = 320.0
	R = 10.0
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Player struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	R float64 `json:"r"`
	A float64 `json:"a"`
}

type S2CMessage struct {
	Players []Player `json:"players"`
}
type C2SMessage struct {
}

type Client struct {
	conn *websocket.Conn

	sendingCh chan *S2CMessage
	recvingCh chan *C2SMessage
	errCh     chan error
}

const (
	RecvBuf = 32
	SendBuf = 32
	ErrBuf  = 32
)

func NewClient(conn *websocket.Conn) *Client {
	c := &Client{
		conn:      conn,
		sendingCh: make(chan *S2CMessage, SendBuf),
		recvingCh: make(chan *C2SMessage, RecvBuf),
		errCh:     make(chan error, ErrBuf),
	}
	return c
}

func (c *Client) Send(msg *S2CMessage) {
	c.sendingCh <- msg
}

func (c *Client) Recv() <-chan *C2SMessage {
	return c.recvingCh
}

func (c *Client) Err() <-chan error {
	return c.errCh
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
				c.errCh <- err
				continue
			}
			err = c.conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				c.errCh <- err
				if _, ok := err.(*websocket.CloseError); ok {
					c.Close()
					break
				}
				continue
			}
		}
	}()

	// recving goroutine
	go func() {
		for {
			mt, message, err := c.conn.ReadMessage()
			if err != nil {
				c.errCh <- err
				if _, ok := err.(*websocket.CloseError); ok {
					c.Close()
					break
				}
				continue
			}

			if mt == websocket.TextMessage {
				var recvMsg C2SMessage
				err = json.Unmarshal(message, &recvMsg)
				if err != nil {
					c.errCh <- err
					continue
				}
				c.recvingCh <- &recvMsg
			}
		}
	}()
}

const (
	Limit = 10
)

type Server struct {
	clients map[*Client]*Player
}

func NewServer() *Server {
	s := &Server{
		clients: make(map[*Client]*Player),
	}
	return s
}

func (s *Server) Broadcast(msg *S2CMessage) {
	for c, _ := range s.clients {
		c.Send(msg)
	}
}

func (s *Server) Main() {
	for {
		updateTick := time.Tick(100 * time.Millisecond)
		for {
			select {
			case <-updateTick:
				var msg S2CMessage
				for _, p := range s.clients {
					p.A += 0.1
					p.X += math.Cos(p.A) * 1.0
					p.Y += math.Sin(p.A) * 1.0
					msg.Players = append(msg.Players, *p)
				}
				s.Broadcast(&msg)
			}
		}
		log.Println("clients:", len(s.clients))
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
	s.clients[client] = &Player{X: rand.Float64() * W, Y: rand.Float64() * H, R: R}
	go client.Main()
	go func() {
		for {
			select {
			case <-client.Recv():

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
	http.Handle("/ws", s)
	fs := assetFS()
	fs.Prefix = "assets"
	http.Handle("/", http.FileServer(fs))
	err := http.ListenAndServe(":12345", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
