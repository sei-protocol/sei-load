package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatePrewarmAccountPools(t *testing.T) {
	require.NoError(t, ValidatePrewarmAccountPools(nil, true))
	require.NoError(t, ValidatePrewarmAccountPools(&LoadConfig{}, false))
	require.NoError(t, ValidatePrewarmAccountPools(&LoadConfig{
		Accounts: &AccountConfig{SingleUseSenders: true},
	}, false))

	err := ValidatePrewarmAccountPools(&LoadConfig{
		Accounts: &AccountConfig{SingleUseSenders: true},
	}, true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "prewarm")

	err = ValidatePrewarmAccountPools(&LoadConfig{
		Scenarios: []Scenario{
			{Name: "S", Accounts: &AccountConfig{SingleUseSenders: true}},
		},
	}, true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "scenarios[0]")
}
