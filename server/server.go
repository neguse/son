package main

import (
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Server struct {
	clients    map[*Client]*PlayerState
	bullets    []Player
	nextId     int64
	arrivingCh chan *Client
	closingCh  chan *Client
}

func NewServer() *Server {
	s := &Server{
		clients:    make(map[*Client]*PlayerState),
		bullets:    nil,
		nextId:     1,
		arrivingCh: make(chan *Client, 32),
		closingCh:  make(chan *Client, 32),
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
	logTick := time.Tick(1 * time.Minute)
	lastTick := time.Now()
	for {
		select {
		case c := <-s.closingCh:
			log.Print("client closing:", s.clients[c].Player.Id)
			delete(s.clients, c)
		case c := <-s.arrivingCh:
			id := s.nextId
			s.nextId++
			s.clients[c] = &PlayerState{
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
			log.Print("client arrived:", id)
			go c.Main()
			go func() {
				for {
					select {
					case recvMsg := <-c.Recv():
						ks := s.clients[c].KeyState
						ks.L = recvMsg.L
						ks.R = recvMsg.R
						ks.U = recvMsg.U
						ks.D = recvMsg.D

					case err := <-c.Err():
						log.Println("client error:", c, err)
						s.Close(c)
					}
				}
			}()
		case <-logTick:
			log.Println("clients:", len(s.clients))
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
	}
}

func (s *Server) Close(c *Client) {
	c.Close()
	s.closingCh <- c
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	client := NewClient(c)
	s.arrivingCh <- client
}
