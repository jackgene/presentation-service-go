# Presentation Service in Go

Build and run:
```shell
go run cmd/server/server.go --port 8973 --html-path (path to deck.html)
```

Build then run:
```shell
go build -o dist/presentation-service cmd/server/server.go
dist/presentation-service --port 8973 --html-path (path to deck.html)
```

### Background
This is built using Gin and Gorilla (for WebSockets).

TODO: attempt Gorilla/Mux implementation.
