import assert from 'node:assert/strict';
import test from 'node:test';
import {createRequestScope} from '../src/utils/requestScope.js';
import {createRuleForm} from '../src/utils/ruleForm.js';
import {isDNSChallenge, sslTypeForRequest} from '../src/utils/setup.js';

test('a changed query invalidates the response before its debounce expires', async () => {
    const scope = createRequestScope();
    const old = scope.start();
    let resolve;
    let displayed = 'new query';
    const response = new Promise(done => { resolve = done; }).then(value => {
        if (old.isCurrent()) displayed = value;
    });
    scope.invalidate();
    resolve('old results');
    await response;
    assert.equal(displayed, 'new query');
    assert.equal(old.signal.aborted, true);
});

test('newer requests win and disposed views cannot start or receive requests', () => {
    const scope = createRequestScope();
    const old = scope.start();
    const current = scope.start();
    assert.equal(old.isCurrent(), false);
    assert.equal(old.signal.aborted, true);
    assert.equal(current.isCurrent(), true);
    scope.dispose();
    assert.equal(current.isCurrent(), false);
    assert.equal(current.signal.aborted, true);
    assert.equal(scope.start(), null);
});

test('cancelled rule edits do not mutate the list and New never reuses an id', () => {
    const saved = {id: 9, name: 'existing', rules: [{field: 'To', type: 'equal', rule: 'a@example.com'}]};
    const edited = createRuleForm(saved);
    edited.rules[0].rule = 'changed';
    edited.rules.push({field: 'Subject', type: 'equal', rule: 'extra'});
    assert.equal(saved.rules[0].rule, 'a@example.com');
    assert.equal(saved.rules.length, 1);
    Object.assign(edited, createRuleForm());
    assert.equal(edited.id, 0);
    assert.equal(edited.name, '');
    assert.deepEqual(edited.rules, [{field: '', type: '', rule: ''}]);
});

test('SSL option strings enter the DNS challenge branch', () => {
    assert.equal(sslTypeForRequest('0', 'dns'), '2');
    assert.equal(isDNSChallenge(sslTypeForRequest('0', 'dns')), true);
    assert.equal(isDNSChallenge(sslTypeForRequest('0', 'http')), false);
    assert.equal(sslTypeForRequest('1', 'dns'), '1');
    assert.equal(isDNSChallenge(2), true);
});
