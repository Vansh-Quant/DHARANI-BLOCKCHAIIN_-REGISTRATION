import { ethers } from "ethers";

const RPC_URL = process.env.BLOCKCHAIN_RPC_URL || process.env.SEPOLIA_RPC_URL;
const PRIVATE_KEY = process.env.BLOCKCHAIN_PRIVATE_KEY || process.env.SEPOLIA_PRIVATE_KEY;
const CONTRACT_ADDRESS = process.env.PROPERTY_REGISTRY_ADDRESS;
const PROPERTY_REF = process.env.BLOCKCHAIN_PROPERTY_REF || "DHARANI-DEMO";
const REPORT_HASH = (process.env.VERIFICATION_REPORT_HASH || "").trim().toLowerCase();
const EXISTING_PROPERTY_ID = process.env.BLOCKCHAIN_PROPERTY_ID;

if (!RPC_URL || !CONTRACT_ADDRESS || !REPORT_HASH) {
  throw new Error("Set BLOCKCHAIN_RPC_URL, PROPERTY_REGISTRY_ADDRESS and VERIFICATION_REPORT_HASH. Set BLOCKCHAIN_PRIVATE_KEY for remote networks; localhost uses Hardhat RPC account #0 automatically.");
}
if (!/^0x[0-9a-f]{40}$/i.test(CONTRACT_ADDRESS)) throw new Error("Invalid PROPERTY_REGISTRY_ADDRESS");
if (!/^[0-9a-f]{64}$/.test(REPORT_HASH) && !/^0x[0-9a-f]{64}$/.test(REPORT_HASH)) throw new Error("VERIFICATION_REPORT_HASH must be a 64-character SHA-256 hex string");

const provider = new ethers.JsonRpcProvider(RPC_URL);
const isLocalhost = /127\.0\.0\.1|localhost/.test(RPC_URL);
let signer: ethers.Signer;
if (PRIVATE_KEY) signer = new ethers.Wallet(PRIVATE_KEY, provider);
else if (isLocalhost) signer = await provider.getSigner(0);
else throw new Error("BLOCKCHAIN_PRIVATE_KEY is required for non-local networks");

const abi = [
  "function registerProperty(string propertyRef) returns (uint256)",
  "function anchorVerification(uint256 propertyId, bytes32 reportHash)",
  "function getProperty(uint256 propertyId) view returns (uint256 propertyId, string propertyRef, address owner, uint256 registrationTime, bool verified, bytes32 latestVerificationHash)",
  "function getPropertyVerification(uint256 propertyId) view returns (bool verified, bytes32 latestVerificationHash)",
  "event PropertyRegistered(uint256 indexed propertyId, string propertyRef, address indexed owner, uint256 registrationTime)",
  "event PropertyVerified(uint256 indexed propertyId, address indexed authority, bytes32 indexed reportHash, uint256 anchoredAt)"
];
const contract = new ethers.Contract(CONTRACT_ADDRESS, abi, signer);
const hashBytes = REPORT_HASH.startsWith("0x") ? REPORT_HASH : `0x${REPORT_HASH}`;
const signerAddress = await signer.getAddress();
const network = await provider.getNetwork();

console.log(JSON.stringify({ step: "signer", address: signerAddress, network: isLocalhost ? "localhost" : `${network.name} (${network.chainId.toString()})` }));

let propertyId: bigint;
if (EXISTING_PROPERTY_ID) {
  propertyId = BigInt(EXISTING_PROPERTY_ID);
} else {
  const tx = await contract.registerProperty(PROPERTY_REF);
  const receipt = await tx.wait();
  const parsed = receipt?.logs.map((log: any) => { try { return contract.interface.parseLog(log); } catch { return null; } }).find((event: any) => event?.name === "PropertyRegistered");
  if (!parsed) throw new Error("PropertyRegistered event not found");
  propertyId = BigInt(parsed.args.propertyId);
  console.log(JSON.stringify({ step: "registered", propertyId: propertyId.toString(), transactionHash: tx.hash }));
}

const beforeVerification = await contract.getPropertyVerification(propertyId);
if (EXISTING_PROPERTY_ID && !beforeVerification) throw new Error("On-chain property does not exist");

const anchorTx = await contract.anchorVerification(propertyId, hashBytes);
const anchorReceipt = await anchorTx.wait();
const onChain = await contract.getPropertyVerification(propertyId);
const chainVerified = Boolean(onChain.verified);
const chainHash = String(onChain.latestVerificationHash).toLowerCase();
const expectedHash = hashBytes.toLowerCase();
if (!chainVerified || chainHash !== expectedHash) {
  throw new Error(`On-chain verification failed: verified=${chainVerified}, latestVerificationHash=${chainHash}`);
}

let propertyRef = PROPERTY_REF;
try {
  const fullProperty = await contract.getProperty(propertyId);
  propertyRef = String(fullProperty.propertyRef);
} catch {
  // The fixed-size verification getter is authoritative for proof validation.
}

console.log(JSON.stringify({
  step: "verified",
  propertyId: propertyId.toString(),
  propertyRef,
  reportHash: hashBytes,
  onChainReportHash: chainHash,
  chainVerified,
  transactionHash: anchorTx.hash,
  contractAddress: CONTRACT_ADDRESS,
  network: `${network.name} (${network.chainId.toString()})`,
  blockNumber: anchorReceipt?.blockNumber
}, null, 2));
