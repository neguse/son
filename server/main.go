package main

import (
	"encoding/json"
	"flag"
	"fmt"
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

	R  = 15.0
	BR = 5.0

	ROT   = 1.4
	ACC   = 80.0
	FRIC  = -0.2
	BVELO = 150.0
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
	Id int64   `json:"id"`
}

func (p *Player) Update(dt float64) {
	p.VX += p.VX * FRIC * dt
	p.VY += p.VY * FRIC * dt
	p.X += p.VX * dt
	p.Y += p.VY * dt
	left := p.R
	right := W - p.R
	top := p.R
	bottom := H - p.R

	if p.Id >= 0 {
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
	}
}

func (p *Player) OutOfField() bool {
	left := -p.R
	right := W + p.R
	top := -p.R
	bottom := H + p.R
	return p.X < left || right < p.X || p.Y < top || bottom < p.Y
}

func collision(p1 *Player, p2 *Player) {
	if p1.Id == p2.Id || p1.Id == -p2.Id {
		return
	}
	dx := p2.X - p1.X
	dy := p2.Y - p1.Y
	d := math.Sqrt(dx*dx + dy*dy)
	dix := dx / d
	diy := dy / d
	l := p1.R + p2.R
	if d < l {
		r1 := (l - d) * p1.R / l
		r2 := (l - d) * p2.R / l
		p1.X -= r1 * dix
		p1.Y -= r1 * diy
		p2.X += r2 * dix
		p2.Y += r2 * diy

		dvx := p2.VX - p1.VX
		dvy := p2.VY - p1.VY
		E := 0.9
		dr := (dvx*dix + dvy*diy) * (1 + E) / l
		p1.VX += dix * dr * p2.R
		p1.VY += diy * dr * p2.R
		p2.VX -= dix * dr * p1.R
		p2.VY -= diy * dr * p1.R
	}
}

type S2CMessage struct {
	Players []Player `json:"players"`
	YourId  int64    `json:"yourid"`
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
	bullets []Player
	nextId  int64
}

func NewServer() *Server {
	s := &Server{
		clients: make(map[*Client]*PlayerState),
		bullets: nil,
		nextId:  1,
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
			var players []Player
			clients := make([]*Client, 0, len(s.clients))
			for c := range s.clients {
				clients = append(clients, c)
			}
			{
				bullets := s.bullets[:0]
				for i := range s.bullets {
					b := &(s.bullets[i])
					b.Update(dt)
					if !b.OutOfField() {
						bullets = append(bullets, *b)
					}
				}
				s.bullets = bullets
			}
			for _, c := range clients {
				ps := s.clients[c]
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
				if ks.D {
					b := *p
					b.Id = -p.Id
					b.VA = 0
					b.VX += math.Cos(p.A) * BVELO
					b.VY += math.Sin(p.A) * BVELO
					b.R = BR
					s.bullets = append(s.bullets, b)

					p.VX -= math.Cos(p.A) * BVELO * 0.1
					p.VY -= math.Sin(p.A) * BVELO * 0.1
				}
				p.Update(dt)
			}
			for _, b := range s.bullets {
				players = append(players, b)
			}
			for i, c := range clients {
				ps := s.clients[c]
				p := ps.Player
				for _, c2 := range clients[i+1:] {
					ps2 := s.clients[c2]
					p2 := ps2.Player
					collision(p, p2)
				}
				for i := range s.bullets {
					collision(p, &s.bullets[i])
				}
				players = append(players, *p)
			}
			for _, c := range clients {
				msg := &S2CMessage{Players: players, YourId: s.clients[c].Player.Id}
				c.Send(msg)
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
	id := s.nextId
	s.nextId++
	s.clients[client] = &PlayerState{
		Player: &Player{
			X:  rand.Float64() * W,
			Y:  rand.Float64() * H,
			R:  R,
			A:  0.0,
			VX: 0.0,
			VY: 0.0,
			Id: id,
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
	var port int
	flag.IntVar(&port, "port", 8080, "listen port")
	flag.Parse()

	s := NewServer()
	go s.Main()
	http.Handle("/ws", s)
	fs := assetFS()
	fs.Prefix = "assets"
	http.Handle("/", http.FileServer(fs))
	err := http.ListenAndServe(fmt.Sprint(":", port), nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
