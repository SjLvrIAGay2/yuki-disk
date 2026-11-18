import { defineConfig } from 'vite';

// Relative asset base keeps the built page servable from a path prefix
// as well as from the origin root.
export default defineConfig({
  base: './',
  build: { outDir: 'dist', emptyOutDir: true },
});
