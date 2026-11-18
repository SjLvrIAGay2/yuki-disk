// Explicit filename classification; parsing/rendering never executes document contents.
const plainExtensions = new Set('txt text log csv tsv nfo asc rst adoc asciidoc srt vtt ass ssa lrc'.split(' '));
const markdownExtensions = new Set('md markdown mdown mkd'.split(' '));
const plainNames = new Set('readme license licence notice copying authors changelog .gitignore .gitattributes .dockerignore'.split(' '));
const codeExtensions = new Set("js jsx mjs cjs ts tsx mts cts html htm xml svg vue svelte xsd xsl xslt css scss less json jsonl ndjson py pyw pyi sh bash zsh sql yaml yml lua luau ini cfg toml env config properties go rs java c h cpp hpp cc cxx hxx cs csx kt kts swift rb rake gemspec php phtml r pl pm scala sc groovy gradle dart ps1 psm1 psd1 bat cmd dockerfile mk makefile cmake diff patch graphql gql proto tex sty cls htaccess conf cnf gitignore gitattributes dockerignore".split(' '));
const codeNames = new Set("dockerfile containerfile makefile gnumakefile cmakelists.txt .bashrc .bash_profile .zshrc .profile .gitconfig .editorconfig .npmrc .yarnrc nginx.conf .htaccess gemfile rakefile".split(' '));

export function textPreviewFormat(filename: string): 'text' | 'code' | 'markdown' | undefined {
  const name = filename.split(/[\\/]/).pop()?.toLowerCase() ?? '';
  if (codeNames.has(name) || /^(dockerfile|containerfile)\..+$/.test(name) || name === '.env' || name.startsWith('.env.')) return 'code';
  if (plainNames.has(name)) return 'text';
  const extension = name.split('.').pop() ?? '';
  if (markdownExtensions.has(extension)) return 'markdown';
  if (codeExtensions.has(extension)) return 'code';
  if (plainExtensions.has(extension)) return 'text';
  return undefined;
}
