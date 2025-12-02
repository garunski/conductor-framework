import { defineConfig } from 'vite';
import vituum from 'vituum';
import posthtml from '@vituum/vite-plugin-posthtml';

export default defineConfig({
  plugins: [vituum(), posthtml({
    root: './src'
  })],
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
});

