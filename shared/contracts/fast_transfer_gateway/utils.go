package fast_transfer_gateway

import "math/big"

func DecodeOrder(bytes []byte) FastTransferOrder {
	var order FastTransferOrder
	order.Sender = [32]byte(bytes[0:32])
	order.Recipient = [32]byte(bytes[32:64])
	order.AmountIn = new(big.Int).SetBytes(bytes[64:96])
	order.AmountOut = new(big.Int).SetBytes(bytes[96:128])
	order.Nonce = uint32(new(big.Int).SetBytes(bytes[128:132]).Uint64())
	order.SourceDomain = uint32(new(big.Int).SetBytes(bytes[132:136]).Uint64())
	order.DestinationDomain = uint32(new(big.Int).SetBytes(bytes[136:140]).Uint64())
	order.TimeoutTimestamp = new(big.Int).SetBytes(bytes[140:148]).Uint64()
	order.Data = bytes[148:]
	return order
}
