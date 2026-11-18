import ts from 'typescript';
import { readFile, readdir } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';

// Layout and domain composition stay in the app; reusable widgets stay in yukiUI.
const widgets = new Set(['button', 'input', 'select', 'textarea', 'dialog', 'progress', 'audio', 'video', 'img', 'pre', 'table', 'thead', 'tbody', 'tr', 'th', 'td', 'details', 'summary']);
async function visitDirectory(directory) {
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const file = path.join(directory, entry.name);
    if (entry.isDirectory()) { await visitDirectory(file); continue; }
    if (!file.endsWith('.tsx')) continue;
    const source = ts.createSourceFile(file, await readFile(file, 'utf8'), ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
    function visit(node) {
      if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
        assert(!widgets.has(node.tagName.getText(source)), `Use the yukiUI component instead of ${node.tagName.getText(source)} in ${file}`);
      }
      ts.forEachChild(node, visit);
    }
    visit(source);
  }
}
await visitDirectory(fileURLToPath(new URL('../src', import.meta.url)));
console.log('PASS application JSX uses yukiUI for all reusable widgets');
