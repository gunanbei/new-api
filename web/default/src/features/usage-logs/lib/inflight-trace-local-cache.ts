import type { InflightTaskTrace } from '../types/inflight-trace'

const databaseName = 'inflight-trace-local-cache'
const archiveStoreName = 'archives'
export const inflightTraceLocalCacheRetentionStorageKey =
  'usage-logs:inflight-trace:cache-retention'

export type InflightTraceLocalCacheRetention = 0 | 10 | 50 | 100

type InflightTraceArchiveCacheEntry = {
  fileName: string
  content: Blob
  cachedAt?: number
}

export type InflightTraceLocalCacheFile = {
  fileName: string
  sizeBytes: number
  cachedAt?: number
}

export function getInflightTraceLocalCacheRetention(): InflightTraceLocalCacheRetention {
  if (typeof window === 'undefined') return 0
  try {
    const value = Number(
      window.localStorage.getItem(inflightTraceLocalCacheRetentionStorageKey)
    )
    return value === 10 || value === 50 || value === 100 ? value : 0
  } catch {
    return 0
  }
}

export function setInflightTraceLocalCacheRetention(
  retention: InflightTraceLocalCacheRetention
): void {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(
      inflightTraceLocalCacheRetentionStorageKey,
      String(retention)
    )
  } catch {
    // localStorage can be unavailable in privacy-restricted browser contexts.
  }
}

function openDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(databaseName, 2)
    request.addEventListener('upgradeneeded', () => {
      if (!request.result.objectStoreNames.contains(archiveStoreName)) {
        request.result.createObjectStore(archiveStoreName, {
          keyPath: 'fileName',
        })
      }
    })
    request.addEventListener('success', () => resolve(request.result))
    request.addEventListener('error', () => reject(request.error))
  })
}

async function readArchive(
  fileName: string
): Promise<InflightTraceArchiveCacheEntry | null> {
  const database = await openDatabase()
  try {
    return await new Promise((resolve, reject) => {
      const request = database
        .transaction(archiveStoreName)
        .objectStore(archiveStoreName)
        .get(fileName)
      request.addEventListener('success', () =>
        resolve(
          (request.result as InflightTraceArchiveCacheEntry | undefined) ?? null
        )
      )
      request.addEventListener('error', () => reject(request.error))
    })
  } finally {
    database.close()
  }
}

function parseCSVRows(csv: string): string[][] {
  const rows: string[][] = []
  let row: string[] = []
  let value = ''
  let quoted = false

  for (let index = 0; index < csv.length; index += 1) {
    const character = csv[index]
    if (quoted) {
      if (character === '"' && csv[index + 1] === '"') {
        value += '"'
        index += 1
      } else if (character === '"') {
        quoted = false
      } else {
        value += character
      }
      continue
    }
    if (character === '"') {
      quoted = true
    } else if (character === ',') {
      row.push(value)
      value = ''
    } else if (character === '\n') {
      row.push(value)
      rows.push(row)
      row = []
      value = ''
    } else if (character !== '\r') {
      value += character
    }
  }
  if (value || row.length > 0) {
    row.push(value)
    rows.push(row)
  }
  return rows
}

export async function cacheInflightTraceArchive(
  fileName: string,
  response: Blob
): Promise<void> {
  const database = await openDatabase()
  try {
    await new Promise<void>((resolve, reject) => {
      const transaction = database.transaction(archiveStoreName, 'readwrite')
      transaction
        .objectStore(archiveStoreName)
        .put({ fileName, content: response, cachedAt: Date.now() })
      transaction.addEventListener('complete', () => resolve())
      transaction.addEventListener('error', () => reject(transaction.error))
      transaction.addEventListener('abort', () => reject(transaction.error))
    })
  } finally {
    database.close()
  }
  await pruneInflightTraceLocalCacheFiles()
}

export async function readInflightTraceArchiveRecord(
  fileName: string,
  requestId: string
): Promise<InflightTaskTrace | null> {
  try {
    const archive = await readArchive(fileName)
    if (!archive) return null
    const rows = parseCSVRows(await archive.content.text())
    for (const row of rows) {
      if (row.length !== 3 || row[0] !== requestId) continue
      return JSON.parse(row[2]) as InflightTaskTrace
    }
  } catch {
    return null
  }
  return null
}

export async function readInflightTraceLocalCacheRows(
  fileName: string
): Promise<string[][] | null> {
  const archive = await readArchive(fileName)
  return archive ? parseCSVRows(await archive.content.text()) : null
}

export async function listInflightTraceLocalCacheFiles(): Promise<
  InflightTraceLocalCacheFile[]
> {
  const database = await openDatabase()
  try {
    const archives = await new Promise<InflightTraceArchiveCacheEntry[]>(
      (resolve, reject) => {
        const request = database
          .transaction(archiveStoreName)
          .objectStore(archiveStoreName)
          .getAll()
        request.addEventListener('success', () => resolve(request.result))
        request.addEventListener('error', () => reject(request.error))
      }
    )
    return archives
      .filter(
        (archive) =>
          archive.fileName.startsWith('Inflight_') &&
          archive.fileName.endsWith('.csv')
      )
      .map((archive) => ({
        fileName: archive.fileName,
        sizeBytes: archive.content.size,
        cachedAt: archive.cachedAt,
      }))
      .sort(
        (left, right) =>
          (left.cachedAt ?? 0) - (right.cachedAt ?? 0) ||
          left.fileName.localeCompare(right.fileName)
      )
  } finally {
    database.close()
  }
}

export async function deleteInflightTraceLocalCacheFiles(
  fileNames: string[]
): Promise<void> {
  if (fileNames.length === 0) return
  const database = await openDatabase()
  try {
    await new Promise<void>((resolve, reject) => {
      const transaction = database.transaction(archiveStoreName, 'readwrite')
      const store = transaction.objectStore(archiveStoreName)
      for (const fileName of fileNames) store.delete(fileName)
      transaction.addEventListener('complete', () => resolve())
      transaction.addEventListener('error', () => reject(transaction.error))
      transaction.addEventListener('abort', () => reject(transaction.error))
    })
  } finally {
    database.close()
  }
}

export async function pruneInflightTraceLocalCacheFiles(
  retention = getInflightTraceLocalCacheRetention()
): Promise<void> {
  if (retention === 0) return
  const files = await listInflightTraceLocalCacheFiles()
  if (files.length <= retention) return
  await deleteInflightTraceLocalCacheFiles(
    files.slice(0, files.length - retention).map((file) => file.fileName)
  )
}
