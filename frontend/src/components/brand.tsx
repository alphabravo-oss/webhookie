import { cn } from '@/lib/utils';

/** Lucide fishing-hook on the Astronomer gradient mark. */
export function LogoMark({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        'inline-flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-blue-500 to-violet-600 text-white',
        className,
      )}
      aria-hidden
    >
      <FishingHookIcon className="h-4 w-4" />
    </span>
  );
}

export function FishingHookIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <path d="m17.586 11.414-5.93 5.93a1 1 0 0 1-8-8l3.137-3.137a.707.707 0 0 1 1.207.5V10" />
      <path d="M20.414 8.586 22 7" />
      <circle cx="19" cy="10" r="2" />
    </svg>
  );
}
