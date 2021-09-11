package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

type server struct {
	listener *net.Listener
	commands chan command
	users    map[net.Addr]*client
	app      *firebase.App
	ctx      *context.Context
	dbClient *firestore.Client
}

func newServer(fbKeyPath string) *server {

	ctx := context.Background()
	sa := option.WithCredentialsFile(fbKeyPath)
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		log.Fatalln(err)
	}

	dbClient, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	return &server{
		commands: make(chan command),
		users:    make(map[net.Addr]*client),
		app:      app,
		ctx:      &ctx,
		dbClient: dbClient,
	}
}

func (s *server) router() {
	for cmd := range s.commands {
		switch cmd.id {
		case CMD_USERS:
			s.listUsers(cmd.client)
		case CMD_MSG:
			text := strings.Join(cmd.args[1:], " ")
			s.broadcast(cmd.client, cmd.client.name+": "+text)
			go s.updateDatabase(text, cmd.client.name)
		case CMD_QUIT:
			s.quit(cmd.client)
		}
	}
}

func (s *server) newClient(conn net.Conn) {
	log.Printf("New client has connected: %s", conn.RemoteAddr().String())

	conn.Write([]byte("Please enter username: "))
	username, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		conn.Write([]byte("Invalid input"))
		return
	}

	username = strings.ToLower(strings.Trim(username, "\r\n"))

	c := &client{
		conn:     conn,
		name:     username,
		commands: s.commands,
	}

	s.users[c.conn.RemoteAddr()] = c
	s.broadcast(c, fmt.Sprintf("%s has join the chat", c.name))
	c.showHist(s)
	c.msg(fmt.Sprintf("%s, Welcome to the chat", c.name))
	c.inputReader()
}

func (s *server) start(addr string) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to start server: %s", err.Error())
	}
	s.listener = &listener
	go s.router()
	log.Printf("Server started on: %s", addr)

	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Panicf("Unable to accept connection: %s", err.Error())
			continue
		}

		go s.newClient(conn)
	}
}

func (s *server) broadcast(sender *client, msg string) {
	for addr, u := range s.users {
		fmt.Println(addr)
		if addr != sender.conn.RemoteAddr() {
			u.msg(msg)
		}
	}
}

func (s *server) updateDatabase(text string, name string) {
	_, _, err := s.dbClient.Collection("chats").Add(*s.ctx, map[string]interface{}{
		"text":      text,
		"name":      name,
		"createdAt": firestore.ServerTimestamp,
	})
	if err != nil {
		fmt.Printf("Failed adding aturing: %v", err.Error())
	}
}

func (s *server) listUsers(c *client) {
	users := make([]string, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u.name)
	}

	c.msg(strings.Join(users, ", "))
}

func (s *server) quit(c *client) {
	s.broadcast(c, fmt.Sprintf("%s has left the chat", c.name))
	c.msg(fmt.Sprintf("Bye, %s!", c.name))
	delete(s.users, c.conn.RemoteAddr())
	c.conn.Close()
	log.Printf("Client has disconnected: %s", c.conn.RemoteAddr().String())
}
