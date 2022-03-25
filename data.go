package avail_go

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type AppID types.U32

type ApplicationData struct {
	Connection
	appId AppID
}

type AppKeyInfo struct {
	Owner types.AccountID
	AppID AppID
}

const MaxAppKeyLength = 32

type AppKey []byte

func (ak AppKey) MustEncode() []byte {
	var buffer bytes.Buffer
	d := scale.NewEncoder(&buffer)
	if err := d.Encode(ak); err != nil {
		panic(err)
	} else {
		return buffer.Bytes()
	}
}

func MustAppKeyFromHex(hexStr string) (ak AppKey) {
	if b, err := hex.DecodeString(hexStr); err != nil || len(b) > MaxAppKeyLength {
		panic(err)
	} else {
		ak = b
	}
	return
}

func CreateApplication(ee *ExtrinsicExecutor, appKey AppKey) (appData *ApplicationData, err error) {

	var retHash types.Hash
	checkFailure := false // TODO: would prefer true :/ - see TODO.md
	if retHash, err = ee.ExecWait("DataAvailability.create_application_key", checkFailure, appKey); err != nil {
		return
	}
	LogDebug(fmt.Sprintf("successfully created application key with transaction %v", retHash.Hex()))

	// ideally *should* be able to get app ID from the emitted event, but that's not working at the moment.
	// but this demonstrates how to get the resultant data from the storage map.
	var key []byte
	if key, err = types.CreateStorageKey(ee.meta, "DataAvailability", "AppKeys", appKey.MustEncode()); err != nil {
		return
	} else {
		LogDebug(fmt.Sprintf("encoded key for app keys: %v", hex.EncodeToString(key)))
		var ok bool
		var keyInfo AppKeyInfo
		if ok, err = ee.api.RPC.State.GetStorageLatest(key, &keyInfo); err != nil {
			return
		} else if !ok {
			// no storage data at key
			err = fmt.Errorf("should not happen - no info for provided key")
			return
		}
		appData = &ApplicationData{
			Connection: ee.Connection,
			appId:      keyInfo.AppID,
		}
	}
	return
}

func GetAppData(conn *Connection, appId AppID) (p *ApplicationData, err error) {
	// TODO!
	return
}
