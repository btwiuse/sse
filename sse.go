package sse

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/btwiuse/gmx"
	"nhooyr.io/websocket"
)

func (s *SSE) broadcast() {
	clients := s.Clients()
	for cid, _ := range clients {
		if len(clients[cid]) > 3 {
			log.Println("[SSE::WARN] client has too many unconsumed messages, which shouldn't happen. Skip notification.", cid)
			continue
		}
		clients[cid] <- struct{}{}
	}
}

func (s *SSE) SetData(data string) {
	s.Data = data
	s.broadcast()
}

func (s *SSE) handlePoll(w http.ResponseWriter, r *http.Request) error {
	_, err := fmt.Fprintf(w, "%s", s.Data)
	if err != nil {
		return err
	}
	w.(http.Flusher).Flush()
	// log.Println(r.RemoteAddr, data)
	return nil
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
	s.StateMutex.Do(InitCid(cid))
}

func (s *SSE) removeClient(cid int) {
	s.StateMutex.Do(RemoveCid(cid))
}

func (s *SSE) Clients() Clients {
	return s.StateMutex.Unwrap().Clients
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
			break
		case <-s.Clients()[cid]:
			break
		}
		if err := s.handleSSEOnce(w, r); err != nil {
			// log.Println(err)
			return
		}
	}
}

func (s *SSE) handleWS(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		OriginPatterns:     []string{"*"},
	})
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
		// read indefinitely to detect client close
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
			break
		case <-s.Clients()[cid]:
			break
		case <-quit:
			return
		}
		if err := c.Write(context.TODO(), websocket.MessageBinary, []byte(s.Data)); err != nil {
			// log.Println(err)
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
		s.handlePoll(w, r)
	}
}

type SSE struct {
	nid                int
	Data               string
	StateMutex         *gmx.RwMx[State]
	MaxMessageInterval time.Duration
}

func (s *SSE) nextID() int {
	s.nid += 1
	return s.nid
}

func NewSSE() *SSE {
	sse := &SSE{
		MaxMessageInterval: 3 * time.Second,
		StateMutex: gmx.RwWrap(
			&State{
				Clients: Clients{},
			},
		),
		nid: 0,
	}
	return sse
}

type Clients = map[int]chan struct{}

type State struct {
	Clients map[int]chan struct{}
}

func InitCid(cid int) gmx.Mutation[State] {
	return func(s *State) {
		s.Clients[cid] = make(chan struct{}, 4)
	}
}

func RemoveCid(cid int) gmx.Mutation[State] {
	return func(s *State) {
		delete(s.Clients, cid)
	}
}
