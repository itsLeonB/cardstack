import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react"
import type { ReactNode } from "react"
import {
  defaultRangeExtractor,
  useWindowVirtualizer,
} from "@tanstack/react-virtual"
import type { Range } from "@tanstack/react-virtual"
import {
  GRID_GAP,
  chunkRows,
  columnsForWidth,
  estimateRowHeight,
  shouldLoadMore,
  withPinnedRow,
} from "@/lib/grid-layout"

const OVERSCAN = 3
// The sticky site header (h-14) covers the top of the window; keep a scrolled-to row below it.
const HEADER_HEIGHT = 56

// Row heights measured so far, by row key (column count plus first card),
// saved when a grid unmounts. Kept across mounts so that coming back to a list
// (browser back) starts from real heights: with the estimate alone, tall rows
// make the document shorter than it was and the router's one-shot scroll
// restoration lands a row or more away from where the user left.
// ponytail: grows with every row ever rendered (tens of bytes each); cap or
// clear it if sessions ever render hundreds of thousands of rows.
const measuredRowHeights = new Map<string, number>()

interface VirtualGridProps<T> {
  items: T[]
  getKey: (item: T) => string
  renderItem: (item: T) => ReactNode
  /** Size of the whole set, for `aria-setsize`, when more is still to load. Defaults to `items.length`. */
  totalCount?: number
  /** Whether scrolling to the end may start a load: more to fetch, nothing in flight, no error. */
  canLoadMore: boolean
  onLoadMore: () => void
}

/**
 * Window-scrolled grid that renders only the rows near the viewport. Columns
 * follow the container width; rows are measured because tile height varies.
 *
 * Markup: a `role="list"` whose presentational row wrappers hold
 * `role="listitem"` tiles with `aria-posinset` / `aria-setsize` (a `ul` of
 * rows is invalid HTML and `ul > div` fails axe). The row holding focus stays
 * rendered, and when the column count changes the viewport is re-anchored on
 * the card that was at the top.
 *
 * Known limitation of virtualizing: browser find-in-page (Ctrl+F) only sees
 * the rendered rows, so it cannot find a loaded card that is scrolled out.
 */
export function VirtualGrid<T>({
  items,
  getKey,
  renderItem,
  totalCount = items.length,
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

  // The row holding focus. `keyboard` is true when it was reached by keyboard
  // (`:focus-visible`), which pauses scroll-triggered loading below.
  const [focusState, setFocus] = useState<{
    row: number
    keyboard: boolean
    columns: number
  } | null>(null)
  // A row index means nothing under another column count (its tiles remount).
  const focus = focusState?.columns === columns ? focusState : null
  const focusedRow = focus?.row ?? null
  const rangeExtractor = useCallback(
    (range: Range) => {
      const indexes = defaultRangeExtractor(range)
      // Never pin a row that no longer exists (list shortened or re-chunked).
      return focusedRow !== null && focusedRow < range.count
        ? withPinnedRow(indexes, focusedRow)
        : indexes
    },
    [focusedRow]
  )

  const virtualizer = useWindowVirtualizer({
    count: rows.length,
    estimateSize: (index) =>
      measuredRowHeights.get(getItemKey(index)) ??
      estimateRowHeight(layout.width, columns),
    getItemKey,
    gap: GRID_GAP,
    overscan: OVERSCAN,
    rangeExtractor,
    scrollMargin: layout.offsetTop,
    scrollPaddingStart: HEADER_HEIGHT,
  })

  const virtualRows = virtualizer.getVirtualItems()
  // From the visible range, not the last rendered row: a pinned focused row
  // far below the window must not look like "scrolled to the end".
  const firstVisibleRow = virtualizer.range?.startIndex ?? 0
  const lastRenderedRow = virtualizer.range
    ? Math.min(virtualizer.range.endIndex + OVERSCAN, rows.length - 1)
    : undefined
  useEffect(() => {
    if (
      shouldLoadMore({
        lastRenderedRow,
        rowCount: rows.length,
        // While tabbing through tiles, scrolling a new page in would push the
        // end of the list away from the keyboard user; "Load more" is theirs.
        canLoad: canLoadMore && !focus?.keyboard,
      })
    ) {
      onLoadMore()
    }
  }, [lastRenderedRow, rows.length, canLoadMore, focus?.keyboard, onLoadMore])

  // Remember the card at the top of the viewport; when the column count
  // changes, put its row back at the top. Deferred one frame: scrolling in
  // the same commit as the layout change landed ~15 rows off when the document
  // got shorter. Skipped at the first card, which also covers the initial
  // 2-to-n columns settle on mount (it must not fight scroll restoration).
  const anchor = useRef({ columns, card: 0 })
  const anchorFrame = useRef(0)
  // No dependency array on purpose: while the columns are stable this tracks
  // the top card after every render (scrolls included).
  useEffect(() => {
    const tracked = anchor.current
    if (tracked.columns === columns) {
      tracked.card = firstVisibleRow * columns
      return
    }
    tracked.columns = columns
    if (tracked.card === 0) return
    const target = Math.floor(tracked.card / columns)
    // Two column changes within a frame must end in one scroll.
    cancelAnimationFrame(anchorFrame.current)
    anchorFrame.current = requestAnimationFrame(() =>
      // Never smooth: no animation, whatever the reduced-motion setting.
      virtualizer.scrollToIndex(target, { align: "start", behavior: "instant" })
    )
  })
  useEffect(
    () => () => {
      cancelAnimationFrame(anchorFrame.current)
      for (const [key, size] of virtualizer.itemSizeCache) {
        measuredRowHeights.set(String(key), size)
      }
    },
    // The virtualizer instance is stable for the component's life.
    [virtualizer]
  )

  // Browsers don't reliably fire blur when the focused tile unmounts (column
  // change, filter change), which would leave a stale pin and a stuck
  // keyboard pause: drop the focus state once focus is no longer in the list.
  // No dependency array: it must check after every render.
  useEffect(() => {
    if (focusState && !listRef.current?.contains(document.activeElement)) {
      setFocus(null)
    }
  })

  return (
    <div
      ref={listRef}
      role="list"
      className="relative"
      style={{ height: virtualizer.getTotalSize() }}
      onFocus={(event) => {
        const row = event.target.closest<HTMLElement>("[data-index]")
        if (row) {
          setFocus({
            row: Number(row.dataset.index),
            keyboard: event.target.matches(":focus-visible"),
            columns,
          })
        }
      }}
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) setFocus(null)
      }}
    >
      {virtualRows.map((row) => (
        <div
          key={row.key}
          role="presentation"
          ref={virtualizer.measureElement}
          data-index={row.index}
          className="absolute top-0 left-0 grid w-full gap-4"
          style={{
            gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
            transform: `translateY(${row.start - layout.offsetTop}px)`,
          }}
        >
          {rows[row.index]!.map((item, column) => (
            <div
              key={getKey(item)}
              role="listitem"
              aria-posinset={row.index * columns + column + 1}
              aria-setsize={totalCount}
            >
              {renderItem(item)}
            </div>
          ))}
        </div>
      ))}
    </div>
  )
}
