package avail_go

import (
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/config"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Connection struct {
	api  *gsrpc.SubstrateAPI
	meta *types.Metadata
}

func MakeConnection() (conn *Connection, err error) {

	conn = &Connection{}

	if conn.api, err = gsrpc.NewSubstrateAPI(config.Default().RPCURL); err != nil {
		return
	}
	if conn.meta, err = conn.api.RPC.State.GetMetadataLatest(); err != nil {
		return
	}

	return
}
