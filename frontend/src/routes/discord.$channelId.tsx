import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/discord/$channelId')({
  component: () => null,
});
