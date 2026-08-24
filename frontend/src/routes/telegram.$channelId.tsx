import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/telegram/$channelId')({
  component: () => null,
});
