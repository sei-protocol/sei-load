// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Simple interface for static calls
interface ISimpleContract {
    function getValue() external view returns (uint256);
}

contract StaticCallHeavy {
    uint256 public counter;
    address public targetContract;

    constructor() {
        counter = 0;
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
                    counter += value;
                } catch {
                    // If call fails, just increment counter
                    counter++;
                }
            } else {
                // If no target contract, just increment counter
                counter++;
            }
        }
    }
}
