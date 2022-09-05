package sse

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

func (s *SSE) broadcast() {
	log.Println(s.Clients)
	s.clock.RLock()
	defer s.clock.RUnlock()
	for cid, _ := range s.Clients {
		if len(s.Clients[cid]) > 3 {
			log.Println("[SSE::WARN] client has too many unconsumed messages, which shouldn't happen. Skip notification.", cid)
			continue
		}
		s.Clients[cid] <- struct{}{}
	}
}

func (s *SSE) SetData(data string) {
	s.Data = data
	s.broadcast()
}

func (s *SSE) handleSSEOnce(w http.ResponseWriter, r *http.Request) error {
	data := fmt.Sprintf("data: %s", s.Data)
	_, err := fmt.Fprintf(w, "%s\n\n", data)
	if err != nil {
		return err
	}
	w.(http.Flusher).Flush()
	// log.Println(r.RemoteAddr, data)
	return nil
}

func (s *SSE) addClient(cid int) {
	s.clock.Lock()
	s.Clients[cid] = make(chan struct{}, 4)
	s.clock.Unlock()
}

func (s *SSE) removeClient(cid int) {
	s.clock.Lock()
	delete(s.Clients, cid)
	s.clock.Unlock()
}

func (s *SSE) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	cid := s.nextID()
	s.addClient(cid)
	defer func() {
		// log.Println("delete sse", cid)
		s.removeClient(cid)
	}()

	for {
		select {
		case <-time.After(s.MaxMessageInterval):
			if err := s.handleSSEOnce(w, r); err != nil {
				// log.Println(err)
				return
			}
		case <-s.Clients[cid]:
			if err := s.handleSSEOnce(w, r); err != nil {
				// log.Println(err)
				return
			}
		}
	}
}

func (s *SSE) handleWS(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer c.Close(websocket.StatusInternalError, "")

	cid := s.nextID()
	s.addClient(cid)
	defer func() {
		// log.Println("delete ws", cid)
		s.removeClient(cid)
	}()

	quit := make(chan struct{})

	go func() {
		for {
			_, _, err := c.Read(context.TODO())
			if err != nil {
				// log.Println(err)
				close(quit)
				return
			}
		}
	}()

	for {
		// log.Println(r.RemoteAddr, s.Data)
		select {
		case <-time.After(s.MaxMessageInterval):
			if err := c.Write(context.TODO(), websocket.MessageBinary, []byte(s.Data)); err != nil {
				// log.Println(err)
				return
			}
		case <-s.Clients[cid]:
			if err := c.Write(context.TODO(), websocket.MessageBinary, []byte(s.Data)); err != nil {
				// log.Println(err)
				return
			}
		case <-quit:
			return
		}
	}
}

func (s *SSE) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	isSSE := strings.Split(r.Header.Get("Accept"), ",")[0] == "text/event-stream"
	isWS := strings.Split(r.Header.Get("Upgrade"), ",")[0] == "websocket"

	switch {
	case isWS:
		s.handleWS(w, r)
	case isSSE:
		s.handleSSE(w, r)
	default:
		s.handleSSEOnce(w, r)
	}
}

type SSE struct {
	nid                int
	Data               string
	clock              *sync.RWMutex
	Clients            map[int]chan struct{}
	MaxMessageInterval time.Duration
}

func (s *SSE) nextID() int {
	s.nid += 1
	return s.nid
}

func NewSSE() *SSE {
	sse := &SSE{
		MaxMessageInterval: 3 * time.Second,
		Clients:            map[int]chan struct{}{},
		clock:              &sync.RWMutex{},
		nid:                0,
	}
	return sse
}
