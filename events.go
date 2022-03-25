package avail_go

import "github.com/centrifuge/go-substrate-rpc-client/v4/types"

type EventDAApplicationKeyCreated struct {
	Phase  types.Phase
	Key    types.Hash // TODO: size defined in pallet?
	Owner  types.AccountID
	Id     AppID
	Topics []types.Hash
}

type AvailEventRecords struct {
	types.EventRecords
	DataAvailability_ApplicationKeyCreated []EventDAApplicationKeyCreated
}
