"""Runs only inside E2B. Host paths are aliases for the sandbox workspace."""
import base64
import glob
import json
import os
from pathlib import Path
import signal
import subprocess
import sys

request = json.loads(base64.b64decode(sys.argv[1]))
args = request['input']
cwd = request['cwd']
os.chdir(cwd)

def path(value='.'):
    value = value or '.'
    host = request['hostWorkDir'].replace('\\', '/').rstrip('/')
    normalized = value.replace('\\', '/')
    if normalized == host or normalized.startswith(host + '/'):
        return Path(cwd) / normalized[len(host):].lstrip('/')
    if len(normalized) > 1 and normalized[1] == ':':
        raise ValueError('Host paths outside this workspace are unavailable in E2B')
    return Path(normalized)

def run(command, stdin=None, env=None):
    proc = subprocess.Popen(command, stdin=subprocess.PIPE if stdin is not None else subprocess.DEVNULL,
                            start_new_session=True, env=env)
    def terminate(*_):
        try:
            os.killpg(proc.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        sys.exit(130)
    signal.signal(signal.SIGTERM, terminate)
    signal.signal(signal.SIGINT, terminate)
    try:
        proc.communicate(stdin, timeout=request['timeout'] / 1000)
    except subprocess.TimeoutExpired:
        os.killpg(proc.pid, signal.SIGKILL)
        proc.wait()
        print('[command timed out]', file=sys.stderr)
        sys.exit(124)
    sys.exit(proc.returncode)

name = request['name']
if name == 'Bash':
    if not args.get('command', '').strip():
        raise ValueError('command is required')
    run(['/bin/bash', '-lc', args['command']])
elif name == 'WebFetch':
    import urllib.request
    import urllib.error
    from urllib.parse import urlparse
    from html.parser import HTMLParser
    url = args['url']
    if urlparse(url).scheme not in ('http', 'https'):
        raise ValueError('WebFetch requires an HTTP(S) URL')
    try:
        response = urllib.request.urlopen(urllib.request.Request(url, headers={'User-Agent':'Blacksand/1.0'}), timeout=60)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        data = response.read(2 * 1024 * 1024 + 1)
        print(f'[{response.status}] {response.url}')
        print(str(response.headers))
        body = data[:2 * 1024 * 1024].decode('utf-8', errors='replace')
        if 'html' in response.headers.get('Content-Type', ''):
            class Extract(HTMLParser):
                def __init__(self):
                    super().__init__(); self.hidden = 0; self.text = []; self.recon = []
                def handle_starttag(self, tag, attrs):
                    if tag in ('script','style'): self.hidden += 1
                    if tag in ('a','script','link','form','input','meta'): self.recon.append(f'{tag}: {dict(attrs)}')
                def handle_endtag(self, tag):
                    if tag in ('script','style'): self.hidden = max(0, self.hidden - 1)
                def handle_data(self, text):
                    if not self.hidden and text.strip(): self.text.append(text.strip())
                def handle_comment(self, text): self.recon.append('comment: ' + text)
            parsed = Extract(); parsed.feed(body)
            print('\n'.join(parsed.text))
            if args.get('extract'): print('\nRECON\n' + '\n'.join(parsed.recon))
        else: print(body)
        if len(data) > 2 * 1024 * 1024: print('[Response truncated]')
elif name == 'MCP':
    run(['node', '.blacksand-mcp.mjs', base64.b64encode(json.dumps(args).encode()).decode()])
elif name == 'Python':
    params = args.get('params', {})
    env = dict(os.environ)
    for key, value in params.items():
        if isinstance(value, (str, int, float, bool)):
            env['TOOL_' + key.upper()] = str(value).lower() if isinstance(value, bool) else str(value)
    run(['python3', '-c', args['code']], json.dumps(params).encode(), env)
elif name == 'Read':
    data = path(args['file_path']).read_bytes()
    if b'\0' in data[:8000]:
        print(f'[binary file not shown: {len(data)} bytes]')
    else:
        lines = data.decode('utf-8', errors='replace').split('\n')
        start = max(0, args.get('offset', 1) - 1)
        limit = args.get('limit', 2000)
        limit = limit if limit > 0 else 2000
        for i in range(start, min(len(lines), start + limit)):
            print(f'{i + 1:6}\t{lines[i][:2000]}')
elif name == 'Write':
    target = path(args['file_path'])
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(args['content'])
    print(f'Wrote {target}')
elif name in ('Edit', 'MultiEdit'):
    target = path(args['file_path'])
    content = target.read_text()
    edits = args['edits'] if name == 'MultiEdit' else [args]
    if not edits:
        raise ValueError('No edits supplied')
    for edit in edits:
        old, new = edit['old_string'], edit['new_string']
        if not old or old == new:
            raise ValueError('old_string must be nonempty and differ from new_string')
        count = content.count(old)
        if count == 0 or (count > 1 and not edit.get('replace_all')):
            raise ValueError('old_string missing or ambiguous')
        content = content.replace(old, new, -1 if edit.get('replace_all') else 1)
    target.write_text(content)
    print(f'Edited {target}')
elif name == 'LS':
    for entry in sorted(path(args.get('path')).iterdir(), key=lambda p: (not p.is_dir(), p.name)):
        print(entry.name + ('/' if entry.is_dir() else f'\t{entry.stat().st_size} bytes'))
elif name == 'Glob':
    root = path(args.get('path'))
    matches = [Path(p) for p in glob.glob(str(root / args['pattern']), recursive=True) if Path(p).is_file()]
    for entry in sorted(matches, key=lambda p: p.stat().st_mtime, reverse=True):
        if not any(x in entry.parts for x in ('.git', 'node_modules', 'vendor')):
            print(entry)
elif name == 'Grep':
    command = ['rg']
    mode = args.get('output_mode', 'files_with_matches')
    if mode == 'files_with_matches': command += ['-l']
    elif mode == 'count': command += ['-c']
    elif mode != 'content': raise ValueError('Unknown Grep output_mode')
    if args.get('-i'): command += ['-i']
    if mode == 'content' and args.get('-n', True): command += ['-n']
    if args.get('multiline'): command += ['-U', '--multiline-dotall']
    for option in ('-A', '-B', '-C'):
        if args.get(option, 0) > 0: command += [option, str(args[option])]
    if args.get('glob'): command += ['-g', args['glob']]
    if args.get('type'): command += ['-t', args['type']]
    command += ['--', args['pattern'], str(path(args.get('path')))]
    result = subprocess.run(command, capture_output=True, text=True, timeout=request['timeout'] / 1000)
    if result.returncode > 1:
        print(result.stderr, file=sys.stderr)
        sys.exit(result.returncode)
    lines = result.stdout.splitlines()
    offset = max(0, args.get('offset', 0))
    limit = args.get('head_limit', 250)
    print('\n'.join(lines[offset:offset + limit] if limit > 0 else lines[offset:]))
