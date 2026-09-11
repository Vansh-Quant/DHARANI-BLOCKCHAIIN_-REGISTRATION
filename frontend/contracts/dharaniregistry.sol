// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract DharaniRegistry is ERC721, Ownable {
    uint256 private _nextTokenId;

    struct PropertyMetadata {
        string statusLabel;
        uint256 registrationTimestamp;
    }

    mapping(uint256 => PropertyMetadata) public properties;

    event PropertyRegistered(uint256 indexed propertyId, address indexed owner);

    constructor() ERC721("Dharani Property", "DHN") Ownable(msg.sender) {}

    function registerProperty(address owner, string memory status) external onlyOwner returns (uint256) {
        uint256 tokenId = _nextTokenId++;
        _safeMint(owner, tokenId);
        
        properties[tokenId] = PropertyMetadata({
            statusLabel: status,
            registrationTimestamp: block.timestamp
        });

        emit PropertyRegistered(tokenId, owner);
        return tokenId;
    }

    function transferProperty(address from, address to, uint256 tokenId) external onlyOwner {
        _transfer(from, to, tokenId);
    }

    function addStatusLabel(uint256 tokenId, string memory status) external onlyOwner {
        require(_ownerOf(tokenId) != address(0), "Property does not exist");
        properties[tokenId].statusLabel = status;
    }
}