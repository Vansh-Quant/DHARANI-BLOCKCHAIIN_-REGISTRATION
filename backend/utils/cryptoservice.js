const crypto = require('crypto');
const ALGORITHM = 'aes-256-gcm';

const encrypt = (text, masterKey) => {
    const iv = crypto.randomBytes(16);
    const cipher = crypto.createCipheriv(ALGORITHM, Buffer.from(masterKey), iv);
    const encrypted = Buffer.concat([cipher.update(text, 'utf8'), cipher.final()]);
    const tag = cipher.getAuthTag();
    return {
        content: encrypted.toString('hex'),
        iv: iv.toString('hex'),
        tag: tag.toString('hex')
    };
};

module.exports = { encrypt };