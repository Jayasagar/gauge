package main

import (
	"bytes"
	"code.google.com/p/goprotobuf/proto"
	"errors"
	"fmt"
	"github.com/getgauge/common"
	"net"
	"os"
	"strconv"
	"time"
)

const (
	runnerConnectionTimeOut = time.Second * 30
)

type messageHandler interface {
	messageBytesReceived([]byte, net.Conn)
}

type dataHandlerFn func(*gaugeConnectionHandler, []byte)

type gaugeConnectionHandler struct {
	tcpListener    *net.TCPListener
	messageHandler messageHandler
}

func newGaugeConnectionHandler(port int, messageHandler messageHandler) (*gaugeConnectionHandler, error) {
	// port = 0 means GO will find a unused port

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: port})
	if err != nil {
		return nil, err
	}

	return &gaugeConnectionHandler{tcpListener: listener, messageHandler: messageHandler}, nil
}

func (connectionHandler *gaugeConnectionHandler) acceptConnection(connectionTimeOut time.Duration) (net.Conn, error) {
	errChannel := make(chan error, 1)
	connectionChannel := make(chan net.Conn, 1)

	go func() {
		connection, err := connectionHandler.tcpListener.Accept()
		if err != nil {
			errChannel <- err
		}
		if connection != nil {
			connectionChannel <- connection
		}
	}()

	select {
	case err := <-errChannel:
		return nil, err
	case conn := <-connectionChannel:
		if connectionHandler.messageHandler != nil {
			go connectionHandler.handleConnectionMessages(conn)
		}
		return conn, nil
	case <-time.After(connectionTimeOut):
		return nil, errors.New(fmt.Sprintf("Timed out connecting to %v", connectionHandler.tcpListener.Addr()))
	}
}

func (connectionHandler *gaugeConnectionHandler) handleConnectionMessages(conn net.Conn) {
	buffer := new(bytes.Buffer)
	data := make([]byte, 8192)
	for {
		n, err := conn.Read(data)
		if err != nil {
			conn.Close()
			//TODO: Move to file
			//			log.Println(fmt.Sprintf("Closing connection [%s] cause: %s", connectionHandler.conn.RemoteAddr(), err.Error()))
			return
		}

		buffer.Write(data[0:n])
		connectionHandler.processMessage(buffer, conn)
	}
}

func (connectionHandler *gaugeConnectionHandler) processMessage(buffer *bytes.Buffer, conn net.Conn) {
	for {
		messageLength, bytesRead := proto.DecodeVarint(buffer.Bytes())
		if messageLength > 0 && messageLength < uint64(buffer.Len()) {
			messageBoundary := int(messageLength) + bytesRead
			receivedBytes := buffer.Bytes()[bytesRead : messageLength+uint64(bytesRead)]
			connectionHandler.messageHandler.messageBytesReceived(receivedBytes, conn)
			buffer.Next(messageBoundary)
			if buffer.Len() == 0 {
				return
			}
		} else {
			return
		}
	}
}

func writeDataAndGetResponse(conn net.Conn, messageBytes []byte) ([]byte, error) {
	if err := write(conn, messageBytes); err != nil {
		return nil, err
	}

	return readResponse(conn)
}

func readResponse(conn net.Conn) ([]byte, error) {
	buffer := new(bytes.Buffer)
	data := make([]byte, 8192)
	for {
		n, err := conn.Read(data)
		if err != nil {
			conn.Close()
			return nil, errors.New(fmt.Sprintf("Connection closed [%s] cause: %s", conn.RemoteAddr(), err.Error()))
		}

		buffer.Write(data[0:n])

		messageLength, bytesRead := proto.DecodeVarint(buffer.Bytes())
		if messageLength > 0 && messageLength < uint64(buffer.Len()) {
			return buffer.Bytes()[bytesRead : messageLength+uint64(bytesRead)], nil
		}
	}
}

func write(conn net.Conn, messageBytes []byte) error {
	messageLen := proto.EncodeVarint(uint64(len(messageBytes)))
	data := append(messageLen, messageBytes...)
	_, err := conn.Write(data)
	return err
}

//accepts multiple connections and Handler responds to incoming messages
func (connectionHandler *gaugeConnectionHandler) handleMultipleConnections() {
	for {
		connectionHandler.acceptConnection(30 * time.Second)
	}

}

func (connectionHandler *gaugeConnectionHandler) connectionPortNumber() int {
	if connectionHandler.tcpListener != nil {
		return connectionHandler.tcpListener.Addr().(*net.TCPAddr).Port
	} else {
		return 0
	}
}

func writeGaugeMessage(message *Message, conn net.Conn) error {
	messageId := common.GetUniqueId()
	message.MessageId = &messageId

	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	return write(conn, data)
}

func getResponseForGaugeMessage(message *Message, conn net.Conn) (*Message, error) {
	messageId := common.GetUniqueId()
	message.MessageId = &messageId

	data, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}
	responseBytes, err := writeDataAndGetResponse(conn, data)
	if err != nil {
		return nil, err
	}
	responseMessage := &Message{}
	err = proto.Unmarshal(responseBytes, responseMessage)
	if err != nil {
		return nil, err
	}
	return responseMessage, err
}

func getPortFromEnvironmentVariable(portEnvVariable string) (int, error) {
	if port := os.Getenv(portEnvVariable); port != "" {
		gport, err := strconv.Atoi(port)
		if err != nil {
			return 0, errors.New(fmt.Sprintf("%s is not a valid port", port))
		}
		return gport, nil
	}
	return 0, errors.New(fmt.Sprintf("%s Environment variable not set", portEnvVariable))
}
