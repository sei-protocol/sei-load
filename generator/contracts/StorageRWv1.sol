// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title StorageRWv1
/// @notice Load-generation contract whose touched storage slot is selected
///         per-tx by the caller, turning contention into a continuum. The
///         generator mods its per-tx key into `slot` over an arbitrary
///         keyspace; the contract bakes in no fixed size, so the keyspace can
///         resize with no redeploy.
/// @dev Versioned (v1) so a future StorageRWv2 can coexist on a persistent chain.
contract StorageRWv1 {
    /// @notice On-chain version marker so downstream consumers can pin to v1
    ///         when a future StorageRWv2 coexists on a persistent chain.
    uint256 public constant VERSION = 1;

    /// @notice Per-slot value store. Keyed by caller-chosen slot index.
    mapping(uint256 => uint256) public store;

    /// @notice Accumulator that makes `read`'s SLOAD non-elidable: the read
    ///         value is folded into observable state, so the compiler cannot
    ///         drop the load and the SUT still pays for the access.
    uint256 public readAccumulator;

    /// @notice Set store[slot] = value.
    /// @param slot  caller-selected slot index over its keyspace
    /// @param value value to write
    /// @param _pad  ignored calldata pad; lets callers vary tx size
    ///              independently of the key/storage logic
    function write(uint256 slot, uint256 value, bytes calldata _pad) external {
        store[slot] = value;
    }

    /// @notice Read store[slot] in a state-touching (non-view) tx. The loaded
    ///         value is accumulated into state so the SLOAD is real, costs gas,
    ///         and exercises the SUT. Deliberately NOT view/pure.
    /// @param slot caller-selected slot index over its keyspace
    /// @param _pad ignored calldata pad
    function read(uint256 slot, bytes calldata _pad) external {
        // unchecked: this is a load contract; readAccumulator is never asserted
        // on — it exists only to make the SLOAD non-elidable. The accumulator is
        // monotonic and unrecoverable, so a checked-math overflow (Panic 0x11)
        // would permanently brick every future read and silently collapse
        // goodput. Wrapping keeps every tx succeeding at constant gas forever.
        unchecked {
            readAccumulator += store[slot];
        }
    }

    /// @notice Read-modify-write store[slot] (store[slot] += 1).
    /// @param slot caller-selected slot index over its keyspace
    /// @param _pad ignored calldata pad
    function rmw(uint256 slot, bytes calldata _pad) external {
        // unchecked: overflow is unreachable in practice (a single slot would
        // need 2^256 increments), but this is a load contract whose result is
        // never asserted on — the arithmetic exists only to make the SSTORE
        // non-elidable. Wrapping matches read()'s never-revert profile and keeps
        // a clean, constant storage-I/O gas cost for the life of the chain.
        unchecked {
            store[slot] = store[slot] + 1;
        }
    }
}
