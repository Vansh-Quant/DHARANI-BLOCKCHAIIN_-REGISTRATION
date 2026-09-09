// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract PropertyRegistry {
    // Property structure
    struct Property {
        uint256 propertyId;
        address owner;
        string propertyName;
        string location;
        uint256 area;
        uint256 registrationTime;
        string propertyHash;
        bool verified;
    }

    // Mapping of property ID to property details
    mapping(uint256 => Property) public properties;
    mapping(address => uint256[]) public ownerProperties;

    uint256 public propertyCounter = 0;

    event PropertyRegistered(
        uint256 indexed propertyId,
        address indexed owner,
        string propertyName,
        string location,
        uint256 registrationTime
    );

    event PropertyVerified(uint256 indexed propertyId);

    event PropertyTransferred(
        uint256 indexed propertyId,
        address indexed from,
        address indexed to,
        uint256 transferTime
    );

    // Register a new property
    function registerProperty(
        string memory _propertyName,
        string memory _location,
        uint256 _area,
        string memory _propertyHash
    ) public {
        require(bytes(_propertyName).length > 0, "Property name required");
        require(_area > 0, "Area must be greater than 0");

        uint256 propertyId = propertyCounter++;

        properties[propertyId] = Property({
            propertyId: propertyId,
            owner: msg.sender,
            propertyName: _propertyName,
            location: _location,
            area: _area,
            registrationTime: block.timestamp,
            propertyHash: _propertyHash,
            verified: false
        });

        ownerProperties[msg.sender].push(propertyId);

        emit PropertyRegistered(
            propertyId,
            msg.sender,
            _propertyName,
            _location,
            block.timestamp
        );
    }

    // Verify a property
    function verifyProperty(uint256 _propertyId) public {
        require(_propertyId < propertyCounter, "Property does not exist");
        Property storage property = properties[_propertyId];
        property.verified = true;

        emit PropertyVerified(_propertyId);
    }

    // Get property details
    function getProperty(uint256 _propertyId)
        public
        view
        returns (Property memory)
    {
        require(_propertyId < propertyCounter, "Property does not exist");
        return properties[_propertyId];
    }

    // Get properties owned by an address
    function getOwnerProperties(address _owner)
        public
        view
        returns (uint256[] memory)
    {
        return ownerProperties[_owner];
    }

    // Transfer property ownership
    function transferProperty(uint256 _propertyId, address _newOwner) public {
        require(_propertyId < propertyCounter, "Property does not exist");
        require(
            properties[_propertyId].owner == msg.sender,
            "Only owner can transfer"
        );
        require(_newOwner != address(0), "Invalid new owner");

        address previousOwner = properties[_propertyId].owner;
        properties[_propertyId].owner = _newOwner;

        ownerProperties[_newOwner].push(_propertyId);

        emit PropertyTransferred(_propertyId, previousOwner, _newOwner, block.timestamp);
    }

    // Get total properties registered
    function getTotalProperties() public view returns (uint256) {
        return propertyCounter;
    }
}
