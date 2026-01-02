import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { rename } from 'fs/promises'

// https://vite.dev/config/
export default defineConfig({
    plugins: [
        react(),
        tailwindcss(),
        {
            name: 'rename-html',
            closeBundle: async () => {
                await rename('dist/index.prod.html', 'dist/index.html')
            }
        }
    ],
    build: {
        rollupOptions: {
            input: {
                index: 'index.prod.html'
            }
        }
    },
    server: {
        proxy: {
            '/auth': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
            '/graphql': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
        },
    },
})
