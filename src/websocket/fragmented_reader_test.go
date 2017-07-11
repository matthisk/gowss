package websocket

import (
	"io"
	"testing"

	"bytes"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type handlerMock struct {
	mock.Mock
}

func (h *handlerMock) ReadMessage() (byte, []byte, error) {
	return 0x0, []byte{}, nil
}

func (h *handlerMock) WriteMessage(byte, []byte) error {
	return nil
}

func (h *handlerMock) CloseConnection(int, string) error {
	return nil
}

func (h *handlerMock) NextReader() (byte, io.Reader, error) {
	args := h.MethodCalled("NextReader")
	return byte(args.Int(0)), args.Get(1).(io.Reader), args.Error(2)
}

func (h *handlerMock) NextWriter(byte, int64) (io.Writer, error) {
	buf := make([]byte, 1024)

	return bytes.NewBuffer(buf), nil
}

func TestFragmentReaderOneFrameExactly(t *testing.T) {
	testPayload := "What up"

	reader := strings.NewReader(testPayload)
	mock := &handlerMock{}

	mock.On("NextReader").Return(0x0, strings.NewReader(testPayload), nil)

	fr := fragmentReader{reader, mock}

	buf := make([]byte, len(testPayload))
	fr.Read(buf)

	assert.Equal(t, testPayload, string(buf))
}
func TestFragmentedReaderTwoFramesExactly(t *testing.T) {
	testPayload := "What up"

	reader := strings.NewReader(testPayload)
	mock := &handlerMock{}

	mock.On("NextReader").Return(0x0, strings.NewReader(testPayload), nil)

	fr := fragmentReader{reader, mock}

	buf := make([]byte, len(testPayload)*2)
	fr.Read(buf)

	assert.Equal(t, testPayload+testPayload, string(buf))
}

func TestFragmentReaderOneFramesPlusABit(t *testing.T) {
	testPayload := "What up"

	reader := strings.NewReader(testPayload)
	mock := &handlerMock{}

	mock.On("NextReader").Return(0x0, strings.NewReader(testPayload), nil)

	fr := fragmentReader{reader, mock}

	buf := make([]byte, len(testPayload)+3)
	fr.Read(buf)

	assert.Equal(t, testPayload+testPayload[:3], string(buf))
}

func TestFragmentedReaderTwoFramesPlusABit(t *testing.T) {
	testPayload := "What up"

	reader := strings.NewReader(testPayload)
	mock := &handlerMock{}

	var nextReader io.Reader
	mock.On("NextReader").Run(func(args Arguments) {
		nextReader = strings.NewReader(testPayload)
	}).Return(0x0, nextReader, nil)

	fr := fragmentReader{reader, mock}

	buf := make([]byte, len(testPayload)*2+3)
	fr.Read(buf)

	assert.Equal(t, testPayload+testPayload+testPayload[:3], string(buf))
}

func TestFragmentedReaderMultipleFrame(t *testing.T) {

}

func TestFragmentReaderMultipleFramesAtOnce(t *testing.T) {

}
