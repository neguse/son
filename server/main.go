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

	ROT  = 1.4
	ACC  = 80.0
	FRIC = -0.01
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type PlayerState struct {
	Player   *Player
	KeyState *KeyState
}
type Player struct {
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	A  float64 `json:"a"`
	VX float64 `json:"vx"`
	VY float64 `json:"vy"`
	VA float64 `json:"va"`
	R  float64 `json:"r"`
}

type S2CMessage struct {
	Players []Player `json:"players"`
}

type KeyState struct {
	L bool `json:"l"`
	R bool `json:"r"`
	U bool `json:"u"`
	D bool `json:"d"`
}

type C2SMessage KeyState

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

type Server struct {
	clients map[*Client]*PlayerState
}

func NewServer() *Server {
	s := &Server{
		clients: make(map[*Client]*PlayerState),
	}
	return s
}

func (s *Server) Broadcast(msg *S2CMessage) {
	for c := range s.clients {
		c.Send(msg)
	}
}

func (s *Server) Main() {
	updateTick := time.Tick(100 * time.Millisecond)
	lastTick := time.Now()
	for {
		select {
		case now := <-updateTick:
			dt := time.Since(lastTick).Seconds()
			lastTick = now
			var msg S2CMessage
			for _, ps := range s.clients {
				p, ks := ps.Player, ps.KeyState
				p.VA = 0
				if ks.L && ks.R {
				} else if ks.L {
					p.VA = -ROT
				} else if ks.R {
					p.VA = ROT
				}
				p.A += p.VA * dt

				if ks.U {
					p.VX += math.Cos(p.A) * ACC * dt
					p.VY += math.Sin(p.A) * ACC * dt
				}
				p.VX += p.VX * FRIC * dt
				p.VY += p.VY * FRIC * dt
				p.X += p.VX * dt
				p.Y += p.VY * dt
				left := p.R
				right := W - p.R
				top := p.R
				bottom := H - p.R
				if p.X < left {
					p.X = left
					p.VX = -p.VX
				}
				if p.Y < top {
					p.Y = top
					p.VY = -p.VY
				}
				if right < p.X {
					p.X = right
					p.VX = -p.VX
				}
				if bottom < p.Y {
					p.Y = bottom
					p.VY = -p.VY
				}
				msg.Players = append(msg.Players, *p)
			}
			s.Broadcast(&msg)
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
	s.clients[client] = &PlayerState{
		Player: &Player{
			X:  rand.Float64() * W,
			Y:  rand.Float64() * H,
			R:  R,
			A:  0.0,
			VX: 0.0,
			VY: 0.0,
		},
		KeyState: &KeyState{},
	}
	go client.Main()
	go func() {
		for {
			select {
			case recvMsg := <-client.Recv():
				ks := s.clients[client].KeyState
				ks.L = recvMsg.L
				ks.R = recvMsg.R
				ks.U = recvMsg.U
				ks.D = recvMsg.D

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
