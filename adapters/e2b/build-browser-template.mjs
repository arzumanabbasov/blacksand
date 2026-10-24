import { Template } from 'e2b';

if (!process.env.E2B_API_KEY?.startsWith('e2b_')) {
  throw new Error('Set E2B_API_KEY to your E2B key before building the template');
}
const template = Template().fromTemplate('base')
  .runCmd('npm install --prefix /home/user/blacksand-browser --no-audit --no-fund playwright @playwright/mcp @modelcontextprotocol/sdk@1.32.1')
  .runCmd('cd /home/user/blacksand-browser && npx playwright install --with-deps chromium');
await Template.build(template, {
  alias: 'blacksand-browser',
  cpuCount: 2,
  memoryMB: 2048,
  onBuildLogs: entry => console.log(entry.message),
});
console.log('Template ready: select blacksand-browser in System → Settings → E2B execution, then restart the backend.');
