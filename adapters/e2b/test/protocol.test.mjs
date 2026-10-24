import { test } from 'node:test';
import assert from 'node:assert/strict';
import { buildCommand, validateRequest } from '../protocol.mjs';

const request = input => ({ name: 'Bash', workspace: 'a'.repeat(24), hostWorkDir: 'C:\\work', input });
test('WebFetch and MCP input stays encoded in the remote dispatcher payload', () => {
  for (const name of ['WebFetch', 'MCP']) {
    const input = {url:'https://example.com', method:'tools/list', server:{command:'npx', args:['@playwright/mcp']}};
    const command = buildCommand({...request(input),name}, '/home/user/work', 300000);
    const decoded = JSON.parse(Buffer.from(command.split(' ')[2], 'base64'));
    assert.equal(decoded.name,name);
    assert.deepEqual(decoded.input,input);
  }
});
test('generated shell syntax is encoded, never included in bridge shell command', () => {
  const input = { command: 'echo "$(touch /tmp/escape)"; `whoami`\npython3 script.py' };
  const command = buildCommand(request(input), '/home/user/work', 1000);
  assert.match(command, /^python3 \.blacksand-tools\.py [A-Za-z0-9+/=]+$/);
  const decoded = JSON.parse(Buffer.from(command.split(' ')[2], 'base64'));
  assert.deepEqual(decoded.input, input);
});
test('unsupported background and arbitrary tools fail before E2B allocation', () => {
  assert.throws(() => validateRequest(request({ command: 'x', run_in_background: true })), /Background/);
  assert.throws(() => validateRequest({ ...request({}), name: 'shell_open' }), /Unsupported/);
  assert.throws(() => validateRequest({ ...request({}), workspace: '../../host' }), /workspace/);
  assert.throws(() => validateRequest(request({ timeout_ms: -1 })), /timeout/);
});
