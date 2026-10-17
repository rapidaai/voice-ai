import path from 'node:path';
import { fileURLToPath } from 'node:url';
import react from '@vitejs/plugin-react';
import browserslistToEsbuild from 'browserslist-to-esbuild';
import { defineConfig } from 'vite';
import { getClientProcessEnv } from './vite.env';

const rootDirectory = path.dirname(
  fileURLToPath(import.meta.url),
);

export default defineConfig(({ command, mode }) => {
  const processEnv = getClientProcessEnv(
    mode,
    command === 'build' ? 'production' : 'development',
  );

  return {
    base: '/',
    plugins: [react()],
    define: {
      'process.env': JSON.stringify(processEnv),
      'process.env.NODE_ENV': JSON.stringify(processEnv.NODE_ENV),
    },
    resolve: {
      alias: {
        '@': path.resolve(rootDirectory, 'src'),
        '@rapidaai/react': path.resolve(
          rootDirectory,
          'node_modules/@rapidaai/react/dist/index.js',
        ),
      },
    },
    server: {
      host: true,
      port: 3000,
      strictPort: true,
    },
    build: {
      outDir: 'build',
      manifest: true,
      sourcemap: false,
      target: browserslistToEsbuild(undefined, { env: 'production' }),
      rollupOptions: {
        output: {
          entryFileNames: 'static/js/[name]-[hash].js',
          chunkFileNames: 'static/js/[name]-[hash].js',
          assetFileNames: 'static/[ext]/[name]-[hash].[ext]',
        },
      },
    },
    worker: {
      format: 'es',
      rollupOptions: {
        output: {
          entryFileNames: 'static/js/[name]-[hash].worker.js',
        },
      },
    },
  };
});
