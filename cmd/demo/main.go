package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/btwiuse/sse"
)

func indexWith(s *sse.SSE) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		isIndex := r.URL.Path == "/"
		isFavicon := r.URL.Path == "/favicon.ico"
		isSSE := strings.Split(r.Header.Get("Accept"), ",")[0] == "text/event-stream"
		isWS := strings.Split(r.Header.Get("Upgrade"), ",")[0] == "websocket"

		if (isWS || isSSE) || !(isIndex || isFavicon) {
			s.ServeHTTP(w, r)
			return
		}

		switch {
		case isIndex:
			handleIndex(w, r)
		case isFavicon:
			http.Error(w, "", http.StatusOK)
		}
	})
}

func DateSSE() *sse.SSE {
	sse := sse.NewSSE()
	go (func() {
		for {
			now := time.Now().Format("2006-01-02 15:04:05")
			sse.SetData(now)
			time.Sleep(time.Second)
		}
	})()
	return sse
}

//go:embed index.html
var IndexHtml string

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, IndexHtml)
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("listening on http://127.0.0.1:8080")
	log.Fatalln(http.ListenAndServe(":8080", indexWith(DateSSE())))
}
