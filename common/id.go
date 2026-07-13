package common

import "github.com/bwmarrin/snowflake"

// The application uses a single node because it is a single-tenant, single-process self-hosted app
var defaultNode = mustNewSnowflakeNode(1)

type ID int64

func NewID() ID {
	return ID(defaultNode.Generate().Int64())
}

func (id ID) IsZero() bool {
	return id == 0
}

func (id ID) Int64() int64 {
	return int64(id)
}

func (id ID) String() string {
	return snowflake.ID(id).String()
}

func MustIDFromString(value string) ID {
	id, err := snowflake.ParseString(value)
	if err != nil {
		panic(err)
	}
	return ID(id.Int64())
}
func IDFromString(value string) (ID, error) {
	id, err := snowflake.ParseString(value)
	if err != nil {
		return 0, err
	}
	return ID(id.Int64()), nil
}

func mustNewSnowflakeNode(nodeId int64) *snowflake.Node {
	node, err := snowflake.NewNode(nodeId)
	if err != nil {
		panic(err)
	}
	return node
}
