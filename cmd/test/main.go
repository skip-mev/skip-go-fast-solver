package main

import (
	"encoding/binary"
	"encoding/hex"
)

func main() {
	b, err := hex.DecodeString("030000003e0000036b2f55d3e9abdfc7344e50d25eb18ed182465ba7d49e88cf843f36861f08bb055c0000a4b1000000000000000000000000d6f9f9e2c231e023fe0a8d752bc4080a112a1eba0000000000000000000000008eb49e3d65d74967cc0fe987fa2d015ae816352e4b09ecc4d0a6c6d9ca59616a3a12c2325462caf9f9347061c165226069eb9fcd")
	if err != nil {
		panic(err)
	}
	originBytes := b[20:36]
	println(hex.EncodeToString(originBytes))
	println(binary.LittleEndian.Uint32(originBytes))
	println(binary.BigEndian.Uint32(originBytes))

}
