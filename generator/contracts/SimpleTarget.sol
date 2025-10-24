// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Simple target contract for static calls
contract SimpleTarget {
    uint256 public value;

    constructor() {
        value = 42;
    }

    function getValue() external view returns (uint256) {
        return value;
    }

    function setValue(uint256 _value) external {
        value = _value;
    }
}
