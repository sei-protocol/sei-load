package scenarios

import (
	"testing"

	"github.com/sei-protocol/sei-load/config"
	"github.com/stretchr/testify/require"
)

// frozenOperations is the whole operation vocabulary. A metric query matches an
// operation by value, so the set is a contract with the dashboards and not a
// detail of any one scenario.
var frozenOperations = map[string]struct{}{
	config.OpRmw:           {},
	config.OpRead:          {},
	config.OpWrite:         {},
	config.OpTransfer:      {},
	config.OpSelfTransfer:  {},
	config.OpERC20Transfer: {},
	config.OpERC721Mint:    {},
	config.OpDisperseEther: {},
}

// TestEveryScenarioNamesAFrozenOperation asserts that every registered scenario
// returns an operation, and that the name comes from the frozen vocabulary. The
// test ranges over the factory registry, so a scenario added later is covered
// without editing this file. It is an internal test because that registry is
// what makes the coverage automatic.
//
// An operation reaches the metrics as a dimension. An empty one is dropped by
// the Prometheus exporter's collision with the omitted-attribute stream, which
// loses the sample and reports no scrape error, so the emptiness never surfaces.
func TestEveryScenarioNamesAFrozenOperation(t *testing.T) {
	require.NotEmpty(t, scenarioFactories)
	for name, factory := range scenarioFactories {
		t.Run(name, func(t *testing.T) {
			op := factory(config.Scenario{Name: name}).Operation()
			require.NotEmpty(t, op)
			require.Contains(t, frozenOperations, op)
		})
	}
}
