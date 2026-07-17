import type { InflightTaskTrace } from '../types/inflight-trace'

const databaseName = 'inflight-trace-local-cache'
const archiveStoreName = 'archives'

type InflightTraceArchiveCacheEntry = {
  fileName: string
  content: Blob
}

export type InflightTraceLocalCacheFile = {
  fileName: string
  sizeBytes: number
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
        .put({ fileName, content: response })
      transaction.addEventListener('complete', () => resolve())
      transaction.addEventListener('error', () => reject(transaction.error))
      transaction.addEventListener('abort', () => reject(transaction.error))
    })
  } finally {
    database.close()
  }
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
      }))
      .sort((left, right) => left.fileName.localeCompare(right.fileName))
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
