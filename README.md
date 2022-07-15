# Presentation Service in Go

Build and run:
```shell
go run cmd/server/server.go --port=8973 --html-path=(path to deck.html)
```

Build then run:
```shell
go build -o dist/server cmd/server/server.go
dist/server --port=8973 --html-path=(path to deck.html)
```
