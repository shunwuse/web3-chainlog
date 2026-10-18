// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/// @title Guestbook
/// @notice A minimal on-chain guestbook: anyone can post a message, anyone can read it.
/// @dev Messages are append-only. Read them through the explicit getters below; the
///      array itself stays private so callers use `getMessage` / `getMessages`.
contract Guestbook {
    /// @notice A single guestbook entry.
    struct Message {
        address sender;
        string content;
        uint256 timestamp;
    }

    /// @notice All posted messages, in posting order.
    Message[] private messages;

    /// @notice Emitted when a message is posted. The indexer decodes this event.
    /// @param sender The address that posted the message.
    /// @param index The position of the message in the guestbook.
    /// @param content The message body.
    /// @param timestamp The block timestamp at posting time.
    event MessagePosted(address indexed sender, uint256 indexed index, string content, uint256 timestamp);

    /// @notice Post a message to the guestbook.
    /// @param content The message body; must not be empty.
    function post(string calldata content) external {
        require(bytes(content).length > 0, "Guestbook: empty content");

        messages.push(Message({sender: msg.sender, content: content, timestamp: block.timestamp}));

        emit MessagePosted(msg.sender, messages.length - 1, content, block.timestamp);
    }

    /// @notice Returns the total number of messages posted so far.
    function messageCount() external view returns (uint256) {
        return messages.length;
    }

    /// @notice Returns a single message by its index.
    /// @param index The position of the message in the guestbook.
    function getMessage(uint256 index) external view returns (Message memory) {
        require(index < messages.length, "Guestbook: index out of bounds");
        return messages[index];
    }

    /// @notice Returns a page of messages.
    /// @param offset The index of the first message to return.
    /// @param limit The maximum number of messages to return.
    /// @return page The requested slice; shorter than `limit` at the end of the list.
    function getMessages(uint256 offset, uint256 limit) external view returns (Message[] memory page) {
        uint256 total = messages.length;
        if (offset >= total || limit == 0) {
            return new Message[](0);
        }

        uint256 remaining = total - offset;
        uint256 count = limit < remaining ? limit : remaining;

        page = new Message[](count);
        for (uint256 i = 0; i < count; i++) {
            page[i] = messages[offset + i];
        }
    }
}
