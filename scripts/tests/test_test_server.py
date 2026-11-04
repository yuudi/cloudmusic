"""Check test-server.sh lifecycle with fake external tools; see scripts/README.md."""

import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import time
import unittest


HELPER = Path(__file__).resolve().parents[1] / 'test-server.sh'
HURL_DIR = HELPER.parent.parent / 'apps/server/tests/hurl'
STUB = '''#!{python}
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys

state = Path(os.environ['TEST_STATE'])
tool = Path(sys.argv[0]).name
mode = os.environ.get('TEST_MODE', '')
args = sys.argv[1:]

def record(event, **values):
    with (state / 'events').open('a') as output:
        output.write(json.dumps(dict(event=event, **values)) + '\\n')

if tool == 'python3':
    if args[0] == '-':
        program = sys.stdin.read()
        if 'import socket' in program:
            print('23456')
        else:
            sys.exit(subprocess.run([{python!r}, *args], input=program, text=True).returncode)
    else:
        os.execv({python!r}, [{python!r}, *args])
elif tool == 'go':
    target = Path(args[args.index('-o') + 1])
    shutil.copyfile(sys.argv[0], target)
    target.chmod(0o755)
elif tool == 'psql':
    query = args[args.index('-c') + 1]
    operation = query.split()[0]
    record(operation, query=query)
    if operation == 'CREATE' and mode == 'create_failure':
        sys.exit(41)
    if operation == 'DROP' and mode == 'drop_failure':
        sys.exit(43)
    if operation == 'DROP':
        (state / 'server_ready').unlink(missing_ok=True)
elif tool == 'migrate':
    record('migrate', url=os.environ['DATABASE_URL'])
    if mode == 'migration_failure':
        print('migration diagnostic')
        sys.exit(42)
elif tool == 'server':
    record('server', pid=os.getpid(), url=os.environ['DATABASE_URL'])
    if mode == 'server_failure':
        print('server diagnostic', flush=True)
        sys.exit(44)
    print('server runtime diagnostic', flush=True)
    signal.signal(signal.SIGTERM, lambda *_: sys.exit(0))
    (state / 'server_ready').touch()
    while True:
        signal.pause()
elif tool == 'curl':
    (state / 'helper_pid').write_text(str(os.getppid()))
    sys.exit(0 if (state / 'server_ready').exists() else 1)
elif tool == 'hurl':
    record('hurl', scenario=Path(args[-1]).name)
    if mode == 'hurl_failure':
        sys.exit(23)
'''


class TestServerLifecycle(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.bin = self.root / 'bin'
        self.bin.mkdir()
        self.run_dir = self.root / 'tmp'
        self.run_dir.mkdir()
        for tool in ('go', 'psql', 'curl', 'python3', 'hurl'):
            path = self.bin / tool
            path.write_text(STUB.format(python=sys.executable))
            path.chmod(0o755)
        self.env = {
            **os.environ,
            'PATH': f'{self.bin}:{os.environ["PATH"]}',
            'TMPDIR': str(self.run_dir),
            'TEST_STATE': str(self.root),
            'TEST_DATABASE_URL': 'postgres://localhost/test?sslmode=disable&search_path=old',
            'REDIS_URL': 'redis://localhost/0',
        }

    def events(self):
        path = self.root / 'events'
        return [json.loads(line) for line in path.read_text().splitlines()] if path.exists() else []

    def run_helper(self, callback='return 0', *, mode='', after=''):
        script = f'''
set -euo pipefail
source "$1"
run_tests() {{
    {callback}
}}
with_test_server check run_tests
{after}
'''
        return subprocess.run(
            ['bash', '-c', script, 'test', str(HELPER)],
            env={**self.env, 'TEST_MODE': mode},
            capture_output=True, text=True, timeout=10,
        )

    def assert_cleaned(self, runs=1):
        events = self.events()
        created = [event['query'] for event in events if event['event'] == 'CREATE']
        dropped = [event['query'] for event in events if event['event'] == 'DROP']
        self.assertEqual(len(created), runs)
        self.assertEqual(dropped, [query.replace('CREATE SCHEMA', 'DROP SCHEMA') + ' CASCADE' for query in created])
        self.assertEqual(list(self.run_dir.iterdir()), [])
        for event in events:
            if event['event'] == 'server':
                self.assert_stopped(event['pid'])

    def assert_stopped(self, pid):
        # A terminated grandchild may briefly await reaping by the container's init.
        path = Path(f'/proc/{pid}/stat')
        self.assertTrue(not path.exists() or path.read_text().split()[2] == 'Z', f'{pid} still running')

    def test_source_has_no_side_effects(self):
        result = subprocess.run(
            ['bash', '-c', 'set -eu; before=$(set +o); source "$1"; [[ "$(set +o)" == "$before" ]]; [[ -z "$(trap -p)" ]]',
             'test', str(HELPER)],
            env=self.env, capture_output=True, text=True, timeout=5,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.events(), [])
        self.assertEqual(list(self.run_dir.iterdir()), [])

    def test_success_isolated_url_and_parent_scope(self):
        self.env['CLOUDMUSIC_API_URL'] = 'parent-value'
        result = self.run_helper(
            '[[ "$CLOUDMUSIC_API_URL" == http://127.0.0.1:23456 ]]; cd /',
            after='[[ "$CLOUDMUSIC_API_URL" == parent-value ]]; [[ -z "$(trap -p)" ]]; [[ "$-" != *m* ]]',
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertNotIn('server runtime diagnostic', result.stderr)
        url = next(event['url'] for event in self.events() if event['event'] == 'migrate')
        self.assertIn('sslmode=disable&search_path=check_', url)
        self.assertNotIn('search_path=old', url)
        self.assert_cleaned()

    def test_existing_signal_handlers_are_restored(self):
        result = subprocess.run(
            ['bash', '-c', '''
set -euo pipefail
source "$1"
trap 'echo original-int' INT
trap 'echo original-term' TERM
before=$(trap -p INT TERM)
with_test_server check true
[[ "$(trap -p INT TERM)" == "$before" ]]
''', 'test', str(HELPER)],
            env=self.env, capture_output=True, text=True, timeout=10,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assert_cleaned()

    def test_callback_failure_preserves_exit_code(self):
        result = self.run_helper('return 23')
        self.assertEqual(result.returncode, 23, result.stderr)
        self.assertIn('Test server log:', result.stderr)
        self.assertIn('server runtime diagnostic', result.stderr)
        self.assert_cleaned()

    def test_hurl_scenarios_use_independent_schemas_and_servers(self):
        result = subprocess.run(
            ['bash', str(HURL_DIR / 'run.sh')], env=self.env,
            capture_output=True, text=True, timeout=20,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        scenarios = sorted(path.name for path in HURL_DIR.glob('*.hurl'))
        self.assertTrue(scenarios)
        events = self.events()
        self.assertEqual([event['scenario'] for event in events if event['event'] == 'hurl'], scenarios)
        servers = [event for event in events if event['event'] == 'server']
        self.assertEqual(len(servers), len(scenarios))
        self.assertEqual(len({event['url'] for event in servers}), len(scenarios))
        self.assertEqual(len({event['pid'] for event in servers}), len(scenarios))
        for scenario in scenarios:
            self.assertIn(f'Running API E2E: {scenario}', result.stdout)
        self.assertNotIn('server runtime diagnostic', result.stderr)
        self.assert_cleaned(runs=len(scenarios))

    def test_hurl_failure_stops_later_scenarios_and_outputs_server_logs(self):
        result = subprocess.run(
            ['bash', str(HURL_DIR / 'run.sh')], env={**self.env, 'TEST_MODE': 'hurl_failure'},
            capture_output=True, text=True, timeout=10,
        )
        self.assertEqual(result.returncode, 23, result.stderr)
        first = min(path.name for path in HURL_DIR.glob('*.hurl'))
        self.assertEqual([event['scenario'] for event in self.events() if event['event'] == 'hurl'], [first])
        self.assertIn('Test server log:', result.stderr)
        self.assertIn('server runtime diagnostic', result.stderr)
        self.assert_cleaned()

    def test_startup_failures_print_logs_and_clean_up(self):
        for mode, status, message in (
            ('migration_failure', 42, 'migration diagnostic'),
            ('server_failure', 1, 'server diagnostic'),
        ):
            with self.subTest(mode=mode):
                # Separate runs must not share ready markers or recorded resources.
                for name in ('events', 'server_ready', 'helper_pid'):
                    (self.root / name).unlink(missing_ok=True)
                result = self.run_helper('echo callback-ran', mode=mode)
                self.assertEqual(result.returncode, status, result.stderr)
                self.assertIn(message, result.stderr)
                self.assertNotIn('callback-ran', result.stdout)
                self.assert_cleaned()

    def test_schema_creation_failure_does_not_drop_schema(self):
        result = self.run_helper(mode='create_failure')
        self.assertEqual(result.returncode, 41, result.stderr)
        self.assertEqual([event['event'] for event in self.events()], ['CREATE'])
        self.assertEqual(list(self.run_dir.iterdir()), [])

    def test_drop_failure_still_removes_files_and_preserves_test_failure(self):
        for callback, expected in (('return 0', 1), ('return 23', 23)):
            with self.subTest(callback=callback):
                result = self.run_helper(callback, mode='drop_failure')
                self.assertEqual(result.returncode, expected, result.stderr)
                self.assertIn('Failed to remove test schema: check_', result.stderr)
                self.assertEqual(list(self.run_dir.iterdir()), [])

    def test_interrupt_stops_callback_children_and_cleans_up(self):
        for signum, expected in ((signal.SIGINT, 130), (signal.SIGTERM, 143)):
            with self.subTest(signal=signum):
                for name in ('events', 'server_ready', 'helper_pid', 'child_pid'):
                    (self.root / name).unlink(missing_ok=True)
                script = '''
set -euo pipefail
source "$1"
run_tests() {
    sleep 120 &
    echo "$!" > "$TEST_STATE/child_pid"
    wait
}
with_test_server check run_tests
'''
                process = subprocess.Popen(
                    ['bash', '-c', script, 'test', str(HELPER)],
                    env=self.env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
                )
                try:
                    deadline = time.monotonic() + 5
                    while not (self.root / 'child_pid').exists():
                        if process.poll() is not None or time.monotonic() >= deadline:
                            self.fail('callback did not start')
                        time.sleep(0.02)
                    os.kill(process.pid, signum)
                    _, stderr = process.communicate(timeout=5)
                    self.assertEqual(process.returncode, expected, stderr)
                    self.assert_cleaned()
                    self.assert_stopped(int((self.root / 'child_pid').read_text()))
                finally:
                    if process.poll() is None:
                        process.kill()
                    process.communicate(timeout=5)


if __name__ == '__main__':
    unittest.main()
