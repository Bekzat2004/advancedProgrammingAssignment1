package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type Message struct {
	from    string
	name    string // Added name field to store the sender's name
	payload []byte
}

type Server struct {
	listenAddr string
	ln         net.Listener
	quitch     chan struct{}
	msgch      chan Message
	logFile    *os.File
}

func NewServer(listenAddr string) *Server {

	logFile, err := os.OpenFile("data.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	return &Server{
		listenAddr: listenAddr,
		quitch:     make(chan struct{}),
		msgch:      make(chan Message, 10),
		logFile:    logFile,
	}
}

func (s *Server) logMessage(message Message) {
	timestamp := time.Now().Format(time.RFC3339)
	logEntry := fmt.Sprintf("Message from %s: %s. Received time: %s\n", message.name, string(message.payload), timestamp)

	if _, err := s.logFile.WriteString(logEntry); err != nil {
		log.Printf("Error writing to log file: %v", err)
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()
	s.ln = ln

	go s.acceptLoop()

	<-s.quitch
	close(s.msgch)
	s.logFile.Close()

	return nil
}

func (s *Server) handleCommand(conn net.Conn, cmd string) {
	cmd = strings.TrimSpace(cmd)

	if cmd == "/join" {
		fmt.Printf("User %s joined the chat.\n", conn.RemoteAddr())
	}
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			continue
		}

		fmt.Println("New connection to the server:", conn.RemoteAddr())

		go s.readLoop(conn)
	}
}

func (s *Server) readLoop(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)

	// Read the sender's name
	name, err := r.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading sender's name:", err)
		return
	}
	name = strings.TrimSpace(name)

	for {
		// Read the message
		msg, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection closed by client")
			} else {
				fmt.Println("Read error:", err)
			}
			return
		}

		if strings.HasPrefix(msg, "/") {
			s.handleCommand(conn, msg)
			continue
		}

		s.msgch <- Message{
			from:    conn.RemoteAddr().String(),
			name:    name,
			payload: []byte(msg),
		}
	}
}

func main() {
	server := NewServer(":3000")

	go func() {
		for msg := range server.msgch {
			fmt.Println("Received message from connection:", msg.from, "Sender: ", msg.name, " Message: ", string(msg.payload))
			server.logMessage(msg)
		}
	}()

	log.Fatal(server.Start())
}
