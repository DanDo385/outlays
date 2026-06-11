// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.28;

import {Test} from "forge-std/Test.sol";
import {AnchorRegistry} from "../src/AnchorRegistry.sol";

contract AnchorRegistryTest is Test {
    AnchorRegistry internal reg;

    // A run UUID (16 bytes) occupying the high-order bytes of bytes32 (D31 encoding).
    bytes32 internal constant RUN_ID = bytes32(bytes16(0x0f60dd6dfdfc4f96a1a4dfa6cfc52dbf));
    bytes32 internal constant ROOT = keccak256("a real D31 root goes here in production");

    event Anchored(bytes32 indexed runId, bytes32 merkleRoot, string uri, address indexed submitter);

    function setUp() public {
        reg = new AnchorRegistry();
    }

    function testAnchorStoresAndEmits() public {
        vm.expectEmit(true, true, true, true, address(reg));
        emit Anchored(RUN_ID, ROOT, "outlays://run/0f60dd6d-fdfc-4f96-a1a4-dfa6cfc52dbf", address(this));
        reg.anchor(RUN_ID, ROOT, "outlays://run/0f60dd6d-fdfc-4f96-a1a4-dfa6cfc52dbf");

        (bytes32 root, string memory uri, address submitter, uint64 anchoredAt) = reg.get(RUN_ID);
        assertEq(root, ROOT, "stored root");
        assertEq(uri, "outlays://run/0f60dd6d-fdfc-4f96-a1a4-dfa6cfc52dbf", "stored uri");
        assertEq(submitter, address(this), "stored submitter");
        assertEq(anchoredAt, uint64(block.timestamp), "stored timestamp");
        assertTrue(reg.isAnchored(RUN_ID), "isAnchored");
    }

    function testDuplicateRunIdRejected() public {
        reg.anchor(RUN_ID, ROOT, "uri-1");
        // A second anchor for the same run must revert even with a different root/uri —
        // corrections are new runs, never rewrites.
        vm.expectRevert(abi.encodeWithSelector(AnchorRegistry.AlreadyAnchored.selector, RUN_ID));
        reg.anchor(RUN_ID, keccak256("different root"), "uri-2");
    }

    function testZeroRootRejected() public {
        vm.expectRevert(AnchorRegistry.EmptyRoot.selector);
        reg.anchor(RUN_ID, bytes32(0), "uri");
    }

    function testUnanchoredReadsZero() public view {
        (bytes32 root,,, uint64 anchoredAt) = reg.get(bytes32(uint256(1)));
        assertEq(root, bytes32(0), "unset root sentinel");
        assertEq(anchoredAt, 0, "unset timestamp");
        assertFalse(reg.isAnchored(bytes32(uint256(1))), "not anchored");
    }

    function testDistinctRunsAnchorIndependently(bytes32 otherRun, bytes32 otherRoot) public {
        vm.assume(otherRun != RUN_ID);
        vm.assume(otherRoot != bytes32(0));
        reg.anchor(RUN_ID, ROOT, "uri-a");
        reg.anchor(otherRun, otherRoot, "uri-b");
        (bytes32 root,,,) = reg.get(otherRun);
        assertEq(root, otherRoot, "independent storage");
    }
}
