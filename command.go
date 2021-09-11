package main

type commandId int

const (
	CMD_USERS commandId = iota
	CMD_MSG
	CMD_QUIT
)

type command struct {
	id     commandId
	client *client
	args   []string
}
