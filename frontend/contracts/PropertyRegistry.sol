// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

/**
 * @title PropertyRegistry
 * @notice Tamper-evident proof layer for DHARANI.
 *
 * DHARANI performs source reconciliation and authority review off-chain.
 * This contract stores only minimal references and cryptographic proof hashes;
 * it is not a statutory land-title or registration system.
 */
contract PropertyRegistry {
    struct Property {
        uint256 propertyId;
        string propertyRef;
        address owner;
        uint256 registrationTime;
        bool verified;
        bytes32 latestVerificationHash;
    }

    struct VerificationAnchor {
        bytes32 reportHash;
        address authority;
        uint256 anchoredAt;
    }

    mapping(uint256 => Property) public properties;
    mapping(uint256 => VerificationAnchor[]) private verificationAnchors;
    mapping(address => uint256[]) private ownerProperties;
    mapping(address => bool) public authorities;

    uint256 public propertyCounter;

    event AuthorityUpdated(address indexed account, bool enabled);

    event PropertyRegistered(
        uint256 indexed propertyId,
        string propertyRef,
        address indexed owner,
        uint256 registrationTime
    );

    event PropertyVerified(
        uint256 indexed propertyId,
        address indexed authority,
        bytes32 indexed reportHash,
        uint256 anchoredAt
    );

    event VerificationAnchored(
        uint256 indexed propertyId,
        bytes32 indexed reportHash,
        address indexed authority,
        uint256 anchoredAt
    );

    event PropertyTransferred(
        uint256 indexed propertyId,
        address indexed from,
        address indexed to,
        uint256 transferTime
    );

    modifier onlyAuthority() {
        require(authorities[msg.sender], "Only authority can perform this action");
        _;
    }

    constructor() {
        authorities[msg.sender] = true;
        emit AuthorityUpdated(msg.sender, true);
    }

    /**
     * @notice Add or remove an authority account.
     * Only an existing authority can manage authorities.
     */
    function setAuthority(address account, bool enabled) external onlyAuthority {
        require(account != address(0), "Invalid authority");
        authorities[account] = enabled;
        emit AuthorityUpdated(account, enabled);
    }

    /**
     * @notice Register a property reference owned by msg.sender.
     * @dev Keep sensitive records off-chain; propertyRef should be a DHARANI ID.
     */
    function registerProperty(string calldata propertyRef) external returns (uint256) {
        require(bytes(propertyRef).length > 0, "Property reference required");

        uint256 propertyId = ++propertyCounter;

        properties[propertyId] = Property({
            propertyId: propertyId,
            propertyRef: propertyRef,
            owner: msg.sender,
            registrationTime: block.timestamp,
            verified: false,
            latestVerificationHash: bytes32(0)
        });

        ownerProperties[msg.sender].push(propertyId);

        emit PropertyRegistered(propertyId, propertyRef, msg.sender, block.timestamp);
        return propertyId;
    }

    /**
     * @notice Anchor a DHARANI verification report hash.
     * The hash is the cryptographic fingerprint of the canonical report.
     */
    function anchorVerification(
        uint256 propertyId,
        bytes32 reportHash
    ) external onlyAuthority {
        require(propertyId > 0 && propertyId <= propertyCounter, "Property does not exist");
        require(reportHash != bytes32(0), "Report hash required");

        Property storage property = properties[propertyId];
        property.verified = true;
        property.latestVerificationHash = reportHash;

        verificationAnchors[propertyId].push(
            VerificationAnchor({
                reportHash: reportHash,
                authority: msg.sender,
                anchoredAt: block.timestamp
            })
        );

        emit PropertyVerified(propertyId, msg.sender, reportHash, block.timestamp);
        emit VerificationAnchored(propertyId, reportHash, msg.sender, block.timestamp);
    }

    function getProperty(uint256 propertyId) external view returns (Property memory) {
        require(propertyId > 0 && propertyId <= propertyCounter, "Property does not exist");
        return properties[propertyId];
    }

    /**
     * @notice Return only the verification fields used by integration clients.
     * Keeping these as fixed-size values avoids client-side ABI decoding issues
     * with the dynamic string field in Property when running against local EVMs.
     */
    function getPropertyVerification(
        uint256 propertyId
    ) external view returns (bool verified, bytes32 latestVerificationHash) {
        require(propertyId > 0 && propertyId <= propertyCounter, "Property does not exist");
        Property storage property = properties[propertyId];
        return (property.verified, property.latestVerificationHash);
    }

    function getVerificationAnchors(
        uint256 propertyId
    ) external view returns (VerificationAnchor[] memory) {
        require(propertyId > 0 && propertyId <= propertyCounter, "Property does not exist");
        return verificationAnchors[propertyId];
    }

    function getOwnerProperties(address owner) external view returns (uint256[] memory) {
        return ownerProperties[owner];
    }

    /**
     * @notice Prototype transfer/readiness action.
     * This does not replace statutory registration or mutation procedures.
     */
    function transferProperty(uint256 propertyId, address newOwner) external {
        require(propertyId > 0 && propertyId <= propertyCounter, "Property does not exist");
        require(properties[propertyId].verified, "Property must be verified first");
        require(properties[propertyId].owner == msg.sender, "Only owner can transfer");
        require(newOwner != address(0), "Invalid new owner");
        require(newOwner != msg.sender, "New owner must differ");

        address previousOwner = properties[propertyId].owner;
        properties[propertyId].owner = newOwner;
        ownerProperties[newOwner].push(propertyId);

        emit PropertyTransferred(propertyId, previousOwner, newOwner, block.timestamp);
    }

    function getTotalProperties() external view returns (uint256) {
        return propertyCounter;
    }
}
