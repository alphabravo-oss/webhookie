import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/pagerduty/$channelId')({
  component: () => null,
});
