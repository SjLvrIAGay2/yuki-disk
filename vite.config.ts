import { defineConfig } from 'vite';

// The server canonicalizes mount roots with a trailing slash. One immutable
// artifact can then resolve assets beneath each configured public path.
export default defineConfig({ base: './' });
