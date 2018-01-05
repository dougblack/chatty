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

func (s *Server) welcome(conn net.Conn) (session *Session) {
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
	return &Session{
		User: username,
		Conn: conn,
	}
}

func (s *Server) Start() {
	addr, _ := net.ResolveTCPAddr("tcp", ":8080")
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
			for _, session := range s.Sessions {
				session.Conn.Write([]byte(message.Body))
				session.Conn.Write([]byte("\n"))
			}
		}
	}
}

func (s *Server) Stop() {
	s.Listener.Close()
}

func (s *Server) listen() {
	for {
		conn, err := s.Listener.AcceptTCP()
		if err != nil {
			panic(err)
		}
		fmt.Printf("Accepted new connection, RemoteAddr: %s\n", conn.RemoteAddr().String())
		session := s.welcome(conn)
		s.NewSessions <- session
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
