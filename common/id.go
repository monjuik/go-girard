package common

import (
	"errors"

	"github.com/bwmarrin/snowflake"
)

var ErrInvalidID = errors.New("id must be positive")

// The application uses a single node because it is a single-tenant, single-process self-hosted app
var defaultNode = mustNewSnowflakeNode(1)

type ID int64

func NewID() ID {
	return ID(defaultNode.Generate().Int64())
}

func (id ID) IsZero() bool {
	return id == 0
}

func (id ID) IsValid() bool {
	return id > 0
}

func (id ID) Int64() int64 {
	return int64(id)
}

func (id ID) String() string {
	return snowflake.ID(id).String()
}

func MustIDFromString(value string) ID {
	id, err := IDFromString(value)
	if err != nil {
		panic(err)
	}
	return id
}

func IDFromString(value string) (ID, error) {
	parsed, err := snowflake.ParseString(value)
	if err != nil {
		return 0, err
	}

	id := ID(parsed.Int64())
	if !id.IsValid() {
		return 0, ErrInvalidID
	}

	return id, nil
}

func mustNewSnowflakeNode(nodeId int64) *snowflake.Node {
	node, err := snowflake.NewNode(nodeId)
	if err != nil {
		panic(err)
	}
	return node
}
