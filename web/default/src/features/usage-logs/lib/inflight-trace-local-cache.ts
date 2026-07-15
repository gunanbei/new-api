import type { InflightTaskTrace } from '../types/inflight-trace'

type FileHandle = {
  kind: 'file'
  name: string
  getFile: () => Promise<File>
  createWritable: () => Promise<{ write: (data: Blob) => Promise<void>; close: () => Promise<void> }>
}

type DirectoryHandle = {
  kind: 'directory'
  getFileHandle: (name: string, options?: { create?: boolean }) => Promise<FileHandle>
  entries: () => AsyncIterableIterator<[string, FileHandle | DirectoryHandle]>
  removeEntry: (name: string) => Promise<void>
}

declare global {
  interface Window {
    showDirectoryPicker?: () => Promise<DirectoryHandle>
  }
}

const databaseName = 'inflight-trace-local-cache'
const storeName = 'settings'
const directoryKey = 'directory'

function openDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(databaseName, 1)
    request.onupgradeneeded = () => request.result.createObjectStore(storeName)
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
}

async function readDirectory(): Promise<DirectoryHandle | null> {
  const database = await openDatabase()
  return new Promise((resolve, reject) => {
    const request = database.transaction(storeName).objectStore(storeName).get(directoryKey)
    request.onsuccess = () => resolve((request.result as DirectoryHandle | undefined) ?? null)
    request.onerror = () => reject(request.error)
  })
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

export function supportsInflightTraceLocalCache(): boolean {
  return typeof window !== 'undefined' && Boolean(window.showDirectoryPicker)
}

export async function selectInflightTraceLocalCacheDirectory(): Promise<DirectoryHandle> {
  if (!window.showDirectoryPicker) throw new Error('Local cache is only supported in Chromium browsers.')
  const directory = await window.showDirectoryPicker()
  const database = await openDatabase()
  await new Promise<void>((resolve, reject) => {
    const request = database.transaction(storeName, 'readwrite').objectStore(storeName).put(directory, directoryKey)
    request.onsuccess = () => resolve()
    request.onerror = () => reject(request.error)
  })
  return directory
}

export async function cacheInflightTraceArchive(fileName: string, response: Blob): Promise<void> {
  let directory = await readDirectory()
  if (!directory) directory = await selectInflightTraceLocalCacheDirectory()
  const file = await directory.getFileHandle(fileName, { create: true })
  const writable = await file.createWritable()
  await writable.write(response)
  await writable.close()
}

export async function readInflightTraceArchiveRecord(
  fileName: string,
  requestId: string
): Promise<InflightTaskTrace | null> {
  const directory = await readDirectory()
  if (!directory) return null
  try {
    const file = await directory.getFileHandle(fileName)
    const rows = parseCSVRows(await (await file.getFile()).text())
    for (const row of rows) {
      if (row.length !== 3 || row[0] !== requestId) continue
      return JSON.parse(row[2]) as InflightTaskTrace
    }
  } catch {
    return null
  }
  return null
}

export async function listInflightTraceLocalCacheFiles(): Promise<string[]> {
  const directory = await readDirectory()
  if (!directory) return []
  const files: string[] = []
  for await (const [name, handle] of directory.entries()) {
    if (handle.kind === 'file' && name.startsWith('Inflight_') && name.endsWith('.csv')) files.push(name)
  }
  return files.sort()
}

export async function deleteInflightTraceLocalCacheFiles(fileNames: string[]): Promise<void> {
  const directory = await readDirectory()
  if (!directory) return
  await Promise.all(fileNames.map((fileName) => directory.removeEntry(fileName)))
}
