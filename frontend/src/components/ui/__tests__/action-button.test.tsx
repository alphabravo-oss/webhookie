import { fireEvent, render, screen } from '@testing-library/react';
import { Trash2 } from 'lucide-react';
import { ActionButton } from '@/components/ui/action-button';

describe('ActionButton', () => {
  it('renders an enabled action and invokes clicks', () => {
    const onClick = vi.fn();

    render(<ActionButton onClick={onClick}>Run action</ActionButton>);
    fireEvent.click(screen.getByRole('button', { name: 'Run action' }));

    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('blocks clicks while loading and shows loading copy', () => {
    const onClick = vi.fn();

    render(
      <ActionButton loading loadingLabel="Deleting" onClick={onClick}>
        Delete
      </ActionButton>,
    );

    const button = screen.getByRole('button', { name: /deleting/i });
    expect(button).toBeDisabled();
    fireEvent.click(button);
    expect(onClick).not.toHaveBeenCalled();
  });

  it('uses disabled reasons as button titles', () => {
    render(
      <ActionButton disabled disabledReason="requires clusters:update">
        Apply
      </ActionButton>,
    );

    expect(screen.getByRole('button', { name: 'Apply' })).toHaveAttribute('title', 'requires clusters:update');
  });

  it('supports icon-only destructive actions', () => {
    render(
      <ActionButton intent="destructive" size="icon" icon={<Trash2 className="h-4 w-4" />} aria-label="Delete" />,
    );

    expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument();
  });
});
