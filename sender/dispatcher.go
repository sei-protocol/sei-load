package sender

import (
	"context"
	mrand "math/rand/v2"
	"time"
	"github.com/sei-protocol/sei-load/generator"
)

// Run begins the dispatcher's transaction generation and sending loop, using
// the configured arrival model.
func Run(ctx context.Context, rng *mrand.Rand, gen generator.Generator, snd TxSender) error {
	for ctx.Err() == nil {
		// Generate a transaction from main generator
		tx, ok := gen.Generate(rng)
		if !ok {
			return nil
		}

		// Stamp before hand-off while sole owner: race-free (see LoadTx). This is
		// the back-pressured enqueue time, not a true schedule instant.
		tx.IntendedSendTime = time.Now()

		// Send the transaction
		if err := snd.Send(ctx, tx); err != nil {
			return err
		}
	}
	return nil 
}
