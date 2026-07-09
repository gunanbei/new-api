/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
export const TRACE_BODY_HEAVY_BYTES = 64 * 1024

function formatBytesBrief(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
}

function isHeavyJsonField(key: string, value: unknown) {
  if (typeof value !== 'string' || value.length <= 64) {
    return false
  }
  const normalized = key.toLowerCase()
  return (
    normalized === 'b64_json' ||
    normalized === 'image' ||
    normalized.endsWith('_b64') ||
    normalized.includes('base64')
  )
}

function summarizeHeavyValue(key: string, value: string) {
  const bytes = new TextEncoder().encode(value).length
  return `[${key}: ${formatBytesBrief(bytes)}]`
}

function redactHeavyJsonValue(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(redactHeavyJsonValue)
  }
  if (value && typeof value === 'object') {
    const record = value as Record<string, unknown>
    const result: Record<string, unknown> = {}
    for (const [key, child] of Object.entries(record)) {
      if (typeof child === 'string' && isHeavyJsonField(key, child)) {
        result[key] = summarizeHeavyValue(key, child)
      } else {
        result[key] = redactHeavyJsonValue(child)
      }
    }
    return result
  }
  if (
    typeof value === 'string' &&
    value.length > 256 &&
    (value.startsWith('data:image/') || /^[A-Za-z0-9+/=]{256,}$/.test(value))
  ) {
    return summarizeHeavyValue('payload', value)
  }
  return value
}

export function isHeavyTraceJsonBody(
  text: string,
  contentType?: string,
  traceKind?: string
) {
  if (!text) {
    return false
  }
  if (text.length >= TRACE_BODY_HEAVY_BYTES) {
    return true
  }
  if (traceKind === 'image') {
    return true
  }
  if (
    contentType?.includes('application/json') &&
    /"b64_json"\s*:/.test(text)
  ) {
    return true
  }
  return /"data:image\/[^"]+;base64,/.test(text)
}

export function shouldUseInstantTraceBodyRender(options: {
  liveUpdate?: boolean
  isSse?: boolean
}) {
  return Boolean(options.liveUpdate || options.isSse)
}

export function formatTraceJsonForDisplay(text: string) {
  try {
    return JSON.stringify(redactHeavyJsonValue(JSON.parse(text)), null, 2)
  } catch {
    return text
  }
}

export function formatJsonText(text: string) {
  try {
    return JSON.stringify(JSON.parse(text), null, 2)
  } catch {
    return text
  }
}
