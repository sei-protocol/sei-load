// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Simple interface for static calls
interface ISimpleContract {
    function getValue() external view returns (uint256);
}

contract StaticCallHeavy {
    mapping(address => uint256) public counters;
    address public targetContract;

    constructor() {
        // No initialization needed for mapping
    }

    // Set the target contract for static calls
    function setTargetContract(address _targetContract) external {
        targetContract = _targetContract;
    }

    // Simple function that performs many static calls
    function performStaticCalls() external {
        // Perform 100 simple static calls
        for (uint256 i = 0; i < 100; i++) {
            if (targetContract != address(0)) {
                try ISimpleContract(targetContract).getValue() returns (uint256 value) {
                    // Just use the value to prevent optimization
                    counters[msg.sender] += value;
                } catch {
                    // If call fails, just increment counter
                    counters[msg.sender]++;
                }
            } else {
                // If no target contract, just increment counter
                counters[msg.sender]++;
            }
        }
    }
}
