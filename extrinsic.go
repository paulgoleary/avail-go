package avail_go

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc/author"
	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"math"
	"math/big"
	"sync"
)

type ExtrinsicExecutor struct {
	Connection

	kp signature.KeyringPair

	genesisHash types.Hash
	rv          *types.RuntimeVersion

	transMtx         sync.Mutex
	lastPendingNonce uint32
}

// MakeExtrinsicExecutor ExtrinsicExecutor encapsulates the base logic of packaging and submitting a transaction (extrinsic) to Avail
// It also provides locking 'nonce management' - that is, an explicit local lock around retrieving and using the nonce
//  for the provided account. This has proven useful in some scenarios to avoid race conditions that result in transaction
//  failures, particularly when server-side code has to submit many transactions using the same account.
func MakeExtrinsicExecutor(conn *Connection, kp signature.KeyringPair) (em *ExtrinsicExecutor, err error) {

	em = &ExtrinsicExecutor{
		Connection:       *conn,
		kp:               kp,
		transMtx:         sync.Mutex{},
		lastPendingNonce: math.MaxUint32,
	}
	if em.genesisHash, err = em.api.RPC.Chain.GetBlockHash(0); err != nil {
		return
	}
	if em.rv, err = em.api.RPC.State.GetRuntimeVersionLatest(); err != nil {
		return
	}

	return
}

// this assumes we have the lock for the manager
func (em *ExtrinsicExecutor) nextNonce() (uint32, error) {
	if em.lastPendingNonce == math.MaxUint32 {
		if key, err := types.CreateStorageKey(em.meta, "System", "Account", em.kp.PublicKey, nil); err != nil {
			return 0, err
		} else {
			var accountInfo types.AccountInfo
			if ok, err := em.api.RPC.State.GetStorageLatest(key, &accountInfo); err != nil || !ok {
				return 0, fmt.Errorf("should not happen - no account or error accessing: %v", err)
			}
			em.lastPendingNonce = uint32(accountInfo.Nonce)
		}
	} else {
		em.lastPendingNonce++
	}
	return em.lastPendingNonce, nil
}

// NewCallArgs Adapted from types.NewCall in go-substrate-rpc-client to avoid mangling of args
func NewCallArgs(m *types.Metadata, call string, args []interface{}) (types.Call, error) {
	c, err := m.FindCallIndex(call)
	if err != nil {
		return types.Call{}, err
	}

	var a []byte
	for _, arg := range args {
		e, err := types.EncodeToBytes(arg)
		if err != nil {
			return types.Call{}, err
		}
		a = append(a, e...)
	}

	return types.Call{c, a}, nil
}

func (em *ExtrinsicExecutor) execPrelude(forWait bool, callName string, callArgs []interface{}) (sub *author.ExtrinsicStatusSubscription, err error) {

	em.transMtx.Lock() // lock needs to be released after the exec function is called but *before* waiting...
	defer em.transMtx.Unlock()

	var c types.Call
	if c, err = NewCallArgs(em.meta, callName, callArgs); err != nil {
		return
	}

	// TODO: just debugging?
	var checkEncode []byte
	if checkEncode, err = types.EncodeToBytes(c); err != nil {
		return
	} else {
		println(hex.EncodeToString(checkEncode))
	}

	var nonce uint32
	if nonce, err = em.nextNonce(); err != nil {
		return
	}

	o := types.SignatureOptions{
		BlockHash:          em.genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        em.genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(nonce)),
		Tip:                types.NewUCompactFromUInt(0),
		SpecVersion:        em.rv.SpecVersion,
		TransactionVersion: em.rv.TransactionVersion,
	}

	ext := types.NewExtrinsic(c)
	if err = ext.Sign(em.kp, o); err != nil {
		return
	}

	if forWait {
		if sub, err = em.api.RPC.Author.SubmitAndWatchExtrinsic(ext); err != nil {
			return
		}
		em.lastPendingNonce = math.MaxUint32 // clear last pending - will refresh next time
	} else {
		if _, err = em.api.RPC.Author.SubmitExtrinsic(ext); err != nil {
			return
		}
	}

	return
}

func findEventDefForEventID(m *types.Metadata, eventID types.EventID) (types.Text, types.Si1Variant, error) {

	if m.Version != 14 {
		panic("should not happen - metadata is currently required to be v14")
	}

	for _, mod := range m.AsMetadataV14.Pallets {
		if !mod.HasEvents {
			continue
		}
		if mod.Index != types.NewU8(eventID[0]) {
			continue
		}
		eventType := mod.Events.Type.Int64()

		if typ, ok := m.AsMetadataV14.EfficientLookup[eventType]; ok {
			if len(typ.Def.Variant.Variants) > 0 {
				for _, vars := range typ.Def.Variant.Variants {
					if uint8(vars.Index) == eventID[1] {
						return mod.Name, vars, nil
					}
				}
			}
		}
	}
	return "", types.Si1Variant{}, fmt.Errorf("module index %v out of range", eventID[0])
}

func getErrorVariant(m *types.Metadata, de types.DispatchError) string {
	if m.Version != 14 {
		panic("should not happen - metadata is currently required to be v14")
	}
	v := m.AsMetadataV14.Pallets[de.Module].Errors.Type.Int64()
	return string(m.AsMetadataV14.EfficientLookup[v].Def.Variant.Variants[de.Error].Name)
}

// TODO: this is experimental and currently not used. may want to use an approach like this if keeping our event types
//  up to date proved too fragile ...
func (em *ExtrinsicExecutor) containsEvent(eventsRaw types.EventRecordsRaw, queryModule, queryName string) (found bool, err error) {

	decoder := scale.NewDecoder(bytes.NewReader(eventsRaw))
	var n *big.Int
	if n, err = decoder.DecodeUintCompact(); err != nil {
		return
	}

	for i := uint64(0); i < n.Uint64(); i++ {
		phase := types.Phase{}
		if err = decoder.Decode(&phase); err != nil {
			return
		}

		id := types.EventID{}
		if err = decoder.Decode(&id); err != nil {
			return
		}

		var moduleName types.Text
		var eventVariant types.Si1Variant
		if moduleName, eventVariant, err = findEventDefForEventID(em.meta, id); err != nil {
			return
		}
		if string(moduleName) == queryModule && string(eventVariant.Name) == queryName {
			return true, nil
		}

		for i := range eventVariant.Fields {
			switch eventVariant.Fields[i].TypeName {
			case "DispatchInfo":
				err = decoder.Decode(&types.DispatchInfo{})
			case "T::AccountId":
				err = decoder.Decode(&types.AccountID{})
			case "T::Hash":
				err = decoder.Decode(&types.Hash{})
			case "T::Balance":
				err = decoder.Decode(&types.U128{})
			}
			if err != nil {
				return
			}
		}
		var h []types.Hash
		if err = decoder.Decode(&h); err != nil {
			return
		}
	}

	return false, nil
}

func (em *ExtrinsicExecutor) processBlockEvents(blockHash types.Hash, proc func(raw types.EventRecordsRaw) error) (err error) {
	if key, err := types.CreateStorageKey(em.meta, "System", "Events", nil); err != nil {
		return err
	} else {
		if raw, err := em.api.RPC.State.GetStorageRaw(key, blockHash); err != nil {
			return err
		} else {
			return proc(types.EventRecordsRaw(*raw))
		}
	}
}

func (em *ExtrinsicExecutor) ExecNoWait(callName string, callArgs ...interface{}) (err error) {
	_, err = em.execPrelude(false, callName, callArgs)
	return
}

func (em *ExtrinsicExecutor) ExecWait(callName string, checkFailure bool, callArgs ...interface{}) (retHash types.Hash, err error) {

	var sub *author.ExtrinsicStatusSubscription
	if sub, err = em.execPrelude(true, callName, callArgs); err != nil {
		return
	} else {
		defer sub.Unsubscribe()

		// TODO: does this wait forever...? probably...
		for {
			status := <-sub.Chan()
			if status.IsInBlock || status.IsFinalized {
				retHash = status.AsInBlock
				break
			}
		}

		if checkFailure {
			events := AvailEventRecords{}
			proc := func(raw types.EventRecordsRaw) error {
				if err = raw.DecodeEventRecords(em.meta, &events); err != nil {
					return err
				}
				return nil
			}
			if err = em.processBlockEvents(retHash, proc); err != nil {
				return
			}

			if len(events.System_ExtrinsicFailed) != 0 {
				var c types.Call
				if c, err = NewCallArgs(em.meta, callName, callArgs); err != nil {
					return
				}
				var theBlock *types.SignedBlock
				if theBlock, err = em.api.RPC.Chain.GetBlock(retHash); err != nil {
					return
				}

				var checkCallBytes []byte
				if checkCallBytes, err = types.EncodeToBytes(c); err != nil {
					return
				}
				compareCalls := func(c types.Call) bool {
					extCallBytes, _ := types.EncodeToBytes(c)
					return bytes.Equal(checkCallBytes, extCallBytes)
				}
				for i := range events.System_ExtrinsicFailed {
					if events.System_ExtrinsicFailed[i].Phase.AsApplyExtrinsic >= uint32(len(theBlock.Block.Extrinsics)) {
						err = fmt.Errorf("should not happen - failed extrinsic outside block bound")
						return
					}
					failedExt := theBlock.Block.Extrinsics[events.System_ExtrinsicFailed[i].Phase.AsApplyExtrinsic]
					// check that the failed call is what was we sent and that we sent it ...
					if compareCalls(failedExt.Method) && bytes.Equal(failedExt.Signature.Signer.AsID[:], em.kp.PublicKey) {
						err = fmt.Errorf("extrinsic '%v' failed: %v", callName,
							getErrorVariant(em.meta, events.System_ExtrinsicFailed[i].DispatchError))
						return
					}
				}
			}
		}
	}
	return
}
