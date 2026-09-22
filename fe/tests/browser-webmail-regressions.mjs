// Run npm run build first. Uses the same optional Playwright environment
// variables as browser-mail-security.mjs; all API responses are local fixtures.
import assert from 'node:assert/strict';
import process from 'node:process';
import {readFile} from 'node:fs/promises';
import {createServer} from 'node:http';
import {extname, resolve} from 'node:path';
import {fileURLToPath} from 'node:url';

const {chromium} = await import(process.env.PMAIL_PLAYWRIGHT_MODULE || 'playwright');
const dist = fileURLToPath(new URL('../dist/', import.meta.url));
const ruleRequests = [];
const sentMessages = [];
const groups = [{id: 0, label: '收件箱', tag: '{"type":0,"status":-1}', children: []}];
const server = createServer(async (req, res) => {
    try {
        if (req.url.startsWith('/api/')) {
            let raw = '';
            for await (const chunk of req) raw += chunk;
            const body = raw ? JSON.parse(raw) : {};
            let data = [];
            let errorNo = 0;
            let errorMsg = '';
            switch (req.url) {
                case '/api/user/info': data = {account: 'admin', name: 'Admin', is_admin: true, domains: ['example.com']}; break;
                case '/api/group': data = groups; break;
                case '/api/email/list': {
                    const keyword = body.keyword || '';
                    if (keyword === 'fail') { errorNo = 100; errorMsg = '测试加载失败'; break; }
                    if (keyword === 'slow') await new Promise(done => setTimeout(done, 800));
                    const start = ((body.current_page || 1) - 1) * 15 + 1;
                    data = {total_page: 3, list: Array.from({length: Math.min(15, 32 - start)}, (_, offset) => ({
                        id: start + offset, sender: {Name: 'Sender', EmailAddress: 'sender@example.com'},
                        title: keyword ? `结果-${keyword}` : `邮件${start + offset}`, desc: '内容',
                        recipients: [{EmailAddress: 'random@a.example.com'}], datetime: '2026-09-21T09:00:00Z',
                        is_read: true, error: '', dangerous: false,
                    }))};
                    break;
                }
                case '/api/rule/get': data = [{id: 9, name: '已有规则', sort: 0, action: 1, params: '', rules: [{field: 'To', type: 'equal', rule: 'a@example.com'}]}]; break;
                case '/api/rule/add':
                case '/api/rule/update': ruleRequests.push({url: req.url, body}); break;
                case '/api/user/list': data = {total_page: 0, list: []}; break;
                case '/api/email/send': sentMessages.push(body); break;
            }
            res.setHeader('Content-Type', 'application/json');
            return res.end(JSON.stringify({errorNo, errorMsg, data}));
        }
        const pathname = new URL(req.url, 'http://localhost').pathname;
        const path = resolve(dist, '.' + (pathname === '/' ? '/index.html' : pathname));
        if (!path.startsWith(dist)) { res.writeHead(404); return res.end(); }
        const types = {'.js': 'text/javascript', '.css': 'text/css', '.html': 'text/html', '.ico': 'image/x-icon'};
        res.setHeader('Content-Type', types[extname(path)] || 'application/octet-stream');
        res.end(await readFile(path));
    } catch {
        res.writeHead(500);
        res.end('fixture error');
    }
});
await new Promise(done => server.listen(0, '127.0.0.1', done));
let browser;
try {
    browser = await chromium.launch({headless: true, executablePath: process.env.PMAIL_BROWSER_EXECUTABLE || undefined});
    const page = await browser.newPage({viewport: {width: 1280, height: 900}});
    const pageErrors = [];
    page.on('pageerror', error => pageErrors.push(error.message));
    await page.goto(`http://127.0.0.1:${server.address().port}`);
    await page.getByText('邮件1', {exact: true}).waitFor();
    await page.locator('.pagination-wrapper .btn-next').click();
    await page.getByText('邮件16', {exact: true}).waitFor();
    assert.equal(await page.locator('.pagination-wrapper').isVisible(), true);
    await page.locator('.pagination-wrapper .btn-next').click();
    await page.getByText('邮件31', {exact: true}).waitFor();
    assert.equal(await page.locator('.pagination-wrapper .btn-prev').isEnabled(), true);
    const search = page.getByPlaceholder('搜索邮件').first();
    await search.fill('slow');
    await page.waitForRequest(req => req.url().endsWith('/api/email/list') && req.postDataJSON().keyword === 'slow');
    await search.fill('latest');
    await page.getByText('结果-latest', {exact: true}).first().waitFor();
    await page.waitForTimeout(900);
    assert.equal(await page.getByText('结果-slow', {exact: true}).count(), 0);
    await search.fill('fail');
    await page.getByText('测试加载失败', {exact: true}).waitFor();
    assert.equal(await page.getByText('结果-latest', {exact: true}).count(), 0);
    await search.fill('');
    await page.getByText('邮件1', {exact: true}).waitFor();

    await page.locator('.settings-btn:visible').click();
    await page.getByRole('tab', {name: '规则', exact: true}).click();
    await page.locator('.el-tab-pane:visible .modern-table button').first().click();
    await page.getByRole('dialog').getByRole('button', {name: 'Cancel', exact: true}).click();
    await page.getByRole('button', {name: '新建收信规则', exact: true}).click();
    const ruleResponse = page.waitForResponse(response => response.url().endsWith('/api/rule/add'));
    await page.getByRole('dialog').getByRole('button', {name: '提交', exact: true}).click();
    await ruleResponse;
    assert.equal(ruleRequests.at(-1).url, '/api/rule/add');
    assert.equal(ruleRequests.at(-1).body.id, 0);

    await page.getByRole('button', {name: 'Close this dialog'}).first().click();
    await page.setViewportSize({width: 390, height: 600});
    await page.locator('.mobile-menu-btn').click();
    const settings = page.locator('.mobile-aside-drawer .settings-btn');
    await settings.waitFor({state: 'visible'});
    const bounds = await settings.boundingBox();
    assert.ok(bounds.y >= 0 && bounds.y + bounds.height <= 600, JSON.stringify(bounds));
    await page.keyboard.press('Escape');
    await settings.waitFor({state: 'hidden'});

    // Exercise the editor and Axios browser adapter against the frozen lockfile,
    // without sending any real mail or relying only on a successful bundle.
    await page.setViewportSize({width: 1280, height: 900});
    await page.goto(`http://127.0.0.1:${server.address().port}/#/editer`);
    const recipient = page.locator('.composer-form .el-select input').first();
    await recipient.fill('test@example.net');
    await page.getByRole('option', {name: 'test@example.net', exact: true}).click();
    await page.getByPlaceholder('Subject', {exact: true}).fill('Editor regression');
    await page.locator('.w-e-text-container [contenteditable="true"]').fill('中文 English 123');
    const sendResponse = page.waitForResponse(response => response.url().endsWith('/api/email/send'), {timeout: 5000}).catch(async error => {
        throw new Error(`${error.message}; page errors: ${JSON.stringify(pageErrors)}; page: ${await page.locator('body').innerText()}`);
    });
    await page.locator('.send-btn').click();
    await sendResponse;
    assert.equal(sentMessages.at(-1).from.email, 'admin@example.com');
    assert.equal(sentMessages.at(-1).to[0].email, 'test@example.net');
    assert.ok(JSON.stringify(sentMessages.at(-1)).includes('中文 English 123'));
    console.log(JSON.stringify({passed: true, desktopPages: [1, 2, 3], staleSearchIgnored: true, errorsVisible: true, newRuleAfterCancel: true, mobileSettingsWithinViewport: true, composeAndSendFixture: true}));
} finally {
    await browser?.close();
    await new Promise(done => server.close(done));
}
