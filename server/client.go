package main

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

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
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					log.Println("unexpected close error : ", err)
				}
				break
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
