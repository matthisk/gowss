package websocket

// type PayloadReader struct {
// 	c      *Conn
// 	header FrameHeader
// }

// // The message types are defined in RFC 6455, section 11.8.
// const (
// 	// TextMessage denotes a text data message. The text message payload is
// 	// interpreted as UTF-8 encoded text data.
// 	TextMessage = 1

// 	// BinaryMessage denotes a binary data message.
// 	BinaryMessage = 2

// 	// CloseMessage denotes a close control message. The optional message
// 	// payload contains a numeric code and text. Use the FormatCloseMessage
// 	// function to format a close message payload.
// 	CloseMessage = 8

// 	// PingMessage denotes a ping control message. The optional message payload
// 	// is UTF-8 encoded text.
// 	PingMessage = 9

// 	// PongMessage denotes a ping control message. The optional message payload
// 	// is UTF-8 encoded text.
// 	PongMessage = 10
// )

// func (r *PayloadReader) ReadTextMessage() (result string, err error) {
// 	reader := r.c.br
// 	remaining := r.header.

// 	for remaining > 0 {
// 		rn, size, err := reader.ReadRune()
// 		remaining -= size
// 		result = result + rn
// 	}

// 	return result, err
// }

// func (r *PayloadReader) Read(b []byte) (n int, err error) {
// 	switch r.header.frameType {
// 	case TextMessage:
// 		c.Read(r.header.payloadLength)
// 	}
// }
