export function validateRequest(request) {
  if (!/^[a-f0-9]{24}$/.test(request.workspace || '')) throw new Error('Invalid workspace');
  if (!request.input || typeof request.input !== 'object' || Array.isArray(request.input)) throw new Error('Invalid tool input');
  if (!['Bash', 'Python', 'Read', 'Write', 'Edit', 'MultiEdit', 'LS', 'Glob', 'Grep', 'WebFetch', 'MCP'].includes(request.name)) throw new Error('Unsupported E2B tool');
  if (request.input.run_in_background) throw new Error('Background commands are unavailable in E2B mode');
  if (request.input.timeout_ms !== undefined && (!Number.isInteger(request.input.timeout_ms) || request.input.timeout_ms <= 0)) throw new Error('Invalid timeout');
}

export function buildCommand(request, cwd, timeout) {
  validateRequest(request);
  // The entire request is encoded as a single shell-safe token. Generated code
  // is interpreted only by the dispatcher inside the remote sandbox.
  const payload = Buffer.from(JSON.stringify({ ...request, cancelPath: undefined, cwd, timeout })).toString('base64');
  return `python3 .blacksand-tools.py ${payload}`;
}
