import { useCallback, useEffect, useState } from "react"
import {
  addOne,
  loadDraft,
  lowerOne,
  removeRow,
  saveDraft,
} from "./draft-addition"
import type { DraftRow } from "./draft-addition"

/** The Draft Addition for one Collection, mirrored to local storage on every change. Remount (or `key`) per Collection. */
export function useDraftAddition(collectionId: string) {
  const [rows, setRows] = useState<DraftRow[]>(() => loadDraft(collectionId))

  useEffect(() => saveDraft(collectionId, rows), [collectionId, rows])

  return {
    rows,
    add: useCallback((cardId: string) => setRows((r) => addOne(r, cardId)), []),
    lower: useCallback(
      (cardId: string) => setRows((r) => lowerOne(r, cardId)),
      []
    ),
    remove: useCallback(
      (cardId: string) => setRows((r) => removeRow(r, cardId)),
      []
    ),
    /** Takes back one copy: the last one removes the row. */
    undo: useCallback(
      (cardId: string) =>
        setRows((r) =>
          r.find((x) => x.cardId === cardId)?.quantity === 1
            ? removeRow(r, cardId)
            : lowerOne(r, cardId)
        ),
      []
    ),
    discard: useCallback(() => setRows([]), []),
  }
}
