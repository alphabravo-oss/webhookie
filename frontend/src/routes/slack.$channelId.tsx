import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/slack/$channelId')({
  component: () => null,
});
