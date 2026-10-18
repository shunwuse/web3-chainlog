// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {Guestbook} from "../src/Guestbook.sol";

contract GuestbookTest is Test {
    Guestbook private guestbook;

    address private alice = address(0xA11CE);
    address private bob = address(0xB0B);

    // Mirrors Guestbook.MessagePosted; vm.expectEmit matches against this signature.
    event MessagePosted(address indexed sender, uint256 indexed index, string content, uint256 timestamp);

    function setUp() public {
        guestbook = new Guestbook();
    }

    function test_MessageCountStartsAtZero() public view {
        assertEq(guestbook.messageCount(), 0);
    }

    function test_PostIncrementsCount() public {
        vm.prank(alice);
        guestbook.post("hello");

        assertEq(guestbook.messageCount(), 1);
    }

    function test_PostStoresSenderContentAndTimestamp() public {
        vm.warp(1_000);
        vm.prank(alice);
        guestbook.post("hello world");

        Guestbook.Message memory message = guestbook.getMessage(0);
        assertEq(message.sender, alice);
        assertEq(message.content, "hello world");
        assertEq(message.timestamp, 1_000);
    }

    function test_PostEmitsMessagePosted() public {
        vm.warp(42);

        vm.expectEmit(true, true, false, true, address(guestbook));
        emit MessagePosted(alice, 0, "gm", 42);

        vm.prank(alice);
        guestbook.post("gm");
    }

    function test_PostRevertsOnEmptyContent() public {
        vm.expectRevert("Guestbook: empty content");
        guestbook.post("");
    }

    function test_GetMessageRevertsWhenIndexOutOfBounds() public {
        vm.expectRevert("Guestbook: index out of bounds");
        guestbook.getMessage(0);
    }

    function test_GetMessagesReturnsEmptyWhenOffsetPastEnd() public {
        _post(bob, "one");

        Guestbook.Message[] memory page = guestbook.getMessages(1, 10);
        assertEq(page.length, 0);
    }

    function test_GetMessagesReturnsEmptyWhenLimitIsZero() public {
        _post(bob, "one");

        Guestbook.Message[] memory page = guestbook.getMessages(0, 0);
        assertEq(page.length, 0);
    }

    function test_GetMessagesClampsLimitToRemaining() public {
        _post(bob, "one");
        _post(bob, "two");
        _post(bob, "three");

        Guestbook.Message[] memory page = guestbook.getMessages(1, 99);
        assertEq(page.length, 2);
        assertEq(page[0].content, "two");
        assertEq(page[1].content, "three");
    }

    function test_GetMessagesReturnsRequestedSlice() public {
        _post(bob, "one");
        _post(bob, "two");
        _post(bob, "three");
        _post(bob, "four");

        Guestbook.Message[] memory page = guestbook.getMessages(1, 2);
        assertEq(page.length, 2);
        assertEq(page[0].content, "two");
        assertEq(page[1].content, "three");
    }

    function _post(address sender, string memory content) private {
        vm.prank(sender);
        guestbook.post(content);
    }
}
