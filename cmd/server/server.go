package main

import (
	"embed"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"presentation-service/internal/chat"
	"presentation-service/internal/chat/counter"
	"presentation-service/internal/chat/moderation"
	"presentation-service/internal/token"
	"presentation-service/internal/transcription"
	"strings"
)

type cliParams struct {
	htmlPath string
	port     uint16
}

func parseFlags() cliParams {
	params := cliParams{}
	var port uint

	flag.StringVar(&params.htmlPath, "html-path", "", "Presentation HTML file path")
	flag.UintVar(&port, "port", 8973, "HTTP server port")
	flag.Parse()

	// Required args
	if port > math.MaxUint16 || params.htmlPath == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	params.port = uint16(port)

	return params
}

const routeSeperator = " to "

var validRecipients = map[string]struct{}{
	"Everyone": {},
	"Me":       {},
}

//go:embed public/html
var fs embed.FS

func clientCloseListener(conn *websocket.Conn) <-chan struct{} {
	closed := make(chan struct{})
	go func() {
		for {
			_, _, readErr := conn.NextReader()
			if readErr != nil {
				if _, ok := readErr.(*websocket.CloseError); ok {
					log.Printf("connection closed by client: %v", readErr)
				} else {
					log.Printf("unexpected websocket error: %v", readErr)
				}
				closed <- struct{}{}
				close(closed)
				break
			}
		}
	}()

	return closed
}

func main() {
	params := parseFlags()

	log.SetPrefix("[service] ")
	wsupgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.SetHTMLTemplate(
		template.Must(template.New("").ParseFS(fs, "public/html/*.html")),
	)

	chatMessageBroadcaster := chat.NewBroadcaster("chat")
	rejectedMessageBroadcaster := chat.NewBroadcaster("rejected")
	languagePollCounter := counter.NewSendersByTokenActor(
		"language-poll",
		token.ExtractLanguages,
		chatMessageBroadcaster, rejectedMessageBroadcaster, 200,
	)
	questionBroadcaster := moderation.NewMessageRouter(
		"question", chatMessageBroadcaster, rejectedMessageBroadcaster, 10,
	)
	transcriptionBroadcaster := transcription.NewBroadcaster()

	// Deck
	r.GET("/", func(c *gin.Context) {
		c.File(params.htmlPath)
	})

	r.GET("/event/language-poll", func(c *gin.Context) {
		conn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("failed to upgrade websocket request %v", err)
			return
		}
		defer func() { _ = conn.Close() }()
		clientClosed := clientCloseListener(conn)

		counts := make(chan counter.Counts)
		languagePollCounter.Subscribe(counts)
		defer languagePollCounter.Unsubscribe(counts)
	poll:
		for {
			select {
			case count := <-counts:
				writeErr := conn.WriteJSON(count)
				if writeErr != nil {
					log.Printf("error sending poll response (%v)", writeErr)
					break poll
				}
			case <-clientClosed:
				break poll
			}
		}
	})

	r.GET("/event/question", func(c *gin.Context) {
		conn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("failed to upgrade websocket request %v", err)
			return
		}
		defer func() { _ = conn.Close() }()
		clientClosed := clientCloseListener(conn)

		msgs := make(chan moderation.Messages)
		questionBroadcaster.Subscribe(msgs)
		defer questionBroadcaster.Unsubscribe(msgs)
	poll:
		for {
			select {
			case msg := <-msgs:
				writeErr := conn.WriteJSON(msg)
				if writeErr != nil {
					log.Printf("error sending questions (%v)", writeErr)
					break poll
				}
			case <-clientClosed:
				break poll
			}
		}
	})

	r.GET("/event/transcription", func(c *gin.Context) {
		conn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("failed to upgrade websocket request %v", err)
			return
		}
		defer func() { _ = conn.Close() }()
		clientClosed := clientCloseListener(conn)

		transcripts := make(chan transcription.Transcript)
		transcriptionBroadcaster.Subscribe(transcripts)
		defer transcriptionBroadcaster.Unsubscribe(transcripts)
	poll:
		for {
			select {
			case msg := <-transcripts:
				writeErr := conn.WriteJSON(msg)
				if writeErr != nil {
					log.Printf("error sending transcription (%v)", writeErr)
					break poll
				}
			case <-clientClosed:
				break poll
			}
		}
	})

	// Moderation
	r.GET("/moderator", func(c *gin.Context) {
		c.HTML(http.StatusOK, "moderator.html", nil)
	})

	r.GET("/moderator/event", func(c *gin.Context) {
		conn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("failed to upgrade websocket request %v", err)
			return
		}
		defer func() { _ = conn.Close() }()
		clientClosed := clientCloseListener(conn)

		msgs := make(chan chat.Message)
		rejectedMessageBroadcaster.Subscribe(msgs)
		defer rejectedMessageBroadcaster.Unsubscribe(msgs)
	poll:
		for {
			select {
			case msg := <-msgs:
				writeErr := conn.WriteJSON(msg)
				if writeErr != nil {
					log.Printf("error sending moderation chats (%v)", writeErr)
					break poll
				}
			case <-clientClosed:
				break poll
			}
		}
	})

	r.POST("/chat", func(c *gin.Context) {
		route := c.Query("route")
		sepIdx := strings.LastIndex(route, routeSeperator)
		if sepIdx == -1 {
			log.Println("malformed chat route")
			c.Status(http.StatusBadRequest)
			return
		}

		recipient := route[sepIdx+len(routeSeperator):]
		if _, ok := validRecipients[recipient]; !ok {
			log.Println("invalid chat recipient")
			c.Status(http.StatusBadRequest)
			return
		}

		sender := route[:sepIdx]

		chatMessageBroadcaster.NewMessage(chat.Message{
			Sender:    sender,
			Recipient: recipient,
			Text:      c.Query("text"),
		})
		c.Status(http.StatusNoContent)
	})

	r.GET("/reset", func(c *gin.Context) {
		languagePollCounter.Reset()
		questionBroadcaster.Reset()
		c.Status(http.StatusNoContent)
	})

	// Transcription
	r.GET("/transcriber", func(c *gin.Context) {
		c.HTML(http.StatusOK, "transcriber.html", nil)
	})

	r.POST("/transcription", func(c *gin.Context) {
		transcriptionBroadcaster.NewTranscriptionText(c.Query("text"))
		c.Status(http.StatusNoContent)
	})

	_ = r.SetTrustedProxies(nil)
	_ = r.Run(fmt.Sprintf("0.0.0.0:%d", params.port))
}
