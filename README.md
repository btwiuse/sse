# sse + ws + http

```
# sse
curl -H 'Accept: text/event-stream' -N http://127.0.0.1:8080
firefox ws://localhost:8080

# ws
websocat ws://[::1]:8080
websocat ws://localhost:8080

# http
curl http://127.0.0.1:8080/once
```
