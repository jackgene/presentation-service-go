package main

import (
	"embed"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"html/template"
	"log"
	"net/http"
	"os"
	"presentation-service/internal/chat"
	"presentation-service/internal/chat/approval"
	"presentation-service/internal/transcription"
	"strings"
)

type cliParams struct {
	htmlPath string
	port     uint
}

func parseFlags() cliParams {
	params := cliParams{}

	flag.StringVar(&params.htmlPath, "html-path", "", "Presentation HTML file path")
	flag.UintVar(&params.port, "port", 8973, "HTTP server port")
	flag.Parse()

	// Required args
	if params.htmlPath == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

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
			_, _, readErr := conn.ReadMessage()
			if readErr != nil {
				if _, ok := readErr.(*websocket.CloseError); ok {
					log.Printf("connection closed by client: %v", readErr)
					closed <- struct{}{}
					close(closed)
					break
				}
				log.Printf("unexpected websocket error: %v", readErr)
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

	r := gin.Default()
	templ := template.Must(template.New("").ParseFS(fs, "public/html/*.html"))
	r.SetHTMLTemplate(templ)

	// Actors
	chatMessageActor := chat.NewBroadcasterActor("chat")
	rejectedMessageActor := chat.NewBroadcasterActor("rejected")
	transcriptionActor := transcription.NewTranscriptionActor()
	questionActor := approval.NewApprovedMessagesActor(chatMessageActor, rejectedMessageActor, 10)

	// Deck
	r.GET("/", func(c *gin.Context) {
		c.File(params.htmlPath)
	})

	r.GET("/event/question", func(c *gin.Context) {
		conn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("failed to upgrade websocket request %v", err)
			return
		}
		defer func() { _ = conn.Close() }()
		clientClosed := clientCloseListener(conn)

		msgs := make(chan approval.Messages)
		questionActor.Register(msgs)
		defer questionActor.Unregister(msgs)
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
		transcriptionActor.Register(transcripts)
		defer transcriptionActor.Unregister(transcripts)
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
		rejectedMessageActor.Register(msgs)
		defer rejectedMessageActor.Unregister(msgs)
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

		chatMessageActor.NewMessage(chat.Message{
			Sender:    sender,
			Recipient: recipient,
			Text:      c.Query("text"),
		})
		c.Status(http.StatusNoContent)
	})

	r.GET("/reset", func(c *gin.Context) {
		questionActor.Reset()
		c.Status(http.StatusNoContent)
	})

	// Transcription
	r.GET("/transcriber", func(c *gin.Context) {
		c.HTML(http.StatusOK, "transcriber.html", nil)
	})

	r.POST("/transcription", func(c *gin.Context) {
		transcriptionActor.NewTranscriptionText(c.Query("text"))
		c.Status(http.StatusNoContent)
	})

	_ = r.SetTrustedProxies(nil)
	_ = r.Run(fmt.Sprintf("0.0.0.0:%d", params.port))
}
