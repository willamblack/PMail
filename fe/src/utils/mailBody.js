// Never grant scripts: same-origin is needed only for authenticated CID images
// with SameSite cookies. Without scripts the document cannot escape its sandbox.
// CSS still lives in a separate document, never inside the mailbox controls.
export const mailBodySandbox = "allow-same-origin allow-popups allow-popups-to-escape-sandbox";

export function mailBodyDocument(html) {
    const document = new DOMParser().parseFromString(html || "", "text/html");
    // Links may open a normal external tab but must never retain an opener or
    // navigate the PMail tab. Reject active/relative URL schemes explicitly.
    for (const link of document.querySelectorAll("a")) {
        const href = (link.getAttribute("href") || "").trim();
        if (!/^(https?:|mailto:)/i.test(href)) link.removeAttribute("href");
        link.setAttribute("target", "_blank");
        link.setAttribute("rel", "noopener noreferrer");
    }
    // Remove navigation controls from mail, including old unsanitized messages.
    document.querySelectorAll("base, meta, link, script, iframe, frame, object, embed, form")
        .forEach(element => element.remove());
    return `<!doctype html><html><head><meta charset="utf-8">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'none'; style-src 'unsafe-inline'; img-src http: https: data:; font-src 'none'; connect-src 'none'; frame-src 'none'; object-src 'none'; form-action 'none'; base-uri 'none'">
<meta name="referrer" content="no-referrer"><base target="_blank">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>body{margin:0;padding:12px;font:16px/1.6 -apple-system,BlinkMacSystemFont,sans-serif;overflow-wrap:anywhere;background:#fff;color:#222}img{max-width:100%;height:auto}</style>
</head><body>${document.head.innerHTML}${document.body.innerHTML}</body></html>`;
}
