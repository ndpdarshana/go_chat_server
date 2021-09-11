package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type client struct {
	conn     net.Conn
	name     string
	commands chan<- command
}

func (c *client) inputReader() {
	for {
		msg, err := bufio.NewReader(c.conn).ReadString('\n')
		if err != nil {
			return
		}

		msg = strings.Trim(msg, "\r\n")
		args := strings.Split(msg, " ")
		cmd := strings.TrimSpace(args[0])

		switch cmd {
		case "/users":
			c.commands <- command{
				id:     CMD_USERS,
				client: c,
				args:   args,
			}
		case "/msg":
			c.commands <- command{
				id:     CMD_MSG,
				client: c,
				args:   args,
			}
		case "/quit":
			c.commands <- command{
				id:     CMD_QUIT,
				client: c,
				args:   args,
			}
		default:
			c.err(fmt.Errorf("unknown command %s", cmd))
		}
	}
}

func (c *client) err(err error) {
	c.conn.Write([]byte("ERROR: " + err.Error() + "\n"))
}

func (c *client) showHist(s *server) {
	iter := s.dbClient.Collection("chats").OrderBy("createdAt", firestore.Asc).Limit(100).Documents(*s.ctx)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			fmt.Println(err.Error())
			break
		}

		msg := fmt.Sprintf("%v: %v", doc.Data()["name"], doc.Data()["text"])
		c.msg(msg)
	}
}

func (c *client) msg(msg string) {
	c.conn.Write([]byte("> " + msg + "\n"))
}
