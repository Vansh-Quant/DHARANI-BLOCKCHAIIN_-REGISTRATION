import { expect } from "chai";
import { network } from "hardhat";

describe("PropertyRegistry", function () {
  async function deployFixture() {
    const { ethers } = await network.connect();
    const [authority, owner, buyer, outsider] = await ethers.getSigners();
    const registry = await ethers.deployContract("PropertyRegistry");
    await registry.waitForDeployment();
    return { ethers, registry, authority, owner, buyer, outsider };
  }

  it("makes the deployer an authority", async function () {
    const { registry, authority } = await deployFixture();
    expect(await registry.authorities(authority.address)).to.equal(true);
  });

  it("registers a property without storing sensitive record data", async function () {
    const { ethers, registry, owner } = await deployFixture();

    const tx = await registry.connect(owner).registerProperty("DHARANI-PROP-001");
    const receipt = await tx.wait();
    const block = await ethers.provider.getBlock(receipt.blockNumber);

    await expect(tx)
      .to.emit(registry, "PropertyRegistered")
      .withArgs(1n, "DHARANI-PROP-001", owner.address, block.timestamp);

    const property = await registry.getProperty(1);
    expect(property.propertyId).to.equal(1n);
    expect(property.propertyRef).to.equal("DHARANI-PROP-001");
    expect(property.owner).to.equal(owner.address);
    expect(property.verified).to.equal(false);
    expect(property.latestVerificationHash).to.equal(ethers.ZeroHash);
  });

  it("allows only an authority to anchor a verification report", async function () {
    const { ethers, registry, authority, owner, outsider } = await deployFixture();
    await registry.connect(owner).registerProperty("DHARANI-PROP-001");

    const reportHash = ethers.keccak256(ethers.toUtf8Bytes("canonical-verification-report"));

    await expect(
      registry.connect(outsider).anchorVerification(1, reportHash)
    ).to.be.revertedWith("Only authority can perform this action");

    const tx = await registry.connect(authority).anchorVerification(1, reportHash);
    const receipt = await tx.wait();
    const block = await ethers.provider.getBlock(receipt.blockNumber);

    await expect(tx)
      .to.emit(registry, "VerificationAnchored")
      .withArgs(1n, reportHash, authority.address, block.timestamp);

    const property = await registry.getProperty(1);
    expect(property.verified).to.equal(true);
    expect(property.latestVerificationHash).to.equal(reportHash);

    const anchors = await registry.getVerificationAnchors(1);
    expect(anchors).to.have.length(1);
    expect(anchors[0].reportHash).to.equal(reportHash);
    expect(anchors[0].authority).to.equal(authority.address);
  });

  it("supports multiple verification anchors for the same property", async function () {
    const { ethers, registry, authority, owner } = await deployFixture();
    await registry.connect(owner).registerProperty("DHARANI-PROP-001");

    const firstHash = ethers.keccak256(ethers.toUtf8Bytes("report-v1"));
    const secondHash = ethers.keccak256(ethers.toUtf8Bytes("report-v2"));

    await registry.connect(authority).anchorVerification(1, firstHash);
    await registry.connect(authority).anchorVerification(1, secondHash);

    const anchors = await registry.getVerificationAnchors(1);
    expect(anchors).to.have.length(2);
    expect(anchors[0].reportHash).to.equal(firstHash);
    expect(anchors[1].reportHash).to.equal(secondHash);

    const property = await registry.getProperty(1);
    expect(property.latestVerificationHash).to.equal(secondHash);
  });

  it("prevents transfer of an unverified property", async function () {
    const { registry, owner, buyer } = await deployFixture();
    await registry.connect(owner).registerProperty("DHARANI-PROP-001");

    await expect(
      registry.connect(owner).transferProperty(1, buyer.address)
    ).to.be.revertedWith("Property must be verified first");
  });

  it("allows the verified owner to transfer the property", async function () {
    const { ethers, registry, authority, owner, buyer } = await deployFixture();
    await registry.connect(owner).registerProperty("DHARANI-PROP-001");

    const reportHash = ethers.keccak256(ethers.toUtf8Bytes("verified-report"));
    await registry.connect(authority).anchorVerification(1, reportHash);

    const tx = await registry.connect(owner).transferProperty(1, buyer.address);
    const receipt = await tx.wait();
    const block = await ethers.provider.getBlock(receipt.blockNumber);

    await expect(tx)
      .to.emit(registry, "PropertyTransferred")
      .withArgs(1n, owner.address, buyer.address, block.timestamp);

    const property = await registry.getProperty(1);
    expect(property.owner).to.equal(buyer.address);
  });

  it("allows an authority to manage another authority", async function () {
    const { registry, authority, outsider } = await deployFixture();

    await registry.connect(authority).setAuthority(outsider.address, true);
    expect(await registry.authorities(outsider.address)).to.equal(true);

    await registry.connect(outsider).setAuthority(authority.address, false);
    expect(await registry.authorities(authority.address)).to.equal(false);
  });
});
