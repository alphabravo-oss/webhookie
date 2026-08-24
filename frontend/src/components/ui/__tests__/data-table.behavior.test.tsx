import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { DataTable, type Column } from '@/components/ui/data-table';

type Row = { id: string; name: string; size: number };

const rows: Row[] = [
  { id: 'b', name: 'Banana', size: 30 },
  { id: 'a', name: 'Apple', size: 10 },
  { id: 'c', name: 'Cherry', size: 20 },
];

const columns: Column<Row>[] = [
  { key: 'name', header: 'Name', accessor: (r) => r.name },
  { key: 'size', header: 'Size', accessor: (r) => String(r.size), sortAccessor: (r) => r.size },
];

// Body rows are every table row except the header row.
const bodyRowText = () => screen.getAllByRole('row').slice(1).map((r) => r.textContent ?? '');

describe('DataTable behavior (TanStack Table engine)', () => {
  it('keeps source order until a column is sorted', () => {
    render(<DataTable data={rows} columns={columns} keyExtractor={(r) => r.id} />);
    expect(bodyRowText().map((t) => t.replace(/[0-9]/g, ''))).toEqual(['Banana', 'Apple', 'Cherry']);
  });

  it('sorts numerically via sortAccessor, ascending then descending', () => {
    render(<DataTable data={rows} columns={columns} keyExtractor={(r) => r.id} />);
    const sizeHeader = screen.getByRole('columnheader', { name: /size/i });

    fireEvent.click(sizeHeader); // asc: 10, 20, 30
    expect(bodyRowText()[0]).toContain('Apple');

    fireEvent.click(sizeHeader); // desc: 30, 20, 10
    expect(bodyRowText()[0]).toContain('Banana');
  });

  it('filters rows with the global search box across visible columns (200ms debounce)', async () => {
    render(<DataTable data={rows} columns={columns} keyExtractor={(r) => r.id} />);
    fireEvent.change(screen.getByPlaceholderText('Search...'), { target: { value: 'Apple' } });
    // Filter application is debounced (200ms): nothing filtered immediately.
    expect(bodyRowText()).toHaveLength(3);
    await waitFor(() => expect(bodyRowText()).toHaveLength(1));
    expect(bodyRowText()[0]).toContain('Apple');
  });

  it('paginates and navigates between pages', () => {
    render(<DataTable data={rows} columns={columns} keyExtractor={(r) => r.id} pageSize={1} />);
    expect(screen.getByText('Showing 1-1 of 3')).toBeInTheDocument();
    expect(bodyRowText()).toHaveLength(1);
    expect(bodyRowText()[0]).toContain('Banana');

    fireEvent.click(screen.getByRole('button', { name: '2' }));
    expect(bodyRowText()[0]).toContain('Apple');
  });

  it('persists column visibility to localStorage when persistKey is set', () => {
    window.localStorage.clear();
    const { unmount } = render(
      <DataTable data={rows} columns={columns} keyExtractor={(r) => r.id} persistKey="test-table" />
    );
    fireEvent.click(screen.getByRole('button', { name: /columns/i }));
    fireEvent.click(screen.getAllByRole('checkbox')[1]); // hide Size

    expect(JSON.parse(window.localStorage.getItem('dt:test-table:visibility')!)).toMatchObject({
      size: false,
    });

    // Remount: the hidden column is restored from storage.
    unmount();
    render(
      <DataTable data={rows} columns={columns} keyExtractor={(r) => r.id} persistKey="test-table" />
    );
    expect(screen.queryByRole('columnheader', { name: /size/i })).not.toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /name/i })).toBeInTheDocument();
    window.localStorage.clear();
  });

  it('filters rows via a faceted multi-select column filter', () => {
    type R = { id: string; name: string; status: string };
    const facetRows: R[] = [
      { id: '1', name: 'Alpha', status: 'ready' },
      { id: '2', name: 'Bravo', status: 'pending' },
      { id: '3', name: 'Charlie', status: 'ready' },
    ];
    const facetColumns: Column<R>[] = [
      { key: 'name', header: 'Name', accessor: (r) => r.name },
      {
        key: 'status',
        header: 'Status',
        accessor: (r) => r.status,
        sortAccessor: (r) => r.status,
        filter: { label: 'Status' },
      },
    ];
    render(
      <DataTable data={facetRows} columns={facetColumns} keyExtractor={(r) => r.id} searchable={false} />
    );

    // Open the Status facet and select 'ready'.
    fireEvent.click(screen.getByRole('button', { name: /status/i }));
    fireEvent.click(screen.getByRole('checkbox', { name: 'ready' }));

    const body = bodyRowText();
    expect(body).toHaveLength(2);
    expect(body.join(' ')).toContain('Alpha');
    expect(body.join(' ')).toContain('Charlie');
    expect(body.join(' ')).not.toContain('Bravo');
  });

  it('server-side mode uses rowCount for paging and reports page changes without client slicing', () => {
    const onPaginationChange = vi.fn();
    const pageRows: Row[] = [
      { id: '1', name: 'A', size: 1 },
      { id: '2', name: 'B', size: 2 },
    ];
    render(
      <DataTable
        data={pageRows}
        columns={columns}
        keyExtractor={(r) => r.id}
        pageSize={2}
        searchable={false}
        serverSide={{
          rowCount: 10,
          pagination: { pageIndex: 0, pageSize: 2 },
          onPaginationChange,
        }}
      />
    );

    // Footer reflects the SERVER total (10), not the 2 loaded rows.
    expect(screen.getByText('Showing 1-2 of 10')).toBeInTheDocument();
    // The 2 loaded rows are shown as-is (no further client-side slicing).
    expect(bodyRowText()).toHaveLength(2);

    // Navigating hands the new page index back to the caller.
    fireEvent.click(screen.getByRole('button', { name: '2' }));
    expect(onPaginationChange).toHaveBeenCalledWith(expect.objectContaining({ pageIndex: 1 }));
  });

  it('renders drag resize handles only when resizable is enabled', () => {
    const { container, rerender } = render(
      <DataTable data={rows} columns={columns} keyExtractor={(r) => r.id} />
    );
    // Default (non-resizable) path: no resize handles in the DOM.
    expect(container.querySelectorAll('[data-resize-handle]')).toHaveLength(0);

    rerender(<DataTable data={rows} columns={columns} keyExtractor={(r) => r.id} resizable />);
    // One handle per (resizable) column header.
    expect(container.querySelectorAll('[data-resize-handle]')).toHaveLength(columns.length);
  });

  it('persists column sizing to localStorage when resizable + persistKey are set', () => {
    window.localStorage.clear();
    const { container } = render(
      <DataTable
        data={rows}
        columns={columns}
        keyExtractor={(r) => r.id}
        resizable
        persistKey="resize-table"
      />
    );

    const handle = container.querySelector('[data-resize-handle]')!;
    // Simulate a drag: press on the handle, move, release.
    fireEvent.mouseDown(handle, { clientX: 0 });
    fireEvent.mouseMove(document, { clientX: 40 });
    fireEvent.mouseUp(document);

    const stored = window.localStorage.getItem('dt:resize-table:sizing');
    expect(stored).not.toBeNull();
    expect(typeof JSON.parse(stored!)).toBe('object');
    window.localStorage.clear();
  });

  describe('virtualized mode', () => {
    // jsdom has no real layout, so the scroll container reports height 0 and
    // the virtualizer would otherwise render nothing. Stub getBoundingClientRect
    // / clientHeight so the virtualizer believes the viewport is ~400px tall and
    // windows a small, bounded slice of the rows.
    const stubLayout = () => {
      // @tanstack/react-virtual observes the scroll element with ResizeObserver,
      // which jsdom lacks — without it the virtualizer measures a 0px viewport
      // and renders nothing. Provide a minimal stub that fires the callback once
      // so the virtualizer reads our stubbed getBoundingClientRect height.
      const prevRO = (globalThis as Record<string, unknown>).ResizeObserver;
      (globalThis as Record<string, unknown>).ResizeObserver = class {
        cb: ResizeObserverCallback;
        constructor(cb: ResizeObserverCallback) {
          this.cb = cb;
        }
        observe(el: Element) {
          this.cb([{ target: el } as ResizeObserverEntry], this as unknown as ResizeObserver);
        }
        unobserve() {}
        disconnect() {}
      };
      // virtual-core measures the scroll element via offsetWidth/offsetHeight
      // (see getRect in @tanstack/virtual-core), so stub those: a 400px-tall
      // grid viewport windows ~8 rows + overscan out of the 1000.
      const props = ['offsetHeight', 'offsetWidth'] as const;
      const descriptors = props.map(
        (p) => [p, Object.getOwnPropertyDescriptor(HTMLElement.prototype, p)] as const
      );
      Object.defineProperty(HTMLElement.prototype, 'offsetHeight', {
        configurable: true,
        get() {
          return this.getAttribute('role') === 'grid' ? 400 : 52;
        },
      });
      Object.defineProperty(HTMLElement.prototype, 'offsetWidth', {
        configurable: true,
        get() {
          return 800;
        },
      });
      return () => {
        for (const [p, d] of descriptors) {
          if (d) Object.defineProperty(HTMLElement.prototype, p, d);
          else delete (HTMLElement.prototype as unknown as Record<string, unknown>)[p];
        }
        (globalThis as Record<string, unknown>).ResizeObserver = prevRO;
      };
    };

    const bigRows: Row[] = Array.from({ length: 1000 }, (_, i) => ({
      id: String(i),
      name: `Row ${i}`,
      size: i,
    }));

    it('renders a role="grid" container with aria-rowcount and only a windowed subset of rows', () => {
      const restore = stubLayout();
      try {
        render(
          <DataTable data={bigRows} columns={columns} keyExtractor={(r) => r.id} virtualized />
        );

        const grid = screen.getByRole('grid');
        expect(grid).toBeInTheDocument();
        expect(grid).toHaveAttribute('aria-rowcount', '1000');

        // Body rows = all role="row" minus the sticky header row. The virtualizer
        // must mount far fewer than the full 1000 rows.
        const bodyRows = screen.getAllByRole('row').slice(1);
        expect(bodyRows.length).toBeGreaterThan(0);
        expect(bodyRows.length).toBeLessThan(1000);
        expect(bodyRows.length).toBeLessThan(100);
      } finally {
        restore();
      }
    });

    it('hides the pagination footer in virtualized mode', () => {
      const restore = stubLayout();
      try {
        render(
          <DataTable
            data={bigRows}
            columns={columns}
            keyExtractor={(r) => r.id}
            virtualized
            pageSize={10}
          />
        );
        // No "Showing X-Y of Z" footer because pagination is disabled.
        expect(screen.queryByText(/Showing/i)).not.toBeInTheDocument();
      } finally {
        restore();
      }
    });

    it('moves row focus with ArrowDown via roving tabindex', async () => {
      const restore = stubLayout();
      try {
        render(
          <DataTable data={bigRows} columns={columns} keyExtractor={(r) => r.id} virtualized />
        );

        // Address rows by their stable data-row-index (the accessible name is
        // ambiguous here — "Row 1" + size cell "1" collides with "Row 11").
        const rowAt = (i: number) =>
          screen.getByRole('grid').querySelector<HTMLElement>(`[data-row-index="${i}"]`)!;

        // The grid container is the single Tab entry point; rows are focusable
        // programmatically only (so a virtualized-out row can't strand keyboard
        // users outside the grid).
        const grid = screen.getByRole('grid');
        expect(grid).toHaveAttribute('tabindex', '0');

        const firstRow = rowAt(0);
        expect(firstRow).toHaveAttribute('tabindex', '-1');

        // ArrowDown on the focused container enters the body at row 0.
        grid.focus();
        fireEvent.keyDown(grid, { key: 'ArrowDown' });
        await waitFor(() => expect(rowAt(0)).toHaveFocus());

        // ArrowDown again moves focus to the next row (deferred to rAF once the
        // target row is mounted).
        fireEvent.keyDown(firstRow, { key: 'ArrowDown' });
        await waitFor(() => expect(rowAt(1)).toHaveFocus());
      } finally {
        restore();
      }
    });

    it('still renders skeleton rows while loading in virtualized mode', () => {
      const restore = stubLayout();
      try {
        render(
          <DataTable
            data={[]}
            columns={columns}
            keyExtractor={(r) => r.id}
            virtualized
            loading
            loadingRows={3}
          />
        );
        expect(screen.getByRole('grid')).toBeInTheDocument();
        // 3 skeleton body rows + 1 header row.
        expect(screen.getAllByRole('row')).toHaveLength(4);
      } finally {
        restore();
      }
    });

    it('renders the empty state in virtualized mode', () => {
      const restore = stubLayout();
      try {
        render(
          <DataTable
            data={[]}
            columns={columns}
            keyExtractor={(r) => r.id}
            virtualized
            emptyMessage="Nothing here"
          />
        );
        expect(screen.getByText('Nothing here')).toBeInTheDocument();
      } finally {
        restore();
      }
    });
  });

  it('toggles column visibility but refuses to hide the last column', () => {
    render(<DataTable data={rows} columns={columns} keyExtractor={(r) => r.id} />);
    fireEvent.click(screen.getByRole('button', { name: /columns/i }));

    const checkboxes = screen.getAllByRole('checkbox'); // [Name, Size] in the dropdown
    fireEvent.click(checkboxes[1]); // hide Size
    expect(screen.queryByRole('columnheader', { name: /size/i })).not.toBeInTheDocument();

    fireEvent.click(checkboxes[0]); // attempt to hide Name (the last visible) → refused
    expect(screen.getByRole('columnheader', { name: /name/i })).toBeInTheDocument();
  });
});
