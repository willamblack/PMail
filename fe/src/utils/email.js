export const formatRecipient = (recipient) => {
    if (!recipient) return "";

    const name = String(recipient.Name || "").trim();
    const address = String(recipient.EmailAddress || "").trim();
    if (name && address && name.toLowerCase() !== address.toLowerCase()) {
        return `${name} <${address}>`;
    }
    return address || name;
};

export const formatRecipientList = (recipients) => {
    if (!Array.isArray(recipients)) return "";
    return recipients.map(formatRecipient).filter(Boolean).join(", ");
};
