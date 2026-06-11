// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.28;

/// @title AnchorRegistry — public, immutable commitments to Outlays ingestion runs (S12, D31).
/// @notice One anchor per run, forever. `runId` is the run UUID's 16 bytes occupying the
///         high-order bytes of a bytes32 (low 16 bytes zero). `merkleRoot` is the D31
///         construction: SHA-256 over the run's sorted, unique fact hashes with 0x00 leaf /
///         0x01 node domain separation and odd-node promotion (see ARCHITECTURE.md Decision
///         Log D31 — reproducible without reading the Go code). `uri` points at where the
///         underlying evidence can be fetched and re-derived.
contract AnchorRegistry {
    struct Anchor {
        bytes32 merkleRoot;
        string uri;
        address submitter;
        uint64 anchoredAt;
    }

    mapping(bytes32 => Anchor) private _anchors;

    event Anchored(bytes32 indexed runId, bytes32 merkleRoot, string uri, address indexed submitter);

    error AlreadyAnchored(bytes32 runId);
    error EmptyRoot();

    /// @notice Record a run's Merkle root. Rejects a zero root (the unset sentinel) and any
    ///         second anchor for the same runId — corrections are new runs, never rewrites
    ///         (Hard Rule 4 extends on-chain).
    function anchor(bytes32 runId, bytes32 merkleRoot, string calldata uri) external {
        if (merkleRoot == bytes32(0)) revert EmptyRoot();
        if (_anchors[runId].merkleRoot != bytes32(0)) revert AlreadyAnchored(runId);
        _anchors[runId] = Anchor({
            merkleRoot: merkleRoot,
            uri: uri,
            submitter: msg.sender,
            anchoredAt: uint64(block.timestamp)
        });
        emit Anchored(runId, merkleRoot, uri, msg.sender);
    }

    /// @notice Read a run's anchor; merkleRoot is bytes32(0) when the run is not anchored.
    function get(bytes32 runId)
        external
        view
        returns (bytes32 merkleRoot, string memory uri, address submitter, uint64 anchoredAt)
    {
        Anchor storage a = _anchors[runId];
        return (a.merkleRoot, a.uri, a.submitter, a.anchoredAt);
    }

    function isAnchored(bytes32 runId) external view returns (bool) {
        return _anchors[runId].merkleRoot != bytes32(0);
    }
}
