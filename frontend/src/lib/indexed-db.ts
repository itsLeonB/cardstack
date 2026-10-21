/**
 * Runs one request on an object store of an IndexedDB database the app owns,
 * creating the database and the store on first use. The store's key is the
 * fields named by `keyPath`. Rejects when IndexedDB cannot be used (missing,
 * private mode, quota), so callers can fall back to memory.
 */
export function transactStore<T>(
  dbName: string,
  storeName: string,
  keyPath: string[],
  mode: IDBTransactionMode,
  work: (store: IDBObjectStore) => IDBRequest<T>
): Promise<T> {
  return new Promise((resolve, reject) => {
    const open = indexedDB.open(dbName, 1)
    open.onupgradeneeded = () => {
      open.result.createObjectStore(storeName, { keyPath })
    }
    open.onerror = () => reject(open.error)
    open.onsuccess = () => {
      const db = open.result
      const tx = db.transaction(storeName, mode)
      const request = work(tx.objectStore(storeName))
      tx.oncomplete = () => {
        db.close()
        resolve(request.result)
      }
      tx.onabort = () => {
        db.close()
        reject(tx.error)
      }
    }
  })
}
