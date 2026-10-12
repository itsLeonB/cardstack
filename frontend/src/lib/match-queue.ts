/** How many match requests run at once; further scans wait as queued. */
export const MAX_MATCHES_IN_FLIGHT = 3

/**
 * Runs match requests with at most `limit` in flight, first in first out. A
 * cancelled or timed-out request is aborted and frees its slot at once. `run`
 * should settle when its request does; a rejection is swallowed so the queue
 * keeps going.
 */
export class MatchQueue<T> {
  private waiting: { id: string; item: T }[] = []
  private running = new Map<string, AbortController>()

  constructor(
    private readonly limit: number,
    private readonly run: (
      id: string,
      item: T,
      signal: AbortSignal
    ) => Promise<void>,
    private readonly timeoutMs = Infinity
  ) {}

  add(id: string, item: T) {
    this.waiting.push({ id, item })
    this.pump()
  }

  cancel(id: string) {
    this.waiting = this.waiting.filter((w) => w.id !== id)
    const controller = this.running.get(id)
    if (controller) {
      this.running.delete(id)
      controller.abort()
      this.pump()
    }
  }

  cancelAll() {
    for (const id of [...this.running.keys()]) this.cancel(id)
    this.waiting = []
  }

  private pump() {
    while (this.running.size < this.limit && this.waiting.length > 0) {
      const next = this.waiting.shift()!
      const controller = new AbortController()
      this.running.set(next.id, controller)
      const free = () => {
        if (this.running.get(next.id) !== controller) return false
        this.running.delete(next.id)
        this.pump()
        return true
      }
      const timer =
        this.timeoutMs === Infinity
          ? undefined
          : setTimeout(() => {
              if (free()) controller.abort()
            }, this.timeoutMs)
      this.run(next.id, next.item, controller.signal)
        .catch(() => {})
        .finally(() => {
          clearTimeout(timer)
          free()
        })
    }
  }
}
