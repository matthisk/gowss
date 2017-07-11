package websocket

// OpError encapsulates a websocket operational error
// e.g. we receive a control frame with a payload length > 125
type OpError struct {
}

func (err *OpError) Error() string {
	return "Operation ERROR"
}
