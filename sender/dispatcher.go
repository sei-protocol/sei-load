package sender

import (
	"context"
	"github.com/sei-protocol/sei-load/generator"
	mrand "math/rand/v2"
	"time"
)

// Run begins the dispatcher's transaction generation and sending loop, using
// the configured arrival model.
func Run(ctx context.Context, rng *mrand.Rand, gen *generator.Generator, snd TxSender) error {
	for ctx.Err() == nil {
		// TODO: make AccountRegistry a proper queue.
		// Generate a transaction from main generator
		gen.Generate(rng)
		// Send the transaction
		if err := snd.Send(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
