import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/opsgenie/$channelId')({
  component: () => null,
});
