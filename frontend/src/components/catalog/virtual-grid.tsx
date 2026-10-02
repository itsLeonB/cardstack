import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react"
import type { ReactNode } from "react"
import { useWindowVirtualizer } from "@tanstack/react-virtual"
import {
  GRID_GAP,
  chunkRows,
  columnsForWidth,
  estimateRowHeight,
  shouldLoadMore,
} from "@/lib/grid-layout"

// The sticky site header (h-14) covers the top of the window.
const HEADER_HEIGHT = 56

interface VirtualGridProps<T> {
  items: T[]
  getKey: (item: T) => string
  renderItem: (item: T) => ReactNode
  /** Whether scrolling to the end may start a load: more to fetch, nothing in flight, no error. */
  canLoadMore: boolean
  onLoadMore: () => void
}

/**
 * Window-scrolled grid that renders only the rows near the viewport. Columns
 * follow the container width; rows are measured because tile height varies.
 * Row `<ul>`s keep each tile an `li`.
 */
export function VirtualGrid<T>({
  items,
  getKey,
  renderItem,
  canLoadMore,
  onLoadMore,
}: VirtualGridProps<T>) {
  const listRef = useRef<HTMLDivElement>(null)
  const [layout, setLayout] = useState({ width: 0, offsetTop: 0 })

  // The list's offset from the document top (header plus filters above it) is
  // the virtualizer's scroll margin; re-read it when the container changes
  // size, or when the page (`main`) does, since content above can grow.
  useLayoutEffect(() => {
    const list = listRef.current
    if (!list) return
    const measure = () => {
      const { width, top } = list.getBoundingClientRect()
      const offsetTop = top + window.scrollY
      setLayout((prev) =>
        prev.width === width && prev.offsetTop === offsetTop
          ? prev
          : { width, offsetTop }
      )
    }
    measure()
    const observer = new ResizeObserver(measure)
    observer.observe(list)
    const page = list.closest("main") ?? list.parentElement
    if (page) observer.observe(page)
    return () => observer.disconnect()
  }, [])

  const columns = columnsForWidth(layout.width)
  const rows = useMemo(() => chunkRows(items, columns), [items, columns])
  // Keyed by column count too, so heights measured at one width are never reused at another.
  const getItemKey = useCallback(
    (index: number) => `${columns}:${getKey(rows[index]![0]!)}`,
    [columns, getKey, rows]
  )

  const virtualizer = useWindowVirtualizer({
    count: rows.length,
    estimateSize: () => estimateRowHeight(layout.width, columns),
    getItemKey,
    gap: GRID_GAP,
    overscan: 3,
    scrollMargin: layout.offsetTop,
    scrollPaddingStart: HEADER_HEIGHT,
  })

  const virtualRows = virtualizer.getVirtualItems()
  const lastRenderedRow = virtualRows.at(-1)?.index
  useEffect(() => {
    if (
      shouldLoadMore({
        lastRenderedRow,
        rowCount: rows.length,
        canLoad: canLoadMore,
      })
    ) {
      onLoadMore()
    }
  }, [lastRenderedRow, rows.length, canLoadMore, onLoadMore])

  return (
    <div
      ref={listRef}
      className="relative"
      style={{ height: virtualizer.getTotalSize() }}
    >
      {virtualRows.map((row) => (
        <ul
          key={row.key}
          ref={virtualizer.measureElement}
          data-index={row.index}
          className="absolute top-0 left-0 grid w-full gap-4"
          style={{
            gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
            transform: `translateY(${row.start - virtualizer.options.scrollMargin}px)`,
          }}
        >
          {rows[row.index]!.map((item) => (
            <li key={getKey(item)}>{renderItem(item)}</li>
          ))}
        </ul>
      ))}
    </div>
  )
}
