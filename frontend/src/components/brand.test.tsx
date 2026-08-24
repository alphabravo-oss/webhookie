import { render } from '@testing-library/react';
import { LogoMark } from './brand';

describe('LogoMark', () => {
  it('renders the fishing-hook paths', () => {
    const { container } = render(<LogoMark />);
    const paths = container.querySelectorAll('path');
    expect(paths.length).toBe(2);
    expect(paths[0].getAttribute('d')).toContain('17.586');
    expect(container.querySelector('circle')).toHaveAttribute('cx', '19');
  });
});
