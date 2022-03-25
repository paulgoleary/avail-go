package avail_go

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestApplicationData(t *testing.T) {

	conn, err := MakeConnection()
	require.NoError(t, err)

	var ee *ExtrinsicExecutor
	ee, err = MakeExtrinsicExecutor(conn, signature.TestKeyringPairAlice, 0)
	require.NoError(t, err)

	app, err := CreateApplication(ee, MustAppKeyFromHex("deadbeef"))
	require.NoError(t, err)
	require.NotNil(t, app)

}
