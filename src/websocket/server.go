package websocket

import (
	"crypto/sha1"
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"time"
)

// Server is a composite of a http.Server and websocket specific configuration
type Server struct {
	*http.Server

	HandshakeTimeout time.Duration
}

type HttpError struct {
	Code    int
	Message string
}

func (err *HttpError) Error() string {
	return err.Message
}

// HandleFunc registers the handler func for the given pattern to the DefaultServerMux
func HandleFunc(pattern string, handler func(Conn)) {
	// Wrap internal http server HandleFunc

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Handeling new HTTP connection")

		// Validate the Request to be a request for a websocket conn upgrade

		invalid := validateRequest(r)

		if invalid != nil {
			log.Println("Bad Request", invalid)

			http.Error(w, invalid.Message, invalid.Code)

			return
		}

		// Send ack handshake to client

		w.Header().Set("Upgrade", "websocket")
		w.Header().Set("Connection", "Upgrade")
		w.Header().Set("Sec-Websocket-Accept", createWebsocketSecHeader(r.Header.Get("Sec-Websocket-Key")))
		w.WriteHeader(http.StatusSwitchingProtocols)

		// Now Hijack this connection so we can send raw TCP

		hj, ok := w.(http.Hijacker)

		if !ok {
			log.Println("Webserver doesn't support http connection hijack")

			http.Error(w, "Webserver doesn't support websocket connection upgrade", http.StatusInternalServerError)
			return
		}

		conn, bufrw, err := hj.Hijack()

		if err != nil {
			log.Println("Unable to hijack http connection with error", err)

			http.Error(w, "Webserver doesn't support websocket connection upgrade", http.StatusInternalServerError)
			return
		}

		defer conn.Close()

		// Handle the Websocket Protocol on this connection

		wsConn, err := NewConn(conn, bufrw, r)

		if err != nil {
			log.Println("Unable to create Websocket Connection", err)

			http.Error(w, "Webserver doesn't support websocket connection upgrade", http.StatusInternalServerError)
		}

		handler(wsConn)
	})
}

func createWebsocketSecHeader(input string) string {
	hasher := sha1.New()

	io.WriteString(hasher, input)
	io.WriteString(hasher, websocketGUID)

	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

func validateRequest(request *http.Request) *HttpError {
	if request.Method != "GET" {
		return &HttpError{400, "Unsupported request method"}
	}

	if proto := request.ProtoMajor*10 + request.ProtoMinor; proto < 11 {
		return &HttpError{400, "Unsupported protocol version"}
	}

	if request.Host == "" {
		return &HttpError{400, "Missing required HTTP header Host"}
	}

	if request.Header.Get("Upgrade") == "" {
		return &HttpError{400, "Missing required HTTP header Upgrade"}
	}

	if request.Header.Get("Upgrade") != "websocket" {
		return &HttpError{400, "Unsupported value for header Upgrade"}
	}

	if request.Header.Get("Connection") == "" {
		return &HttpError{400, "Missing required HTTP header Upgrade"}
	}

	if request.Header.Get("Connection") != "Upgrade" {
		return &HttpError{400, "Unsupported value for header Connection"}
	}

	if request.Header.Get("Sec-Websocket-Key") == "" {
		return &HttpError{400, "Missing required HTTP header Sec-Websocket-Key"}
	}

	secWebKey := request.Header.Get("Sec-Websocket-Key")
	secWebKeyBytes, err := base64.StdEncoding.DecodeString(secWebKey)

	if len(secWebKeyBytes) != 16 {
		return &HttpError{400, "Unsupported byte length for header Sec-Websocket-Key"}
	}

	if err != nil {
		return &HttpError{400, "Unsupported value for header Sec-Websocket-Key, expected a valid base64 encoded string"}
	}

	if request.Header.Get("Origin") == "" {
		return &HttpError{400, "Missing required HTTP header Origin"}
	}

	if request.Header.Get("Sec-Websocket-Version") == "" {
		return &HttpError{400, "Missing required HTTP header Sec-Websocket-Version"}
	}

	if request.Header.Get("Sec-Websocket-Version") != "13" {
		return &HttpError{400, "Unsupported HTTP header value for Sec-Websocket-Version, expected 13"}
	}

	return nil
}

// CreateWSServer return a HTTP server that can be used to hijack net.connns
func CreateWSServer() *Server {
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: nil,

		WriteTimeout: 2 * time.Second,
	}

	return &Server{
		httpServer,
		20 * time.Millisecond,
	}
}
