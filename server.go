package chat

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

type Session struct {
	User string
	Conn net.Conn
}

type Server struct {
	Sessions    map[string]*Session
	Port        int
	Listener    *net.TCPListener
	NewMessages chan *Message
	NewSessions chan *Session
}

type Message struct {
	User string
	Body string
}

// Create a new chat server on the specified port.
func NewServer(port int) (*Server, error) {
	if port > 0 {
		return &Server{
			Port:        port,
			Sessions:    make(map[string]*Session),
			NewMessages: make(chan *Message),
			NewSessions: make(chan *Session),
		}, nil
	}
	return nil, errors.New("Invalid port")
}

func (s *Server) welcome(conn net.Conn) {
	welcomeMessage := fmt.Sprintf("Total users: %d\nusername: ", len(s.Sessions))
	conn.Write([]byte(welcomeMessage))

	rawUsername := make([]byte, 16)
	var username string
	for {
		read, _ := conn.Read(rawUsername)
		if read != 0 {
			username = strings.TrimRight(string(rawUsername), "\r\n\x00")
			break
		}
	}
	session := &Session{
		User: username,
		Conn: conn,
	}
	s.NewSessions <- session
}

// Start the server.
//
// This will both accept new connections and listen for
// messages from existing connections and broadcast messages
// to all other connections.
func (s *Server) Start() {
	portString := fmt.Sprintf(":%d", s.Port)
	addr, _ := net.ResolveTCPAddr("tcp", portString)
	ln, err := net.ListenTCP("tcp", addr)
	s.Listener = ln
	if err != nil {
		panic(err)
	}
	fmt.Printf("Listening for connections on port %d\n", s.Port)

	go s.listen()

	for {
		select {
		case session := <-s.NewSessions:
			s.Sessions[session.User] = session
			go s.handle(session)
		case message := <-s.NewMessages:
			if message.Body == "/exit" {
				s.Sessions[message.User].Conn.Close()
				delete(s.Sessions, message.User)
				break
			}
			for _, session := range s.Sessions {
				if session.User != message.User {
					msg := fmt.Sprintf("%s: %s\n", message.User, message.Body)
					session.Conn.Write([]byte(msg))
				}
			}
		}
	}
}

// Shutdown the server.
func (s *Server) Stop() {
	for _, session := range s.Sessions {
		session.Conn.Close()
	}
	s.Listener.Close()
}

func (s *Server) listen() {
	for {
		conn, err := s.Listener.AcceptTCP()
		if err != nil {
			panic(err)
		}
		fmt.Printf("New connection: %s\n", conn.RemoteAddr().String())
		go s.welcome(conn)
	}
}

func (s *Server) handle(session *Session) {
	var body string
	for {
		buf := make([]byte, 128)
		for {
			read, _ := session.Conn.Read(buf)
			if read > 0 {
				body = strings.TrimRight(string(buf), "\r\n\x00")
				break
			}
		}

		message := &Message{
			User: session.User,
			Body: body,
		}

		s.NewMessages <- message
	}
}
