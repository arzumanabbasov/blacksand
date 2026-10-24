import { Sandbox } from 'e2b';
import { readFile, stat } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { buildCommand, validateRequest } from './protocol.mjs';

let raw = '';
for await (const chunk of process.stdin) {
  raw += chunk;
  if (raw.length > 2 * 1024 * 1024) throw new Error('Request too large');
}
const emit = value => process.stdout.write(`${JSON.stringify(value)}\n`);
let sandbox, timer, handle;
try {
  const request = JSON.parse(raw);
  validateRequest(request);
  const timeout = Math.min(600000, Math.max(1, request.input.timeout_ms || 120000));
  let cancelled = false;
  const checkCancelled = async () => {
    if (request.cancelPath && (await stat(request.cancelPath)).size > 0) {
      cancelled = true;
      if (sandbox) await sandbox.kill();
    }
  };
  await checkCancelled();
  if (cancelled) throw new Error('Call cancelled before execution');
  timer = setInterval(() => { checkCancelled().catch(() => {}); }, 250);
  // The public cookbook's older browser image cannot use secured envd access.
  // Use current base and install browser dependencies remotely on MCP startup.
  const requestedTemplate = process.env.BLACKSAND_E2B_TEMPLATE || 'base';
  const template = requestedTemplate === 'playwright-chromium' ? 'base' : requestedTemplate;
  sandbox = request.sandboxId
    ? await Sandbox.connect(request.sandboxId)
    : await Sandbox.create(template, {
        timeoutMs: 900000,
        allowInternetAccess: true,
        metadata: { application: 'blacksand', workspace: request.workspace },
      });
  emit({ sandboxId: sandbox.sandboxId });
  if (cancelled) { await sandbox.kill(); throw new Error('Call cancelled before execution'); }
  await sandbox.setTimeout(900000);
  const cwd = `/home/user/blacksand/${request.workspace}`;
  await sandbox.files.makeDir(cwd);
  // Trusted dispatcher is uploaded by the control plane, never executed locally.
  await sandbox.files.write(`${cwd}/.blacksand-tools.py`, await readFile(fileURLToPath(new URL('./remote_tools.py', import.meta.url)), 'utf8'));
  if (request.name === 'MCP') {
    await sandbox.files.write(`${cwd}/.blacksand-runtime-path.mjs`, await readFile(fileURLToPath(new URL('./runtime-path.mjs', import.meta.url)), 'utf8'));
    await sandbox.files.write(`${cwd}/.blacksand-mcp.mjs`, await readFile(fileURLToPath(new URL('./remote_mcp.mjs', import.meta.url)), 'utf8'));
    const install = await sandbox.commands.run('test -f .mcp-runtime/node_modules/@modelcontextprotocol/sdk/package.json || npm install --prefix .mcp-runtime --no-audit --no-fund @modelcontextprotocol/sdk@1.32.1', {cwd,timeoutMs:180000});
    if(install.exitCode) throw new Error('Cannot install MCP runtime inside E2B');
  }
  const command = buildCommand(request, cwd, timeout);
  handle = await sandbox.commands.run(command, { cwd, background: true, timeoutMs: timeout + 10000 });
  if (cancelled) await sandbox.kill();
  let result;
  try { result = await handle.wait(); }
  catch (error) {
    if (error.exitCode !== undefined) result = error;
    else {
      // A transport failure leaves process state uncertain. Destroy this
      // sandbox rather than leave generated descendants running unattended.
      await sandbox.kill().catch(() => {});
      throw error;
    }
  }
  if (cancelled) throw new Error('Call cancelled; sandbox and its commands were stopped');
  let text = `${result.stdout || ''}${result.stderr || ''}` || '(no output)';
  if (request.name === 'MCP' && text.length > 1500000) throw new Error('MCP response exceeds transport limit');
  if (request.name !== 'MCP' && text.length > 30000) {
    const output = `${cwd}/output-${crypto.randomUUID()}.txt`;
    await sandbox.files.write(output, text);
    text = `${text.slice(0, 28000)}\n[Output truncated; full output: ${output}]`;
  }
  if (result.exitCode) emit({ error: `${text}\n[exit code ${result.exitCode}]` });
  else emit({ text });
} catch (error) {
  let message = String(error.message || error);
  if (process.env.E2B_API_KEY) message = message.split(process.env.E2B_API_KEY).join('[redacted]');
  emit({ error: `E2B execution failed: ${message}. No command was run on the host.` });
} finally {
  clearInterval(timer);
}
