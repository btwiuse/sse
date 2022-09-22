package main

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/ebi-yade/altsvc-go"
	"k0s.io/pkg/dial"
	"nhooyr.io/websocket"
)

func main() {
	u, err := url.Parse("ws://127.0.0.1:8080")
	if err != nil {
		log.Fatalln(err)
	}

	conn, err := dial3(u)
	if err != nil {
		log.Fatalln(err)
	}

	io.Copy(os.Stdout, conn)
}

func dial1(u *url.URL) (net.Conn, error) {
	return dial.Dial(u)
}

func dial2(u *url.URL) (net.Conn, error) {
	wsconn, _, err := websocket.Dial(
		context.Background(),
		u.String(),
		nil,
	)
	if err != nil {
		return nil, err
	}
	return websocket.NetConn(context.Background(), wsconn, websocket.MessageBinary), nil
}

func dial3(u *url.URL) (net.Conn, error) {
	wsconn, resp, err := websocket.Dial(
		context.Background(),
		u.String(),
		nil,
	)
	if err != nil {
		return nil, err
	}
	endpoints, ok := extractH3(resp.Header)
	if ok {
		log.Println("Found", endpoints)
		log.Println("TODO: switch to webtransport")
	}
	return websocket.NetConn(context.Background(), wsconn, websocket.MessageBinary), nil
}

func extractH3(h http.Header) ([]string, bool) {
	line := h.Get("Alt-Svc")
	if line == "" {
		return []string{}, false
	}
	svcs, err := altsvc.Parse(line)
	if err != nil {
		log.Println(err)
		return []string{}, false
	}
	results := []string{}
	log.Println(svcs)
	for _, svc := range svcs {
		if svc.ProtocolID == "h3" {
			results = append(results, svc.AltAuthority.Host+":"+svc.AltAuthority.Port)
		}
	}
	return results, len(results) > 0
}
