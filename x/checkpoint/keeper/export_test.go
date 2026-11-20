package keeper

import "context"

// ForgetBufferHeight lets a test install a buffer directly as if it predated every recorded install.
func (k *Keeper) ForgetBufferHeight(ctx context.Context) error {
	return k.bufferHeight.Remove(ctx)
}
