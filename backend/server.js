require('dotenv').config();
const express = require('express');
const mongoose = require('mongoose');
const cors = require('cors');
const helmet = require('helmet');
const { ethers } = require('ethers');
const User = require('./models/User');
const { encrypt } = require('./utils/cryptoService');

const app = express();

// Middleware
app.use(express.json());
app.use(cors());
app.use(helmet());

// Database Connection
mongoose.connect(process.env.MONGO_URI)
    .then(() => console.log("Dharani DB Connected"))
    .catch(err => console.error("DB Connection Error:", err));

// ---------------------------------------------------
// API: REGISTER USER (Generates Custodial Wallet)
// ---------------------------------------------------
app.post('/api/register-user', async (req, res) => {
    try {
        const { fullName, email } = req.body;

        // 1. Generate a new random Ethereum wallet
        const wallet = ethers.Wallet.createRandom();

        // 2. Encrypt the private key using our Master Key
        const encrypted = encrypt(wallet.privateKey, process.env.MASTER_ENCRYPTION_KEY);

        // 3. Save to MongoDB
        const newUser = new User({
            fullName,
            email,
            walletAddress: wallet.address,
            encryptedKey: encrypted,
            properties: []
        });

        await newUser.save();

        res.status(201).json({
            message: "User and Wallet created successfully",
            walletAddress: wallet.address
        });
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

const PORT = process.env.PORT || 5000;
app.listen(PORT, () => console.log(`Dharani Backend running on port ${PORT}`));

const abi = require('./utils/abi');

// ---------------------------------------------------
// API: REGISTER PROPERTY (Authority Mints to User)
// ---------------------------------------------------
app.post('/api/register-property', async (req, res) => {
    try {
        const { userEmail, statusLabel } = req.body;

        // 1. Find the citizen in MongoDB
        const user = await User.findOne({ email: userEmail });
        if (!user) return res.status(404).json({ error: "Citizen not found" });

        // 2. Setup connection to your local Hardhat node
        const provider = new ethers.JsonRpcProvider(process.env.RPC_URL);
        
        // 3. Setup the Platform Wallet (The Authority)
        const platformSigner = new ethers.Wallet(process.env.PLATFORM_PRIVATE_KEY, provider);
        
        // 4. Connect to the Smart Contract
        const dharaniContract = new ethers.Contract(
            process.env.CONTRACT_ADDRESS, 
            abi, 
            platformSigner
        );

        // 5. Execute on-chain minting
        console.log(`Minting property for address: ${user.walletAddress}...`);
        const tx = await dharaniContract.registerProperty(user.walletAddress, statusLabel);
        
        // 6. Wait for the transaction to be mined
        const receipt = await tx.wait();

        // 7. Extract the Token ID from the blockchain event logs
        // Note: ethers v6 uses BigInt, we convert to Number for MongoDB
        const tokenId = Number(receipt.logs[0].topics[3]); 

        // 8. Update User records in MongoDB
        user.properties.push(tokenId);
        await user.save();

        res.json({
            message: "Property successfully registered on-chain",
            tokenId: tokenId,
            transactionHash: receipt.hash,
            owner: user.walletAddress
        });

    } catch (error) {
        console.error("Registration Error:", error);
        res.status(500).json({ error: error.message });
    }
});