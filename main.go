package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"nhooyr.io/websocket"
)

//go:embed index.html
var IndexHtml string

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, IndexHtml)
}

func date() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func handleSSEOnce(w http.ResponseWriter, r *http.Request) {
	data := fmt.Sprintf("data: %s", date())
	_, err := fmt.Fprintf(w, "%s\n\n", data)
	if err != nil {
		log.Println(err)
		return
	}
	w.(http.Flusher).Flush()
	log.Println(r.RemoteAddr, data)
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	for {
		handleSSEOnce(w, r)
		time.Sleep(1 * time.Second)
	}
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer c.Close(websocket.StatusInternalError, "")

	for {
		log.Println(r.RemoteAddr, date())
		err := c.Write(context.TODO(), websocket.MessageText, []byte(date()))
		if err != nil {
			log.Println(err)
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func handleAll(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	isFavicon := r.URL.Path == "/favicon.ico"
	isOnce := r.URL.Path == "/once"
	isSSE := strings.Split(r.Header.Get("Accept"), ",")[0] == "text/event-stream"
	isWS := strings.Split(r.Header.Get("Upgrade"), ",")[0] == "websocket"

	switch {
	case isFavicon:
		http.Error(w, "", http.StatusOK)
	case isOnce:
		handleSSEOnce(w, r)
	case isSSE:
		handleSSE(w, r)
	case isWS:
		handleWS(w, r)
	default:
		handleIndex(w, r)
	}
}

func createHandler() http.HandlerFunc {
	return handleAll
}

func createServer() http.Handler {
	handler := createHandler()
	return http.HandlerFunc(handler)
}

func main() {
	server := createServer()
	log.Fatalln(http.ListenAndServe(":8080", server))
}
