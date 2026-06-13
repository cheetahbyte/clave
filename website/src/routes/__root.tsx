import { createRootRoute, Outlet } from '@tanstack/react-router'
import { lazy } from 'react'
import { Toaster } from 'sonner'

const TanStackRouterDevtools = import.meta.env.DEV
  ? lazy(() =>
      import('@tanstack/react-router-devtools').then((m) => ({
        default: m.TanStackRouterDevtools,
      })),
    )
  : () => null

export const Route = createRootRoute({
  component: () => (
    <>
      <Outlet />
      <Toaster richColors position="bottom-right" />
      <TanStackRouterDevtools />
    </>
  ),
})
