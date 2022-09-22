package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	// "github.com/btwiuse/h3/utils"
	"github.com/ebi-yade/altsvc-go"
	"github.com/marten-seemann/webtransport-go"
	"k0s.io/pkg/dial"
	"nhooyr.io/websocket"
)

func main() {
	u, err := url.Parse("ws://localhost:8080")
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
		ep := endpoints[0]
		log.Println("Found", endpoints)
		log.Println("switch to webtransport", ep, u.Host)
		// use u.Host for now

		ur := fmt.Sprintf("https://%s", u.Host)
		ctx, _ := context.WithTimeout(context.TODO(), time.Second)
		var d webtransport.Dialer
		log.Printf("dialing %s (UDP)", ur)
		resp, conn, err := d.Dial(ctx, ur, nil)
		if err != nil {
			log.Fatalln(err)
		}
		_ = resp
		handleConn(conn)
		return nil, nil
	}
	return websocket.NetConn(context.Background(), wsconn, websocket.MessageBinary), nil
}

type HostPort struct {
	Host string
	Port string
}

func (hp *HostPort) String() string {
	return hp.Host + ":" + hp.Port
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

func handleConn(conn *webtransport.Session) {
	log.Println("new conn", conn.LocalAddr())
	ctx, _ := context.WithTimeout(context.TODO(), time.Second)
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		log.Println("error opening stream:", err)
		return
	}
	go io.Copy(os.Stdout, stream)
	io.Copy(stream, os.Stdin)
}
