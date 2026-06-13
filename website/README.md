# Clave — Website

Admin dashboard and self-service portal for Clave, the self-hosted license server.

Built with React + TypeScript + Vite + shadcn/ui + TanStack Router.

## Development

```bash
pnpm install
pnpm dev        # http://localhost:5173
pnpm typecheck  # TypeScript check
pnpm lint       # ESLint
```

Vite proxies `/api` to the backend on `:8000`. See the repo root README for backend setup.
