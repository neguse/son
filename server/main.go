package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
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
	log.Println("Listen:", port)
	err := http.ListenAndServe(fmt.Sprint(":", port), nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
