import { forwardRef, type ComponentPropsWithoutRef } from 'react';
import { Link as RouterLink } from '@tanstack/react-router';

export interface LinkLocation {
  to: string;
  search: Record<string, string>;
  hash?: string;
}

function splitOnce(value: string, separator: string): [string, string | undefined] {
  const index = value.indexOf(separator);
  if (index === -1) return [value, undefined];
  return [value.slice(0, index), value.slice(index + 1)];
}

/**
 * Parse a string href into a TanStack Router location. TanStack's `to` does
 * not parse query strings (audit/search pages build hrefs with them), so the
 * href is split on `#` then `?` and the query handed over as a search object.
 * `search` is always present (possibly empty) so navigation fully specifies
 * the destination URL, matching Next.js string-href semantics.
 */
export function hrefToLocation(href: string): LinkLocation {
  const [withoutHash, hash] = splitOnce(href, '#');
  const [to, query] = splitOnce(withoutHash, '?');
  const search: Record<string, string> = {};
  for (const [key, value] of new URLSearchParams(query ?? '')) {
    search[key] = value;
  }
  return hash ? { to, search, hash } : { to, search };
}

export interface LinkProps extends Omit<ComponentPropsWithoutRef<'a'>, 'href'> {
  href: string;
}

/**
 * Drop-in replacement for the old `next/link` re-export: accepts a string
 * `href` (string-only is safe — D2: zero UrlObject consumers) plus anchor
 * props, and renders a TanStack Router `<Link>` with the parsed location.
 * Runtime-string `to` cannot satisfy the registered-route literal union, so
 * one deliberate widening cast keeps every consumer's `href: string` API.
 */
export const Link = forwardRef<HTMLAnchorElement, LinkProps>(function Link(
  { href, ...rest },
  ref,
) {
  const location = hrefToLocation(href) as unknown as ComponentPropsWithoutRef<
    typeof RouterLink
  >;
  return <RouterLink ref={ref} {...rest} {...location} />;
});
