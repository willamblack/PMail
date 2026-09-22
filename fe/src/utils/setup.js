export function sslTypeForRequest(type, challenge) {
    return String(type) === "0" && challenge === "dns" ? "2" : String(type);
}

export function isDNSChallenge(type) {
    return String(type) === "2";
}
