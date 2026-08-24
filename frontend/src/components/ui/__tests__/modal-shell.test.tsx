import { fireEvent, render, screen } from '@testing-library/react';
import { ModalShell } from '@/components/ui/modal-shell';

describe('ModalShell', () => {
  it('renders title and body content', () => {
    render(
      <ModalShell title="Security action" onClose={vi.fn()}>
        <p>Confirm the sensitive action.</p>
      </ModalShell>,
    );

    expect(screen.getByRole('dialog', { name: 'Security action' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Security action' })).toBeInTheDocument();
    expect(screen.getByText('Confirm the sensitive action.')).toBeInTheDocument();
  });

  it('closes on Escape', () => {
    const onClose = vi.fn();
    render(
      <ModalShell title="Security action" onClose={onClose}>
        <p>Confirm the sensitive action.</p>
      </ModalShell>,
    );

    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('moves focus into the dialog and restores prior focus on unmount', () => {
    const opener = document.createElement('button');
    opener.textContent = 'Open modal';
    document.body.appendChild(opener);
    opener.focus();

    const { unmount } = render(
      <ModalShell title="Security action" onClose={vi.fn()}>
        <button type="button">Confirm action</button>
      </ModalShell>,
    );

    expect(screen.getByLabelText('Close')).toHaveFocus();

    unmount();
    expect(opener).toHaveFocus();
    opener.remove();
  });

  it('traps Tab focus inside the dialog', () => {
    render(
      <ModalShell
        title="Security action"
        onClose={vi.fn()}
        footer={
          <>
            <button type="button">Cancel</button>
            <button type="button">Submit</button>
          </>
        }
      >
        <p>Confirm the sensitive action.</p>
      </ModalShell>,
    );

    const close = screen.getByLabelText('Close');
    const submit = screen.getByRole('button', { name: 'Submit' });

    expect(close).toHaveFocus();

    fireEvent.keyDown(document, { key: 'Tab', shiftKey: true });
    expect(submit).toHaveFocus();

    fireEvent.keyDown(document, { key: 'Tab' });
    expect(close).toHaveFocus();
  });
});
