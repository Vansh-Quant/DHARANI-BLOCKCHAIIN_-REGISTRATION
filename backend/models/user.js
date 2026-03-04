const mongoose = require('mongoose');

const UserSchema = new mongoose.Schema({
    fullName: String,
    email: { type: String, unique: true },
    walletAddress: String,
    // Store encrypted private key components
    encryptedKey: {
        content: String, // The actual encrypted hex
        iv: String,      // Initialization vector
        tag: String      // Authentication tag for AES-GCM
    },
    properties: [Number] // Array of Token IDs owned
});

module.exports = mongoose.model('User', UserSchema);