import { test } from 'node:test';
import assert from 'node:assert/strict';
import { pathToFileURL } from 'node:url';
import path from 'node:path';
import { modulePath } from '../runtime-path.mjs';

test('daemon entry point works with Node 20.9 import metadata lacking filename', () => {
 const entry = path.resolve('workspace with spaces', '.blacksand-mcp.mjs');
 const meta = {url:pathToFileURL(entry).href};
 assert.equal(meta.filename,undefined);
 assert.equal(modulePath(meta),entry);
});
