# sse + ws + http

```
# SSE
curl -H 'Accept: text/event-stream' -N http://127.0.0.1:8080
firefox ws://localhost:8080

# WebSocket
websocat ws://[::1]:8080
websocat ws://localhost:8080

# HTTP
curl http://127.0.0.1:8080/once

# SSE over HTTP/3.
# Needs ufo, see https://github.com/btwiuse/ufo/blob/main/sse/sse.go
curl3 -H "Accept: text/event-stream" -N https://4.ufo.k0s.io:302 --http3 -H "Host: 4.ufo.k0s.io"
```
