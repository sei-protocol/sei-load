package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateFixedReceiverAddresses(t *testing.T) {
	require.NoError(t, ValidateFixedReceiverAddresses(nil))
	require.NoError(t, ValidateFixedReceiverAddresses(&LoadConfig{}))
	require.NoError(t, ValidateFixedReceiverAddresses(&LoadConfig{
		Scenarios: []Scenario{
			{Name: "A", FixedReceiver: ""},
			{Name: "B", FixedReceiver: "  "},
			{Name: "C", FixedReceiver: "0x0000000000000000000000000000000000000000"},
			{Name: "D", FixedReceiver: "0xDC5b20847F43d67928F49Cd4f85D696b5A7617B5"},
		},
	}))

	err := ValidateFixedReceiverAddresses(&LoadConfig{
		Scenarios: []Scenario{{Name: "bad", FixedReceiver: "0xinvalid"}},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "fixedReceiver")

	err = ValidateFixedReceiverAddresses(&LoadConfig{
		Scenarios: []Scenario{{Name: "short", FixedReceiver: "0xabc"}},
	})
	require.Error(t, err)
}
