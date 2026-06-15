package sender

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sei-protocol/sei-load/config"
)

// TestArrivalModelConstantsMatchConfig pins the two representations of the
// arrival model together: config.ArrivalModel* (string, the CLI/config wire
// values) and sender.Arrival* (typed, the dispatcher's internal model). main.go
// bridges them via string(dispatcher.ArrivalModel()) into the run summary, so a
// drift between the two would silently mislabel a run. They live in separate
// packages on purpose (the sender core stays config-free); this test is the
// cheap drift guard in lieu of coupling the packages to share one constant.
func TestArrivalModelConstantsMatchConfig(t *testing.T) {
	require.Equal(t, config.ArrivalModelOpenLoop, string(ArrivalOpenLoop),
		"open_loop value drifted between config and sender")
	require.Equal(t, config.ArrivalModelClosedLoop, string(ArrivalClosedLoop),
		"closed_loop value drifted between config and sender")
}
