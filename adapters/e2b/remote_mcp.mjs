// This file executes only inside the E2B sandbox. A private Unix socket keeps
// each MCP connection (and browser state) alive between foreground tool calls.
import http from 'node:http';
import { createHash } from 'node:crypto';
import { spawn, execFileSync } from 'node:child_process';
import { existsSync, unlinkSync, readdirSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';
import { modulePath } from './.blacksand-runtime-path.mjs';
import { Client } from './.mcp-runtime/node_modules/@modelcontextprotocol/sdk/dist/esm/client/index.js';
import { StdioClientTransport } from './.mcp-runtime/node_modules/@modelcontextprotocol/sdk/dist/esm/client/stdio.js';
import { StreamableHTTPClientTransport } from './.mcp-runtime/node_modules/@modelcontextprotocol/sdk/dist/esm/client/streamableHttp.js';
import { SSEClientTransport } from './.mcp-runtime/node_modules/@modelcontextprotocol/sdk/dist/esm/client/sse.js';

const input = JSON.parse(Buffer.from(process.argv[2], 'base64').toString());
const socketPath = `/tmp/sw-mcp-${createHash('sha256').update(JSON.stringify(input.server)).digest('hex').slice(0,24)}.sock`;
if (process.argv[3] === 'serve') {
 const server = input.server;
 const client = new Client({name:'blacksand', version:'1.0'});
 let transport;
 if (server.transport === 'stdio') {
  let command = server.command;
  if (command === 'npx.cmd') command = 'npx';
  const args = [...(server.args || [])];
  if (args.some(a => a.includes('@playwright/mcp'))) {
   execFileSync('npm', ['exec', '--yes', '--package=playwright', '--', 'playwright', 'install', '--with-deps', 'chromium'], {timeout:180000, stdio:'ignore'});
   // MCP defaults to the Chrome channel; explicitly use our installed Chromium.
   if (!args.some(a => a === '--executable-path' || a.startsWith('--executable-path='))) {
    const cache = process.env.PLAYWRIGHT_BROWSERS_PATH || join(homedir(), '.cache', 'ms-playwright');
    const binaries = readdirSync(cache).filter(name => name.startsWith('chromium-')).sort().reverse()
     .flatMap(name => ['chrome-linux/chrome', 'chrome-linux64/chrome'].map(binary => join(cache, name, binary)));
    const executable = binaries.find(binary => existsSync(binary));
    if (!executable) throw new Error('Installed Chromium executable was not found inside the MCP sandbox');
    args.push('--executable-path', executable);
   }
   if (!args.some(a => a === '--browser' || a.startsWith('--browser='))) args.push('--browser', 'chromium');
   if (!args.includes('--no-sandbox')) args.push('--no-sandbox');
  }
  transport = new StdioClientTransport({command, args, env:{...process.env, ...(server.env || {})}, stderr:'ignore'});
 } else {
  const options = {requestInit:{headers:server.env || {}}};
  transport = server.transport === 'sse' ? new SSEClientTransport(new URL(server.url), options) : new StreamableHTTPClientTransport(new URL(server.url), options);
 }
 await client.connect(transport);
 const service = http.createServer(async (req,res) => {
  try {
   let raw = ''; for await (const chunk of req) {raw += chunk; if(raw.length > 2097152) throw new Error('MCP request too large');}
   const action = JSON.parse(raw);
   let result;
   if (action.method === 'tools/list') {
    const tools = []; let cursor;
    do {const page = await client.listTools(cursor ? {cursor} : {}); tools.push(...page.tools); cursor = page.nextCursor;} while(cursor);
    result = {tools};
   } else if(action.method === 'tools/call') result = await client.callTool(action.params);
   else throw new Error('Unsupported MCP operation');
   res.end(JSON.stringify(result));
  } catch(error) {res.statusCode=500; res.end(JSON.stringify({error:String(error.message)}));}
 });
 service.listen(socketPath);
} else {
 async function call() {
  return await new Promise((resolve,reject) => {
   const req = http.request({socketPath, path:'/', method:'POST'}, res => {let text=''; res.on('data',c=>text+=c); res.on('end',()=>res.statusCode===200 ? resolve(text) : reject(new Error('Remote MCP request failed: '+text)));});
   req.on('error',reject); req.end(JSON.stringify(input));
  });
 }
 if (!existsSync(socketPath)) {
  const child = spawn(process.execPath, [modulePath(import.meta), process.argv[2], 'serve'], {detached:true,stdio:'ignore'}); child.unref();
  let failed = false;
  child.on('error', () => {failed=true;}); child.on('exit', () => {failed=true;});
  for(let i=0;i<900 && !existsSync(socketPath);i++) {
   if(failed) throw new Error('MCP startup failed inside E2B. Check the server command, credentials and sandbox dependencies.');
   await new Promise(r=>setTimeout(r,250));
  }
 }
 try {console.log(await call());} catch(error) {
  // Stale sockets fail visibly; never restart an MCP browser mid-session.
  if(error.code==='ECONNREFUSED') {try{unlinkSync(socketPath);}catch{}}
  throw error;
 }
}
