package ws

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	pingMessage = "ping"
	pongMessage = "pong"
)

type WS interface {
	GetConn() *websocket.Conn
	ReadMessage()
	WriteTextMessage(msg []byte)
	WriteBinaryMessage(msg []byte)
}

type ws struct {
	binaryChan chan []byte
	textChan   chan []byte

	conn *websocket.Conn
	ctx  context.Context

	forwardChan chan []byte
}

func New(c *gin.Context, forwardChan chan []byte) (WS, error) {
	w := &ws{
		textChan:    make(chan []byte),
		binaryChan:  make(chan []byte),
		ctx:         c.Request.Context(),
		forwardChan: forwardChan,
	}
	if err := w.upgrade(c); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *ws) upgrade(c *gin.Context) error {
	upGrader := websocket.Upgrader{
		CheckOrigin:     func(*http.Request) bool { return true },
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	conn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return err
	}
	w.conn = conn
	go w.writeMessage()
	return nil
}

func (w *ws) writeMessage() {
	for {
		select {
		case <-w.ctx.Done():
			return
		case cl := <-w.textChan:
			_ = w.conn.WriteMessage(websocket.TextMessage, cl)
		case cl := <-w.binaryChan:
			_ = w.conn.WriteMessage(websocket.BinaryMessage, cl)
		}
	}
}

func (w *ws) GetConn() *websocket.Conn {
	return w.conn
}

func (w *ws) ReadMessage() {
	for {
		_, message, err := w.conn.ReadMessage()
		if err != nil {
			close(w.forwardChan)
			break
		}
		if string(message) == pingMessage {
			w.textChan <- []byte(pongMessage)
		} else {
			w.forwardChan <- message
		}
	}
	_ = w.conn.Close()
}

func (w *ws) WriteTextMessage(msg []byte) {
	if msg == nil {
		return
	}
	w.textChan <- msg
}

func (w *ws) WriteBinaryMessage(msg []byte) {
	if msg == nil {
		return
	}
	w.binaryChan <- msg
}
