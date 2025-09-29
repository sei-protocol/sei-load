// Package seievmtx provides helpers for turning signed go-ethereum
// transactions into the protobuf Cosmos tx bytes that Sei's EVM module
// expects. It mirrors the behaviour used by the on-chain RPC layer so that
// external tooling (e.g. load generators) can craft ready-to-broadcast payloads
// without needing an initialized app module or tx configuration.
package seievmtx
