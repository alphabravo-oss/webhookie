import { fireEvent, render, screen } from '@testing-library/react';
import { TeamsPreview } from './preview-teams';

describe('TeamsPreview', () => {
  it('renders MessageCard actions', () => {
    const onAction = vi.fn();
    render(
      <TeamsPreview
        onAction={onAction}
        body={JSON.stringify({
          '@type': 'MessageCard',
          title: 'Deploy',
          text: 'Ship it?',
          potentialAction: [{ '@type': 'HttpPOST', name: 'Approve' }],
        })}
      />,
    );
    expect(screen.getByText('Deploy')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Approve' }));
    expect(onAction).toHaveBeenCalledWith({ actionId: 'Approve', value: 'Approve', text: 'Approve' });
  });

  it('replaces actions with a submitted receipt', () => {
    render(
      <TeamsPreview
        taken={{ actionId: 'Approve', text: 'Approve' }}
        body={JSON.stringify({
          '@type': 'MessageCard',
          title: 'Deploy',
          text: 'Ship it?',
          potentialAction: [{ '@type': 'HttpPOST', name: 'Approve' }],
        })}
      />,
    );
    expect(screen.queryByRole('button', { name: 'Approve' })).not.toBeInTheDocument();
    expect(screen.getByText(/Your response was sent to the app/)).toBeInTheDocument();
  });
});
