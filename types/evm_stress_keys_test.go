package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvmStressRecipientAddressGolden(t *testing.T) {
	// Golden: EvmStressPrivateKey(0) EVM address (fixed stress recipient).
	require.Equal(t,
		"0xDC5b20847F43d67928F49Cd4f85D696b5A7617B5",
		EvmStressRecipientAddress().Hex(),
	)
}

func TestEvmStressSenderMatchesKeyOne(t *testing.T) {
	accs := GenerateEvmStressSenderAccounts(1)
	require.Len(t, accs, 1)
	k, err := EvmStressPrivateKey(1)
	require.NoError(t, err)
	require.Equal(t, accs[0].PrivKey.D, k.D)
}
