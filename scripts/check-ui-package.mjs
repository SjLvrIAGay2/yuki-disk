import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFile, readdir, lstat } from 'node:fs/promises';

const installed = new URL('../node_modules/@yuki-ui/react/', import.meta.url);
const source = new URL('../../../yukiUI/', import.meta.url);
const readJSON = async (url) => JSON.parse(await readFile(url, 'utf8'));
const expected = await readJSON(new URL('package.json', source));
const actual = await readJSON(new URL('package.json', installed));
assert.equal(actual.name, '@yuki-ui/react');
assert.equal(actual.version, expected.version, 'Run npm ci after updating yukiUI');
assert.deepEqual(actual.exports, expected.exports);
assert.deepEqual(actual.dependencies, expected.dependencies, 'Run npm install after changing yukiUI dependencies');
assert.equal((await lstat(installed)).isSymbolicLink(), false, 'Use install-links=true');
const files = async (base) => (await readdir(new URL('dist/', base), { recursive: true, withFileTypes: true }))
  .filter((entry) => entry.isFile());
const sourceFiles = await files(source);
assert.ok(sourceFiles.length > 0, 'Build yukiUI before installing web dependencies');
assert.equal((await files(installed)).length, sourceFiles.length);
const { relative, join } = await import('node:path');
const { fileURLToPath } = await import('node:url');
for (const entry of sourceFiles) {
  const name = relative(fileURLToPath(source), join(entry.parentPath, entry.name)).replaceAll('\\', '/');
  const hash = async (base) => createHash('sha256').update(await readFile(new URL(name, base))).digest('hex');
  assert.equal(await hash(installed), await hash(source), `Stale UI package: ${name}; rebuild yukiUI and run npm ci`);
}
console.log(`PASS @yuki-ui/react ${actual.version}: ${sourceFiles.length} compiled files match sibling repository; package is not a symlink`);
