// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.28;

/// @title Placeholder
/// @notice Keeps `forge build` green on a clean checkout. The real AnchorRegistry
///         (anchor(runId, merkleRoot, uri) + event, duplicate-runId rejection, forge tests)
///         lands in S12 and replaces this file.
contract Placeholder {
    string public constant NOTE = "AnchorRegistry implemented in S12";
}
