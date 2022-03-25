package avail_go

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBalanceTransferExample(t *testing.T) {

	conn, err := MakeConnection()
	require.NoError(t, err)

	var ee *ExtrinsicExecutor
	ee, err = MakeExtrinsicExecutor(conn, signature.TestKeyringPairAlice, 0)
	require.NoError(t, err)

	bob, err := types.NewMultiAddressFromHexAccountID("0x8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48")
	require.NoError(t, err)

	amount := types.NewUCompactFromUInt(12345)

	var retHash types.Hash
	retHash, err = ee.ExecWait("Balances.transfer", false, bob, amount)
	require.NoError(t, err)
	println(retHash.Hex())

}
