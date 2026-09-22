// Optional browser regression; install Playwright in a temporary directory and
// pass PMAIL_PLAYWRIGHT_MODULE=/absolute/path/to/playwright/index.mjs.
// PMAIL_BROWSER_EXECUTABLE can point to an existing Chromium/Chrome executable.
import assert from 'node:assert/strict';
import process from 'node:process';
import {Buffer} from 'node:buffer';
import {readFile} from 'node:fs/promises';
import {createServer} from 'node:http';

const {chromium} = await import(process.env.PMAIL_PLAYWRIGHT_MODULE || 'playwright');
const image = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+j6s0AAAAASUVORK5CYII=', 'base64');
const requests = [];
const source = await readFile(new URL('../src/utils/mailBody.js', import.meta.url), 'utf8');
const server = createServer((req, res) => {
    if (req.url === '/mailBody.js') {
        res.setHeader('Content-Type', 'text/javascript');
        return res.end(source);
    }
    if (req.url === '/attachments/image') {
        requests.push({cookie: req.headers.cookie, site: req.headers['sec-fetch-site']});
        if (req.headers['sec-fetch-site'] === 'cross-site' || !req.headers.cookie?.includes('pmail=test')) {
            res.writeHead(403);
            return res.end('Forbidden');
        }
        res.setHeader('Content-Type', 'image/png');
        return res.end(image);
    }
    res.setHeader('Content-Type', 'text/html');
    if (req.url === '/target') return res.end('<script>window.linkWorks=true</script>target');
    res.setHeader('Set-Cookie', 'pmail=test; HttpOnly; SameSite=Lax; Path=/');
    res.end(`<div id="outside">Mailbox controls</div><iframe id="mail"></iframe>
<script type="module">
import {mailBodyDocument,mailBodySandbox} from '/mailBody.js';
window.renderMail = html => {const frame=document.querySelector('#mail');frame.sandbox=mailBodySandbox;frame.srcdoc=mailBodyDocument(html)};
</script>`);
});
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
const origin = `http://127.0.0.1:${server.address().port}`;
let browser;
try {
    browser = await chromium.launch({headless: true, executablePath: process.env.PMAIL_BROWSER_EXECUTABLE || undefined});
    const page = await browser.newPage();
    await page.goto(origin);
    await page.waitForFunction(() => typeof window.renderMail === 'function');
    await page.evaluate(origin => window.renderMail(`<style>#outside{display:none}</style>
<script>parent.__mailExecuted=true</script>
<img id="cid" src="/attachments/image" onload="parent.__mailExecuted=true">
<a id="good" href="${origin}/target" target="_top">Open link</a>
<a id="bad" href="javascript:parent.__mailExecuted=true">Bad link</a>`), origin);
    const frame = page.frames().find(item => item.parentFrame());
    await frame.waitForSelector('#cid');
    await frame.waitForFunction(() => document.querySelector('#cid').complete);
    assert.equal(await frame.locator('#cid').evaluate(img => img.naturalWidth), 1, `CID authorization failed: ${JSON.stringify(requests)}`);
    assert.equal(await page.locator('#outside').isVisible(), true);
    assert.equal(await page.evaluate(() => Boolean(window.__mailExecuted)), false);
    assert.equal(await frame.locator('#bad').getAttribute('href'), null);
    const popupPromise = page.waitForEvent('popup');
    await frame.locator('#good').click();
    const popup = await popupPromise;
    await popup.waitForLoadState();
    assert.equal(await popup.evaluate(() => window.opener === null && window.linkWorks === true), true);
    assert.equal(page.url(), `${origin}/`);
    console.log(JSON.stringify({passed: true, cidRequests: requests, cssIsolated: true, scriptBlocked: true, externalLinksIsolated: true}));
} finally {
    await browser?.close();
    await new Promise(resolve => server.close(resolve));
}
