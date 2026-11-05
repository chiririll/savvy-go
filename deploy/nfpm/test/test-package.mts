// Installs a built deb/rpm/apk into a systemd (apk: OpenRC) container and checks that the service
// works across install, crash, upgrade and removal.
// Usage: node test-package.mts <deb|rpm|apk> [path/to/package]   (default: dist/savvy-go.<format>)
import { spawnSync, type SpawnSyncOptionsWithStringEncoding, type SpawnSyncReturns } from 'node:child_process';
import { existsSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const format = process.argv[2] ?? '';
const appUser = ({ deb: 'www-data', rpm: 'savvy-go', apk: 'savvy-go' } as Record<string, string>)[format];
if (!appUser) {
    console.error('usage: test-package.mts <deb|rpm|apk> [package]');
    process.exit(1);
}

const here = dirname(fileURLToPath(import.meta.url));
const root = resolve(here, '..', '..', '..');
const pkg = resolve(process.argv[3] ?? join(root, 'dist', `savvy-go.${format}`));
if (!existsSync(pkg)) {
    console.error(`missing package: ${pkg}`);
    process.exit(1);
}

const image = `savvy-go-pkgtest-${format}`;
const name = `${image}-${process.pid}`;
const unit = 'savvy-go.service';
const openrc = format === 'apk'; // Alpine has OpenRC instead of systemd
const baseUrl = 'http://127.0.0.1:8080';
const dataDir = '/var/lib/savvy-go';
const config = '/etc/savvy-go/config.toml';

const docker = (args: string[], opts: Partial<SpawnSyncOptionsWithStringEncoding> = {}) =>
    spawnSync('docker', args, { encoding: 'utf8', ...opts });
const dexec = (...cmd: string[]) => docker(['exec', name, ...cmd]);
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

class Failure extends Error {}
const fail = (msg: string): never => {
    throw new Failure(msg);
};
const step = (msg: string) => console.log(`\n== ${msg}`);

const check = (desc: string, ...cmd: string[]) => {
    if (dexec(...cmd).status !== 0) fail(desc);
    console.log(`ok: ${desc}`);
};
const checkNot = (desc: string, ...cmd: string[]) => {
    if (dexec(...cmd).status === 0) fail(desc);
    console.log(`ok: ${desc}`);
};
const waitFor = async (desc: string, ...cmd: string[]) => {
    for (let i = 0; i < 30; i++) {
        if (dexec(...cmd).status === 0) {
            console.log(`ok: ${desc}`);
            return;
        }
        await sleep(1000);
    }
    fail(`${desc} (timed out)`);
};
const must = (result: Pick<SpawnSyncReturns<string>, 'status' | 'stdout' | 'stderr'>, desc: string) => {
    if (result.status !== 0) fail(`${desc}\n${result.stdout}${result.stderr}`);
};
const mainPid = () =>
    (openrc
        ? dexec('pgrep', '-f', '^/usr/bin/savvy-go').stdout.split('\n')[0]
        : dexec('systemctl', 'show', '-p', 'MainPID', '--value', unit).stdout
    ).trim();
// Shell snippet that exits 0 when the service has a running main process.
const isActive = openrc ? ['rc-service', 'savvy-go', 'status'] : ['systemctl', 'is-active', unit];
const isEnabled = openrc
    ? ['sh', '-c', 'rc-update show default | grep -q savvy-go']
    : ['systemctl', 'is-enabled', unit];
const restart = () => (openrc ? dexec('rc-service', 'savvy-go', 'restart') : dexec('systemctl', 'restart', unit));
const unitFile = openrc ? '/etc/init.d/savvy-go' : `/usr/lib/systemd/system/${unit}`;

async function run() {
    must(docker(['build', '-q', '-t', image, '-f', join(here, `Dockerfile.${format}`), here]), 'image build failed');
    // systemd needs to own the container's cgroup; --privileged is the portable way
    // on CI runners and Docker Desktop.
    must(
        docker(
            openrc
                ? ['run', '-d', '--init', '--name', name, image] // --init reaps the supervisor's children, like a real init
                : [
                      'run', '-d', '--name', name, '--privileged', '--cgroupns=host',
                      '-v', '/sys/fs/cgroup:/sys/fs/cgroup:rw', '--tmpfs', '/run', '--tmpfs', '/run/lock', image,
                  ],
        ),
        'container start failed',
    );

    if (!openrc) {
        step('boot systemd');
        // "degraded" (some unit that makes no sense in a container failed) is fine. Waiting for the end of
        // the boot matters: it cleans or mounts /tmp, which would swallow the package copied before it.
        await waitFor(
            'systemd finished booting',
            'sh', '-c', 'case "$(systemctl is-system-running 2>/dev/null)" in running|degraded) exit 0;; *) exit 1;; esac',
        );
    }
    must(docker(['cp', pkg, `${name}:/tmp/pkg.${format}`]), 'docker cp failed');

    const install = (reinstall: boolean) => {
        if (openrc) {
            // The package is unsigned here; the repository signs the index, not the files.
            return dexec('apk', 'add', '--allow-untrusted', ...(reinstall ? ['--force-overwrite'] : []), '/tmp/pkg.apk');
        }
        if (format === 'deb') {
            must(dexec('apt-get', 'update', '-qq'), 'apt-get update failed');
            return dexec('apt-get', 'install', '-y', ...(reinstall ? ['--reinstall'] : []), '/tmp/pkg.deb');
        }
        if (reinstall) {
            const r = dexec('dnf', 'reinstall', '-y', '/tmp/pkg.rpm');
            if (r.status === 0) return r;
        }
        return dexec('dnf', 'install', '-y', '/tmp/pkg.rpm');
    };
    const remove = (purge = false) =>
        openrc
            ? dexec('apk', 'del', 'savvy-go')
            : format === 'deb'
              ? dexec('apt-get', purge ? 'purge' : 'remove', '-y', 'savvy-go')
              : dexec('dnf', 'remove', '-y', 'savvy-go');

    step('fresh install');
    must(install(false), 'install failed');
    check(openrc ? 'init script installed' : 'unit file installed', 'test', '-f', unitFile);
    check('binary installed', 'test', '-x', '/usr/bin/savvy-go');
    if (openrc) {
        // Alpine packages do not start or enable services themselves.
        checkNot('service not started by the package', ...isActive);
        must(dexec('rc-update', 'add', 'savvy-go', 'default'), 'rc-update add failed');
        must(dexec('rc-service', 'savvy-go', 'start'), 'rc-service start failed');
    }
    check('service enabled', ...isEnabled);
    await waitFor('service active', ...isActive);
    await waitFor('/livez answers', 'curl', '-fsS', `${baseUrl}/livez`);
    await waitFor('/readyz answers', 'curl', '-fsS', `${baseUrl}/readyz`);
    // The UI is built into the binary: the page must point at a Vite asset that is served too.
    check(
        'embedded SPA and its assets served',
        'sh', '-c', `a=$(curl -fsS ${baseUrl}/ | grep -o '/build/assets/[^"]*' | head -n1) && [ -n "$a" ] && curl -fsS -o /dev/null ${baseUrl}$a`,
    );
    check('config created', 'test', '-f', config);
    check('data dir owned by app user', 'sh', '-c', `[ "$(stat -c %U ${dataDir})" = ${appUser} ]`);
    check('process runs as app user', 'sh', '-c', `[ "$(stat -c %U /proc/${mainPid()})" = ${appUser} ]`);

    step('crash recovery (Restart=always)');
    const oldPid = mainPid();
    dexec('kill', '-9', oldPid);
    await waitFor(
        'service restarted with a new pid',
        'sh', '-c', openrc
            ? `p=$(pgrep -f ^/usr/bin/savvy-go | head -n1); [ -n "$p" ] && [ "$p" != ${oldPid} ]`
            : `p=$(systemctl show -p MainPID --value ${unit}); [ "$p" != 0 ] && [ "$p" != ${oldPid} ]`,
    );
    await waitFor('/livez answers after crash', 'curl', '-fsS', `${baseUrl}/livez`);

    step('manual restart keeps state');
    restart();
    await waitFor('/readyz answers after restart', 'curl', '-fsS', `${baseUrl}/readyz`);
    check('database kept', 'sh', '-c', `ls ${dataDir}/*.sqlite`);

    step('upgrade (reinstall same version)');
    must(install(true), 'reinstall failed');
    check('service still enabled', ...isEnabled);
    await waitFor('service active after upgrade', ...isActive);
    await waitFor('/readyz answers after upgrade', 'curl', '-fsS', `${baseUrl}/readyz`);

    step('removal');
    must(remove(), 'remove failed');
    checkNot('service stopped', ...isActive);
    checkNot(openrc ? 'init script removed' : 'unit file removed', 'test', '-e', unitFile);
    checkNot('binary removed', 'test', '-e', '/usr/bin/savvy-go');
    check('data kept on remove', 'test', '-d', dataDir);
    check('config kept on remove', 'test', '-f', config);

    if (format === 'deb') {
        step('purge');
        must(install(false), 'reinstall failed');
        await waitFor('service active after reinstall', 'systemctl', 'is-active', unit);
        must(remove(true), 'purge failed');
        checkNot('data removed on purge', 'test', '-e', dataDir);
        checkNot('config removed on purge', 'test', '-e', '/etc/savvy-go');
    }

    console.log(`\nAll ${format} package checks passed.`);
}

let code = 0;
try {
    await run();
} catch (err) {
    code = 1;
    console.error(`\nFAIL: ${err instanceof Failure ? err.message : (err as Error).stack}`);
    const logs = openrc ? dexec('tail', '-n', '50', '/var/log/savvy-go.log') : dexec('journalctl', '-u', unit, '--no-pager', '-n', '50');
    console.error(`--- ${openrc ? 'log' : `journal (${unit})`} ---\n${logs.stdout ?? ''}`);
} finally {
    docker(['rm', '-f', name]);
}
process.exit(code);
