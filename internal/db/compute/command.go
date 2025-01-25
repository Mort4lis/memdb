package compute

import (
	pkgmaps "github.com/Mort4lis/memdb/pkg/maps"
)

const (
	SetCommandName = "SET"
	GetCommandName = "GET"
	DelCommandName = "DEL"
)

type CommandID int

const (
	SetCommandID CommandID = iota + 1
	GetCommandID
	DelCommandID
)

var commandIDNameMapping = map[CommandID]string{
	SetCommandID: SetCommandName,
	GetCommandID: GetCommandName,
	DelCommandID: DelCommandName,
}

var nameCommandIDMapping = pkgmaps.Reverse(commandIDNameMapping)

var commandIDArgNumbersMapping = map[CommandID]int{
	SetCommandID: 2, //nolint:mnd // ignore magic number
	GetCommandID: 1,
	DelCommandID: 1,
}

func (c CommandID) String() string {
	return commandIDNameMapping[c]
}

type Query struct {
	cmdID CommandID
	args  []string
}

func (q Query) CommandID() CommandID {
	return q.cmdID
}

func (q Query) Args() []string {
	return q.args
}
