'use client';

import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import type { ReactNode } from 'react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import {
  useReactTable,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  getPaginationRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  type ColumnDef,
  type SortingState,
  type RowSelectionState,
  type VisibilityState,
  type ColumnFiltersState,
  type PaginationState,
  type ColumnSizingState,
  type Updater,
  type Column as RtColumn,
  type Table as RtTable,
  type Row as RtRow,
} from '@tanstack/react-table';
import type { Virtualizer } from '@tanstack/react-virtual';
import { useDebouncedValue } from '@tanstack/react-pacer';
import {
  ChevronDown,
  ChevronUp,
  ChevronsUpDown,
  ChevronLeft,
  ChevronRight,
  Search,
  SlidersHorizontal,
  Filter,
  X,
} from 'lucide-react';
import { cn } from '@/lib/utils';

// ============================================================
// Types
// ============================================================

export interface Column<T> {
  key: string;
  header: string;
  accessor: (row: T) => React.ReactNode;
  sortAccessor?: (row: T) => string | number;
  sortable?: boolean;
  filterable?: boolean;
  hidden?: boolean;
  width?: string;
  align?: 'left' | 'center' | 'right';
  /**
   * When set, renders a faceted multi-select filter for this column in the
   * toolbar. The facet options are derived automatically from the column's
   * `sortAccessor` value (so faceted columns should define a `sortAccessor`
   * that returns the scalar to filter on).
   */
  filter?: { label?: string };
}

interface DataTableProps<T> {
  data: T[];
  columns: Column<T>[];
  keyExtractor: (row: T) => string;
  density?: 'compact' | 'comfortable';
  searchable?: boolean;
  searchPlaceholder?: string;
  selectable?: boolean;
  onRowClick?: (row: T) => void;
  onSelectionChange?: (selected: T[]) => void;
  bulkActions?: (selected: T[]) => ReactNode;
  pageSize?: number;
  loadingRows?: number;
  emptyMessage?: string;
  loading?: boolean;
  /**
   * When true, the table renders a single distinct error row (styled unlike the
   * empty state) instead of data/loading/empty. Pass `query.isError` here.
   */
  isError?: boolean;
  /** Message shown in the error row. */
  errorMessage?: string;
  /** Optional retry action; when provided, an inline Retry button is shown. */
  onRetry?: () => void;
  toolbar?: ReactNode;
  className?: string;
  /**
   * When set, the user's column-visibility choices are persisted to
   * localStorage under this key and restored on next mount. Omit for
   * ephemeral (non-persisted) tables.
   */
  persistKey?: string;
  /**
   * Opt into interactive column resizing. When false (the default), the table
   * renders identically to before — fixed widths come from each column's
   * `width`. When true, columns become drag-resizable and their pixel sizes are
   * persisted under `dt:<persistKey>:sizing` if `persistKey` is set.
   */
  resizable?: boolean;
  /**
   * Opt into row virtualization for large datasets. When false (the default),
   * the table renders identically to before using the semantic table/row
   * primitives with client-side pagination. When true, the table renders a
   * DIV-based ARIA grid that windows the *full* filtered+sorted row model
   * (pagination is disabled and the pagination footer is hidden); only the
   * rows in (and near) the viewport are mounted. Search/sort/faceted-filter/
   * selection still apply over the full row model.
   *
   * Not compatible with `serverSide` (virtualization needs the full row model
   * locally) — if both are passed, `serverSide` paging is ignored.
   */
  virtualized?: boolean;
  /**
   * Opt into server-driven pagination. `data` should hold only the current
   * page's rows; the table will not slice further. The caller owns the
   * pagination state and feeds it into its query params so each page is a
   * separate fetch. (Search/sort remain client-side over the loaded page —
   * pass `searchable={false}` if that's misleading for the dataset.)
   */
  serverSide?: {
    rowCount: number;
    pagination: PaginationState;
    onPaginationChange: (next: PaginationState) => void;
  };
}

const visibilityStorageKey = (persistKey: string) => `dt:${persistKey}:visibility`;
const sizingStorageKey = (persistKey: string) => `dt:${persistKey}:sizing`;

function readPersistedVisibility(persistKey: string | undefined): VisibilityState | null {
  if (!persistKey || typeof window === 'undefined') return null;
  try {
    const raw = window.localStorage.getItem(visibilityStorageKey(persistKey));
    return raw ? (JSON.parse(raw) as VisibilityState) : null;
  } catch {
    return null;
  }
}

function readPersistedSizing(persistKey: string | undefined): ColumnSizingState | null {
  if (!persistKey || typeof window === 'undefined') return null;
  try {
    const raw = window.localStorage.getItem(sizingStorageKey(persistKey));
    return raw ? (JSON.parse(raw) as ColumnSizingState) : null;
  } catch {
    return null;
  }
}

// ============================================================
// Internals
// ============================================================

// Sort value for a column: prefer sortAccessor, else the stringified cell
// output (matching the previous hand-rolled behavior, where columns without a
// sortAccessor sorted on `accessor(row)?.toString()`).
function sortValue<T>(col: Column<T>, row: T): string | number {
  if (col.sortAccessor) return col.sortAccessor(row);
  const val = col.accessor(row);
  return val?.toString() ?? '';
}

// ============================================================
// DataTable Component
// ============================================================

export function DataTable<T>({
  data,
  columns,
  keyExtractor,
  density = 'comfortable',
  searchable = true,
  searchPlaceholder = 'Search...',
  selectable = false,
  onRowClick,
  onSelectionChange,
  bulkActions,
  pageSize = 20,
  loadingRows,
  emptyMessage = 'No results found',
  loading = false,
  isError = false,
  errorMessage = 'Failed to load — try again',
  onRetry,
  toolbar,
  className,
  persistKey,
  resizable = false,
  virtualized = false,
  serverSide,
}: DataTableProps<T>) {
  // serverSide pagination is incompatible with virtualization (the virtualizer
  // windows a fully-loaded row model), so it is ignored when virtualized.
  const effectiveServerSide = virtualized ? undefined : serverSide;
  // The search input is controlled by `searchInput` (instant) while the
  // filter the table actually applies is the 200ms-debounced copy — typing
  // doesn't re-filter the row model on every keystroke. One hook here
  // covers every search-enabled DataTable in the app.
  const [searchInput, setSearchInput] = useState('');
  const [globalFilter] = useDebouncedValue(searchInput, { wait: 200 });
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>(() =>
    Object.fromEntries(columns.filter((c) => c.hidden).map((c) => [c.key, false]))
  );
  const [columnSizing, setColumnSizing] = useState<ColumnSizingState>({});

  // Restore persisted visibility after mount. Reading localStorage during the
  // initial render would diverge from the server-rendered markup and trigger a
  // hydration mismatch, so we apply it in an effect instead.
  useEffect(() => {
    const stored = readPersistedVisibility(persistKey);
    if (stored) setColumnVisibility((prev) => ({ ...prev, ...stored }));
  }, [persistKey]);

  // Restore persisted column sizing after mount, using the same post-mount
  // pattern as visibility to avoid a hydration mismatch. Only meaningful when
  // resizing is enabled.
  useEffect(() => {
    if (!resizable) return;
    const stored = readPersistedSizing(persistKey);
    if (stored) setColumnSizing((prev) => ({ ...prev, ...stored }));
  }, [persistKey, resizable]);
  const [showColumnToggle, setShowColumnToggle] = useState(false);

  const cellPadding = density === 'compact' ? 'px-3 py-2' : 'px-4 py-3';
  const selectPadding = density === 'compact' ? 'px-3 py-2' : 'px-3 py-3';
  const skeletonRows = loadingRows ?? Math.min(pageSize, 8);

  // Lookup of original Column by key — used by sorting/global-filter without
  // threading metadata through react-table's typed column meta.
  const colByKey = useMemo(() => new Map(columns.map((c) => [c.key, c])), [columns]);

  const columnDefs = useMemo<ColumnDef<T>[]>(
    () =>
      columns.map((col) => ({
        id: col.key,
        accessorFn: (row: T) => sortValue(col, row),
        enableSorting: col.sortable !== false,
        enableHiding: true,
        enableColumnFilter: !!col.filter,
        // Faceted multi-select: keep the row when nothing is selected, otherwise
        // when its (stringified) value is among the selected facet values.
        filterFn: (row, columnId, value) => {
          const selected = (value as string[]) ?? [];
          return selected.length === 0 || selected.includes(String(row.getValue(columnId)));
        },
        // Numeric sortAccessors sort numerically; everything else by locale.
        // react-table negates this for descending, so we return the ascending
        // comparison — matching the old `aVal - bVal` / localeCompare logic.
        sortingFn: (a, b, columnId) => {
          const av = a.getValue(columnId);
          const bv = b.getValue(columnId);
          if (typeof av === 'number' && typeof bv === 'number') {
            return av === bv ? 0 : av < bv ? -1 : 1;
          }
          return String(av).localeCompare(String(bv));
        },
      })),
    [columns]
  );

  const table = useReactTable<T>({
    data,
    columns: columnDefs,
    getRowId: keyExtractor,
    state: {
      globalFilter,
      sorting,
      columnFilters,
      rowSelection,
      columnVisibility,
      ...(resizable ? { columnSizing } : {}),
      ...(effectiveServerSide ? { pagination: effectiveServerSide.pagination } : {}),
    },
    ...(resizable
      ? {
          enableColumnResizing: true,
          columnResizeMode: 'onChange' as const,
          onColumnSizingChange: (updater: Updater<ColumnSizingState>) =>
            setColumnSizing((prev) => {
              const next = typeof updater === 'function' ? updater(prev) : updater;
              if (persistKey && typeof window !== 'undefined') {
                try {
                  window.localStorage.setItem(sizingStorageKey(persistKey), JSON.stringify(next));
                } catch {
                  /* ignore quota/availability errors — persistence is best-effort */
                }
              }
              return next;
            }),
        }
      : {}),
    manualPagination: !!effectiveServerSide,
    ...(effectiveServerSide
      ? {
          rowCount: effectiveServerSide.rowCount,
          onPaginationChange: (updater: Updater<PaginationState>) => {
            const next =
              typeof updater === 'function' ? updater(effectiveServerSide.pagination) : updater;
            effectiveServerSide.onPaginationChange(next);
          },
        }
      : {}),
    enableRowSelection: selectable,
    enableSortingRemoval: false, // 2-state toggle (asc ⇄ desc), never back to unsorted
    sortDescFirst: false, // always start ascending, even for numeric columns
    // The old hand-rolled table only reset to page 1 on a *search* change (done
    // explicitly in the input handler) — never on a data refetch. Default
    // autoReset would snap polling tables back to page 1 on every poll, so disable it.
    autoResetPageIndex: false,
    // Global search: match the previous behavior — any *visible* column whose
    // stringified accessor output contains the query. The fn ignores columnId
    // and checks all visible cells, so a single match includes the row.
    globalFilterFn: (row, _columnId, filterValue) => {
      const q = String(filterValue ?? '').toLowerCase().trim();
      if (!q) return true;
      return row.getVisibleCells().some((cell) => {
        const original = colByKey.get(cell.column.id);
        if (!original) return false;
        const val = original.accessor(row.original);
        return val != null && String(val).toLowerCase().includes(q);
      });
    },
    // Programmatic setGlobalFilter routes through the same debounce as typing.
    onGlobalFilterChange: (updater: Updater<string>) =>
      setSearchInput((prev) => (typeof updater === 'function' ? updater(prev) : updater)),
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: (updater: Updater<VisibilityState>) =>
      setColumnVisibility((prev) => {
        const next = typeof updater === 'function' ? updater(prev) : updater;
        // Never allow hiding the last visible column.
        const visibleCount = columns.filter((c) => next[c.key] !== false).length;
        if (visibleCount < 1) return prev;
        if (persistKey && typeof window !== 'undefined') {
          try {
            window.localStorage.setItem(visibilityStorageKey(persistKey), JSON.stringify(next));
          } catch {
            /* ignore quota/availability errors — persistence is best-effort */
          }
        }
        return next;
      }),
    onRowSelectionChange: (updater: Updater<RowSelectionState>) =>
      setRowSelection((prev) => {
        const next = typeof updater === 'function' ? updater(prev) : updater;
        onSelectionChange?.(data.filter((row) => next[keyExtractor(row)]));
        return next;
      }),
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    // In virtualized mode the virtualizer windows the *full* filtered+sorted
    // row model, so we skip the pagination row model entirely — getRowModel()
    // then returns every matching row and the virtualizer renders only the
    // visible window.
    ...(virtualized ? {} : { getPaginationRowModel: getPaginationRowModel() }),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
    initialState: { pagination: { pageSize } },
  });

  // Snap back to page 1 when the debounced search takes effect — the old
  // input handler reset the page on every keystroke; now the reset rides
  // along with the (debounced) filter application instead.
  useEffect(() => {
    if (table.getState().pagination.pageIndex !== 0) table.setPageIndex(0);
  }, [table, globalFilter]);

  // A column is visible unless explicitly toggled off. Derived from the
  // columnVisibility state (which we own) rather than querying the table, so the
  // memo deps are exactly what it reads.
  const activeColumns = useMemo(
    () => columns.filter((c) => columnVisibility[c.key] !== false),
    [columns, columnVisibility]
  );

  // Faceted filters render for visible columns that opted in via `filter`.
  const facetColumns = activeColumns.filter((c) => c.filter);

  const rows = table.getRowModel().rows;
  // Header objects keyed by column id — needed to wire each resizable column's
  // drag handle. Only consulted when `resizable` is true.
  const headerByKey = useMemo(
    () => new Map(table.getHeaderGroups()[0]?.headers.map((h) => [h.column.id, h])),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [table, columnSizing, columnVisibility, columns]
  );
  const selectedRows = table.getSelectedRowModel().rows.map((r) => r.original);
  const filteredCount = table.getFilteredRowModel().rows.length;
  const totalPages = table.getPageCount();
  const page = table.getState().pagination.pageIndex;
  // Footer counts: server mode reports the server total; client mode the
  // filtered-row count. `effPageSize` is the page size actually in effect.
  const effPageSize = effectiveServerSide ? effectiveServerSide.pagination.pageSize : pageSize;
  const totalRows = effectiveServerSide ? effectiveServerSide.rowCount : filteredCount;

  // ---- Virtualization ----
  // The scroll container that the virtualizer measures against. Only used by
  // the virtualized render branch, but the hook must run unconditionally (rules
  // of hooks), so it is always created — it is cheap when `virtualized` is off.
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const estimateSize = density === 'compact' ? 40 : 52;
  const rowVirtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => estimateSize,
    overscan: 12,
  });
  // Roving-tabindex focus index for the virtualized grid body. -1 = none yet.
  const [focusedRowIndex, setFocusedRowIndex] = useState(-1);

  // Move keyboard focus between virtual rows. Because off-screen rows are not
  // mounted, we ask the virtualizer to scroll the target into view first, then
  // focus it on the next frame once it has been rendered.
  const focusRowAt = (target: number) => {
    if (target < 0 || target >= rows.length) return;
    setFocusedRowIndex(target);
    rowVirtualizer.scrollToIndex(target, { align: 'auto' });
    requestAnimationFrame(() => {
      const el =
        scrollRef.current?.querySelector<HTMLElement>(`[data-row-index="${target}"]`) ?? null;
      el?.focus();
    });
  };

  return (
    <div className={cn('space-y-3', className)}>
      {/* Toolbar */}
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2 flex-1">
          {searchable && (
            <div className="relative max-w-sm flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder={searchPlaceholder}
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                className="w-full h-9 pl-9 pr-8 rounded-md border border-border bg-background text-sm
                  placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              />
              {searchInput && (
                <button
                  onClick={() => setSearchInput('')}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
          )}
          {facetColumns.map((col) => {
            const column = table.getColumn(col.key);
            if (!column) return null;
            return (
              <FacetedFilter
                key={col.key}
                column={column}
                label={col.filter?.label ?? col.header}
                onChange={() => table.setPageIndex(0)}
              />
            );
          })}
          {toolbar}
        </div>

        <div className="relative">
          <button
            onClick={() => setShowColumnToggle(!showColumnToggle)}
            className="inline-flex items-center gap-1.5 h-9 px-3 rounded-md border border-border
              text-sm text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
          >
            <SlidersHorizontal className="h-4 w-4" />
            Columns
          </button>

          {showColumnToggle && (
            <div className="absolute right-0 top-full mt-1 w-48 rounded-md border border-border bg-popover p-1 shadow-lg z-50">
              {columns.map((col) => {
                const column = table.getColumn(col.key);
                const isVisible = column?.getIsVisible() ?? true;
                return (
                  <label
                    key={col.key}
                    className="flex items-center gap-2 px-2 py-1.5 rounded text-sm hover:bg-accent cursor-pointer"
                  >
                    <input
                      type="checkbox"
                      checked={isVisible}
                      onChange={() => column?.toggleVisibility()}
                      className="rounded border-border text-primary focus:ring-ring"
                    />
                    {col.header}
                  </label>
                );
              })}
            </div>
          )}
        </div>
      </div>

      {selectable && selectedRows.length > 0 && bulkActions ? (
        <div
          className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-border bg-muted/30 px-3 py-2 text-sm"
          aria-live="polite"
        >
          <span className="text-muted-foreground">
            {selectedRows.length} {selectedRows.length === 1 ? 'row' : 'rows'} selected
          </span>
          <div className="flex flex-wrap items-center gap-2">
            {bulkActions(selectedRows)}
          </div>
        </div>
      ) : null}

      {/* Table — virtualized (DIV grid) branch */}
      {virtualized ? (
        <VirtualizedGrid
          activeColumns={activeColumns}
          table={table}
          rows={rows}
          rowVirtualizer={rowVirtualizer}
          scrollRef={scrollRef}
          totalRows={rows.length}
          selectable={selectable}
          resizable={resizable}
          cellPadding={cellPadding}
          selectPadding={selectPadding}
          rowHeight={estimateSize}
          loading={loading}
          skeletonRows={skeletonRows}
          emptyMessage={emptyMessage}
          isError={isError}
          errorMessage={errorMessage}
          onRetry={onRetry}
          keyExtractor={keyExtractor}
          onRowClick={onRowClick}
          focusedRowIndex={focusedRowIndex}
          setFocusedRowIndex={setFocusedRowIndex}
          focusRowAt={focusRowAt}
        />
      ) : (
      /* Table — default (semantic table) branch */
      <div className="rounded-lg border border-border overflow-hidden">
        <div className="overflow-x-auto">
          <Table className="w-full text-sm">
            <TableHeader>
              <TableRow className="border-b border-border bg-muted/50">
                {selectable && (
                  <TableHead className={cn('w-10', selectPadding)}>
                    <input
                      type="checkbox"
                      checked={table.getIsAllPageRowsSelected()}
                      onChange={table.getToggleAllPageRowsSelectedHandler()}
                      className="rounded border-border text-primary focus:ring-ring"
                    />
                  </TableHead>
                )}
                {activeColumns.map((col) => {
                  const column = table.getColumn(col.key);
                  const sorted = column?.getIsSorted();
                  const header = resizable ? headerByKey.get(col.key) : undefined;
                  return (
                    <TableHead
                      key={col.key}
                      className={cn(
                        cellPadding,
                        'font-medium text-muted-foreground whitespace-nowrap',
                        resizable && 'relative',
                        col.align === 'center' && 'text-center',
                        col.align === 'right' && 'text-right',
                        col.sortable !== false && 'cursor-pointer select-none hover:text-foreground'
                      )}
                      style={
                        resizable
                          ? { width: column?.getSize() }
                          : col.width
                            ? { width: col.width }
                            : undefined
                      }
                      onClick={() => col.sortable !== false && column?.toggleSorting()}
                    >
                      <div
                        className={cn(
                          'flex items-center gap-1',
                          col.align === 'center' && 'justify-center',
                          col.align === 'right' && 'justify-end'
                        )}
                      >
                        {col.header}
                        {col.sortable !== false && (
                          <span className="text-muted-foreground/50">
                            {sorted === 'asc' ? (
                              <ChevronUp className="h-3.5 w-3.5" />
                            ) : sorted === 'desc' ? (
                              <ChevronDown className="h-3.5 w-3.5" />
                            ) : (
                              <ChevronsUpDown className="h-3 w-3" />
                            )}
                          </span>
                        )}
                      </div>
                      {resizable && header?.column.getCanResize() && (
                        <span
                          role="separator"
                          aria-orientation="vertical"
                          data-resize-handle=""
                          onMouseDown={header.getResizeHandler()}
                          onTouchStart={header.getResizeHandler()}
                          onClick={(e) => e.stopPropagation()}
                          className={cn(
                            'absolute right-0 top-0 h-full w-1 cursor-col-resize select-none touch-none',
                            'bg-transparent hover:bg-border/80',
                            header.column.getIsResizing() && 'bg-primary/60'
                          )}
                        />
                      )}
                    </TableHead>
                  );
                })}
              </TableRow>
            </TableHeader>
            <TableBody>
              {isError ? (
                <TableRow>
                  <TableCell
                    colSpan={activeColumns.length + (selectable ? 1 : 0)}
                    className="px-4 py-12 text-center"
                  >
                    <div className="flex flex-col items-center gap-3 text-sm text-destructive">
                      <span>{errorMessage}</span>
                      {onRetry && (
                        <button
                          onClick={onRetry}
                          className="inline-flex items-center gap-1.5 h-8 px-3 rounded-md border border-destructive/50
                            text-destructive hover:bg-destructive/10 transition-colors"
                        >
                          Retry
                        </button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ) : loading ? (
                Array.from({ length: skeletonRows }).map((_, i) => (
                  <TableRow key={i} className="border-b border-border last:border-0">
                    {selectable && <TableCell className={selectPadding}><div className="h-4 w-4 rounded bg-muted animate-pulse" /></TableCell>}
                    {activeColumns.map((col) => (
                      <TableCell
                        key={col.key}
                        className={cellPadding}
                        style={resizable ? { width: table.getColumn(col.key)?.getSize() } : undefined}
                      >
                        <div
                          className="h-4 w-24 max-w-full rounded bg-muted animate-pulse"
                          style={{ width: col.width ? `min(100%, ${col.width})` : undefined }}
                        />
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : rows.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={activeColumns.length + (selectable ? 1 : 0)}
                    className="px-4 py-12 text-center text-muted-foreground"
                  >
                    {emptyMessage}
                  </TableCell>
                </TableRow>
              ) : (
                rows.map((row) => {
                  const key = keyExtractor(row.original);
                  const isSelected = row.getIsSelected();
                  return (
                    <TableRow
                      key={key}
                      className={cn(
                        'border-b border-border last:border-0 transition-colors',
                        onRowClick && 'cursor-pointer hover:bg-muted/50',
                        isSelected && 'bg-muted/30'
                      )}
                      onClick={() => onRowClick?.(row.original)}
                    >
                      {selectable && (
                        <TableCell className={selectPadding} onClick={(e) => e.stopPropagation()}>
                          <input
                            type="checkbox"
                            checked={isSelected}
                            onChange={row.getToggleSelectedHandler()}
                            className="rounded border-border text-primary focus:ring-ring"
                          />
                        </TableCell>
                      )}
                      {activeColumns.map((col) => (
                        <TableCell
                          key={col.key}
                          className={cn(
                            cellPadding,
                            col.align === 'center' && 'text-center',
                            col.align === 'right' && 'text-right'
                          )}
                          style={resizable ? { width: table.getColumn(col.key)?.getSize() } : undefined}
                        >
                          {col.accessor(row.original)}
                        </TableCell>
                      ))}
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </div>
      </div>
      )}

      {/* Pagination — hidden in virtualized mode (the virtualizer windows the
          full row model, so there are no pages). */}
      {!virtualized && totalPages > 1 && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">
            Showing {totalRows === 0 ? 0 : page * effPageSize + 1}-{page * effPageSize + rows.length} of{' '}
            {totalRows.toLocaleString()}
          </span>
          <div className="flex items-center gap-1">
            <button
              onClick={() => table.previousPage()}
              disabled={!table.getCanPreviousPage()}
              className="inline-flex items-center justify-center h-8 w-8 rounded-md border border-border
                text-muted-foreground hover:text-foreground hover:bg-accent disabled:opacity-50
                disabled:pointer-events-none transition-colors"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            {Array.from({ length: Math.min(totalPages, 5) }).map((_, i) => {
              const pageNum = totalPages <= 5 ? i : Math.max(0, Math.min(page - 2, totalPages - 5)) + i;
              return (
                <button
                  key={pageNum}
                  onClick={() => table.setPageIndex(pageNum)}
                  className={cn(
                    'inline-flex items-center justify-center h-8 w-8 rounded-md text-sm transition-colors',
                    pageNum === page
                      ? 'bg-primary text-primary-foreground'
                      : 'text-muted-foreground hover:text-foreground hover:bg-accent'
                  )}
                >
                  {pageNum + 1}
                </button>
              );
            })}
            <button
              onClick={() => table.nextPage()}
              disabled={!table.getCanNextPage()}
              className="inline-flex items-center justify-center h-8 w-8 rounded-md border border-border
                text-muted-foreground hover:text-foreground hover:bg-accent disabled:opacity-50
                disabled:pointer-events-none transition-colors"
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

// ============================================================
// VirtualizedGrid — DIV-based ARIA grid used when `virtualized` is set.
//
// A position:absolute table row breaks native table layout, so the virtualized
// branch does NOT reuse the semantic table/row primitives. Instead it renders:
//   - a scroll container with role="grid" + aria-rowcount (total body rows),
//   - a sticky header row (role="row") of role="columnheader" cells,
//   - a body sized to rowVirtualizer.getTotalSize() containing absolutely
//     positioned virtual rows (role="row", aria-rowindex 1-based, +1 to
//     account for the header), each holding role="gridcell" cells.
// Column widths are shared between the header and rows (via a flex-basis /
// width derived from col.width or the resizable size) so columns stay aligned.
//
// NOTE: because off-screen rows are not mounted, the browser's native Ctrl-F
// find will not match text in rows outside the rendered window. This is an
// inherent trade-off of virtualization.
// ============================================================

function VirtualizedGrid<T>({
  activeColumns,
  table,
  rows,
  rowVirtualizer,
  scrollRef,
  totalRows,
  selectable,
  resizable,
  cellPadding,
  selectPadding,
  rowHeight,
  loading,
  skeletonRows,
  emptyMessage,
  isError,
  errorMessage,
  onRetry,
  keyExtractor,
  onRowClick,
  focusedRowIndex,
  setFocusedRowIndex,
  focusRowAt,
}: {
  activeColumns: Column<T>[];
  table: RtTable<T>;
  rows: RtRow<T>[];
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>;
  scrollRef: React.RefObject<HTMLDivElement | null>;
  totalRows: number;
  selectable: boolean;
  resizable: boolean;
  cellPadding: string;
  selectPadding: string;
  rowHeight: number;
  loading: boolean;
  skeletonRows: number;
  emptyMessage: string;
  isError: boolean;
  errorMessage: string;
  onRetry?: () => void;
  keyExtractor: (row: T) => string;
  onRowClick?: (row: T) => void;
  focusedRowIndex: number;
  setFocusedRowIndex: (i: number) => void;
  focusRowAt: (i: number) => void;
}) {
  // Per-column width style shared by header + body cells so they line up.
  const colStyle = (col: Column<T>): React.CSSProperties => {
    const width = resizable ? `${table.getColumn(col.key)?.getSize()}px` : col.width;
    return width
      ? { width, flex: `0 0 ${width}`, minWidth: width }
      : { flex: '1 1 0', minWidth: 0 };
  };
  const selectColStyle: React.CSSProperties = { flex: '0 0 2.5rem', width: '2.5rem' };

  const alignClass = (col: Column<T>) =>
    cn(col.align === 'center' && 'text-center justify-center', col.align === 'right' && 'text-right justify-end');

  const virtualItems = rowVirtualizer.getVirtualItems();

  return (
    <div className="rounded-lg border border-border overflow-hidden">
      <div
        ref={scrollRef}
        role="grid"
        aria-rowcount={totalRows}
        aria-colcount={activeColumns.length + (selectable ? 1 : 0)}
        aria-multiselectable={selectable || undefined}
        // The grid container is the single Tab entry point. Rows are focused
        // programmatically (arrow keys / click) and stay out of the Tab order,
        // so the grid is reachable even when the previously-focused row has been
        // virtualized out of the DOM. Arrow keys here move focus into the body.
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.target !== e.currentTarget) return;
          if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
            e.preventDefault();
            focusRowAt(focusedRowIndex >= 0 ? focusedRowIndex : 0);
          }
        }}
        className="relative max-h-[28rem] overflow-auto text-sm outline-none focus:ring-1 focus:ring-inset focus:ring-ring"
      >
        {/* Sticky header row */}
        <div
          role="row"
          aria-rowindex={1}
          className="sticky top-0 z-10 flex border-b border-border bg-muted/50 text-muted-foreground"
        >
          {selectable && (
            <div role="columnheader" className={cn('flex items-center', selectPadding)} style={selectColStyle}>
              <input
                type="checkbox"
                checked={table.getIsAllPageRowsSelected()}
                onChange={table.getToggleAllPageRowsSelectedHandler()}
                className="rounded border-border text-primary focus:ring-ring"
              />
            </div>
          )}
          {activeColumns.map((col) => {
            const column = table.getColumn(col.key);
            const sorted = column?.getIsSorted();
            return (
              <div
                key={col.key}
                role="columnheader"
                aria-sort={
                  col.sortable !== false
                    ? sorted === 'asc'
                      ? 'ascending'
                      : sorted === 'desc'
                        ? 'descending'
                        : 'none'
                    : undefined
                }
                className={cn(
                  cellPadding,
                  'flex items-center gap-1 font-medium whitespace-nowrap',
                  col.sortable !== false && 'cursor-pointer select-none hover:text-foreground',
                  alignClass(col)
                )}
                style={colStyle(col)}
                onClick={() => col.sortable !== false && column?.toggleSorting()}
              >
                {col.header}
                {col.sortable !== false && (
                  <span className="text-muted-foreground/50">
                    {sorted === 'asc' ? (
                      <ChevronUp className="h-3.5 w-3.5" />
                    ) : sorted === 'desc' ? (
                      <ChevronDown className="h-3.5 w-3.5" />
                    ) : (
                      <ChevronsUpDown className="h-3 w-3" />
                    )}
                  </span>
                )}
              </div>
            );
          })}
        </div>

        {/* Body */}
        {isError ? (
          <div role="row">
            <div role="gridcell" className="px-4 py-12 text-center">
              <div className="flex flex-col items-center gap-3 text-sm text-destructive">
                <span>{errorMessage}</span>
                {onRetry && (
                  <button
                    onClick={onRetry}
                    className="inline-flex items-center gap-1.5 h-8 px-3 rounded-md border border-destructive/50
                      text-destructive hover:bg-destructive/10 transition-colors"
                  >
                    Retry
                  </button>
                )}
              </div>
            </div>
          </div>
        ) : loading ? (
          <div>
            {Array.from({ length: skeletonRows }).map((_, i) => (
              <div
                key={i}
                role="row"
                className="flex border-b border-border"
                style={{ height: rowHeight }}
              >
                {selectable && (
                  <div role="gridcell" className={cn('flex items-center', selectPadding)} style={selectColStyle}>
                    <div className="h-4 w-4 rounded bg-muted animate-pulse" />
                  </div>
                )}
                {activeColumns.map((col) => (
                  <div
                    key={col.key}
                    role="gridcell"
                    className={cn('flex items-center', cellPadding)}
                    style={colStyle(col)}
                  >
                    <div
                      className="h-4 w-24 max-w-full rounded bg-muted animate-pulse"
                      style={{ width: col.width ? `min(100%, ${col.width})` : undefined }}
                    />
                  </div>
                ))}
              </div>
            ))}
          </div>
        ) : rows.length === 0 ? (
          <div role="row">
            <div role="gridcell" className="px-4 py-12 text-center text-muted-foreground">
              {emptyMessage}
            </div>
          </div>
        ) : (
          <div style={{ height: rowVirtualizer.getTotalSize(), position: 'relative', width: '100%' }}>
            {virtualItems.map((virtualRow) => {
              const row = rows[virtualRow.index];
              const key = keyExtractor(row.original);
              const isSelected = row.getIsSelected();
              return (
                <div
                  key={key}
                  // measureElement reads each row's real height for variable-size
                  // rows; data-index lets the virtualizer key the measurement.
                  ref={rowVirtualizer.measureElement}
                  data-index={virtualRow.index}
                  data-row-index={virtualRow.index}
                  role="row"
                  // 1-based, and +1 again because the sticky header is row 1.
                  aria-rowindex={virtualRow.index + 2}
                  aria-selected={selectable ? isSelected : undefined}
                  // Programmatically focusable only (-1): the grid container owns
                  // the Tab stop, so a virtualized-out focused row can't strand
                  // keyboard users outside the grid.
                  tabIndex={-1}
                  onFocus={() => setFocusedRowIndex(virtualRow.index)}
                  onKeyDown={(e) => {
                    if (e.key === 'ArrowDown') {
                      e.preventDefault();
                      focusRowAt(virtualRow.index + 1);
                    } else if (e.key === 'ArrowUp') {
                      e.preventDefault();
                      focusRowAt(virtualRow.index - 1);
                    } else if (e.key === 'Enter' && onRowClick) {
                      e.preventDefault();
                      onRowClick(row.original);
                    }
                  }}
                  onClick={() => onRowClick?.(row.original)}
                  className={cn(
                    'absolute left-0 top-0 flex w-full border-b border-border transition-colors',
                    'focus:outline-none focus:ring-1 focus:ring-inset focus:ring-ring',
                    onRowClick && 'cursor-pointer hover:bg-muted/50',
                    isSelected && 'bg-muted/30'
                  )}
                  style={{ transform: `translateY(${virtualRow.start}px)` }}
                >
                  {selectable && (
                    <div
                      role="gridcell"
                      className={cn('flex items-center', selectPadding)}
                      style={selectColStyle}
                      onClick={(e) => e.stopPropagation()}
                    >
                      <input
                        type="checkbox"
                        checked={isSelected}
                        onChange={row.getToggleSelectedHandler()}
                        className="rounded border-border text-primary focus:ring-ring"
                      />
                    </div>
                  )}
                  {activeColumns.map((col) => (
                    <div
                      key={col.key}
                      role="gridcell"
                      className={cn('flex items-center', cellPadding, alignClass(col))}
                      style={colStyle(col)}
                    >
                      {col.accessor(row.original)}
                    </div>
                  ))}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

// ============================================================
// FacetedFilter — multi-select dropdown driven by the column's faceted
// unique values. Rendered in the toolbar for columns with a `filter` config.
// ============================================================

function FacetedFilter<T>({
  column,
  label,
  onChange,
}: {
  column: RtColumn<T, unknown>;
  label: string;
  onChange?: () => void;
}) {
  const [open, setOpen] = useState(false);
  const selected = (column.getFilterValue() as string[] | undefined) ?? [];
  const options = Array.from(column.getFacetedUniqueValues().keys())
    .map((v) => String(v))
    .filter((v) => v !== '')
    .sort();

  const apply = (next: string[]) => {
    column.setFilterValue(next.length ? next : undefined);
    onChange?.();
  };
  const toggle = (value: string) =>
    apply(selected.includes(value) ? selected.filter((v) => v !== value) : [...selected, value]);

  return (
    <div className="relative">
      <button
        onClick={() => setOpen((o) => !o)}
        className={cn(
          'inline-flex items-center gap-1.5 h-9 px-3 rounded-md border border-border text-sm transition-colors',
          selected.length > 0
            ? 'border-primary/50 text-foreground bg-accent'
            : 'text-muted-foreground hover:text-foreground hover:bg-accent'
        )}
      >
        <Filter className="h-3.5 w-3.5" />
        {label}
        {selected.length > 0 && (
          <span className="ml-0.5 inline-flex items-center justify-center min-w-5 h-5 px-1 rounded-full bg-primary text-primary-foreground text-2xs">
            {selected.length}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute left-0 top-full mt-1 w-52 rounded-md border border-border bg-popover p-1 shadow-lg z-50">
          {options.length === 0 ? (
            <p className="px-2 py-1.5 text-sm text-muted-foreground">No values</p>
          ) : (
            options.map((opt) => (
              <label
                key={opt}
                className="flex items-center gap-2 px-2 py-1.5 rounded text-sm hover:bg-accent cursor-pointer"
              >
                <input
                  type="checkbox"
                  checked={selected.includes(opt)}
                  onChange={() => toggle(opt)}
                  className="rounded border-border text-primary focus:ring-ring"
                />
                {opt}
              </label>
            ))
          )}
          {selected.length > 0 && (
            <button
              onClick={() => apply([])}
              className="w-full mt-1 px-2 py-1.5 rounded text-sm text-muted-foreground hover:text-foreground hover:bg-accent text-left"
            >
              Clear filter
            </button>
          )}
        </div>
      )}
    </div>
  );
}
