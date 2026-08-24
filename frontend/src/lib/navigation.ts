import { useMemo } from 'react';
import {
  useLocation,
  useNavigate,
  useParams as useRouteParams,
  useRouter as useTanstackRouter,
  type NavigateOptions,
} from '@tanstack/react-router';
import { hrefToLocation } from '@/lib/link';

// Runtime-string hrefs cannot satisfy the registered-route literal union; one
// deliberate widening cast keeps the wrapper's `href: string` API (D2).
function locationOptions(href: string): NavigateOptions {
  return hrefToLocation(href) as unknown as NavigateOptions;
}

/**
 * Drop-in replacement for the old `next/navigation` `useRouter` re-export.
 * `replace` always navigates with `resetScroll: false` to preserve
 * `useTabParam`'s no-jump contract (`?tab=` switches must not scroll to top);
 * push navigations keep the router-level `scrollRestoration` reset-to-top.
 */
export function useRouter() {
  const navigate = useNavigate();
  const { history } = useTanstackRouter();
  return useMemo(
    () => ({
      push: (href: string) => {
        void navigate(locationOptions(href));
      },
      replace: (href: string, _options?: { scroll?: boolean }) => {
        void navigate({ ...locationOptions(href), replace: true, resetScroll: false });
      },
      back: () => history.back(),
    }),
    [navigate, history],
  );
}

export function usePathname(): string {
  return useLocation({ select: (location) => location.pathname });
}

/** Real `URLSearchParams` built from the current location's raw search string. */
export function useSearchParams(): URLSearchParams {
  const searchStr = useLocation({ select: (location) => location.searchStr });
  return useMemo(() => new URLSearchParams(searchStr), [searchStr]);
}

export function useParams<
  T extends Record<string, string | string[]> = Record<string, string>,
>(): T {
  // Two statements on purpose: `return useRouteParams(...) as T` would give
  // the call a contextual type of T and derail TanStack's TSelected inference.
  const params = useRouteParams({ strict: false });
  return params as T;
}
