import { useEffect } from 'react';
import { createRouter, type ErrorComponentProps } from '@tanstack/react-router';
import { routeTree } from './routeTree.gen';

function RootErrorPanel({ error, reset }: ErrorComponentProps) {
  useEffect(() => {
    console.error('Root route error:', error);
  }, [error]);

  return (
    <div
      data-testid="route-error-boundary"
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#0b0b0f',
        color: '#e5e5e5',
        fontFamily: 'system-ui, -apple-system, Segoe UI, Roboto, sans-serif',
      }}
    >
      <div style={{ maxWidth: 420, padding: 24, textAlign: 'center' }}>
        <h1 style={{ fontSize: 20, fontWeight: 600, margin: '0 0 8px' }}>Something went wrong</h1>
        <p style={{ fontSize: 14, color: '#9ca3af', margin: '0 0 20px' }}>
          {error.message || 'The application failed to load.'}
        </p>
        <button
          type="button"
          onClick={reset}
          style={{
            height: 36,
            padding: '0 16px',
            borderRadius: 8,
            border: 'none',
            background: '#3b82f6',
            color: '#fff',
            fontSize: 14,
            fontWeight: 500,
            cursor: 'pointer',
          }}
        >
          Try again
        </button>
      </div>
    </div>
  );
}

export const router = createRouter({
  routeTree,
  defaultPreload: false,
  scrollRestoration: true,
  defaultErrorComponent: RootErrorPanel,
});

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
