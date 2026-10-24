"""Contract checks for remote file operations using a disposable fixture."""
import base64
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import threading

DISPATCHER = Path(__file__).with_name('remote_tools.py')

class RemoteFiles(unittest.TestCase):
    def setUp(self):
        self.workspace = tempfile.TemporaryDirectory()
        self.addCleanup(self.workspace.cleanup)

    def call(self, name, args):
        request = dict(name=name, input=args, cwd=self.workspace.name,
                       hostWorkDir='C:/host/work', timeout=1000)
        payload = base64.b64encode(json.dumps(request).encode()).decode()
        return subprocess.run([sys.executable, str(DISPATCHER), payload], capture_output=True, text=True)

    def test_written_file_can_be_read_and_edited_across_calls(self):
        self.assertEqual(self.call('Write', dict(file_path='script.py', content='old\n')).returncode, 0)
        self.assertIn('old', self.call('Read', dict(file_path='C:/host/work/script.py')).stdout)
        self.assertEqual(self.call('Edit', dict(file_path='script.py', old_string='old', new_string='new')).returncode, 0)
        self.assertIn('new', self.call('Read', dict(file_path='script.py')).stdout)

    def test_failed_multiedit_does_not_partially_write(self):
        self.call('Write', dict(file_path='file.txt', content='original'))
        result = self.call('MultiEdit', dict(file_path='file.txt', edits=[
            dict(old_string='original', new_string='modified'),
            dict(old_string='missing', new_string='replacement')]))
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(Path(self.workspace.name, 'file.txt').read_text(), 'original')

    def test_host_path_outside_workspace_is_rejected(self):
        result = self.call('Write', dict(file_path='C:/host/secret.txt', content='x'))
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Host paths outside', result.stderr)

    def test_webfetch_http_error_and_html_recon(self):
        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(404)
                self.send_header('Content-Type', 'text/html')
                self.end_headers()
                self.wfile.write(b'<h1>Fixture</h1><!--evidence--><script>hidden()</script><a href="/next">Next</a>')
            def log_message(self, *_): pass
        server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True); thread.start()
        self.addCleanup(server.server_close); self.addCleanup(server.shutdown)
        result = self.call('WebFetch', dict(url=f'http://127.0.0.1:{server.server_port}/', extract=True))
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('[404]', result.stdout)
        self.assertIn('Fixture', result.stdout)
        self.assertIn('comment: evidence', result.stdout)
        self.assertIn('/next', result.stdout)
        self.assertNotIn('hidden()', result.stdout)

    def test_webfetch_rejects_local_file_scheme(self):
        self.assertNotEqual(self.call('WebFetch', dict(url='file:///etc/passwd')).returncode, 0)

if __name__ == '__main__':
    unittest.main()
