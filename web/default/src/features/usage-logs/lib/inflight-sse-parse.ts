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
import { JSONPath } from 'jsonpath-plus'

export type SseParseStrategy =
  | 'openai'
  | 'gemini'
  | 'claude'
  | 'ollama_generate'
  | 'ollama_chat'
  | 'custom'

export type ParsedSseEvent = {
  index: number
  raw: string
  dataPayload: string | null
  parsed: unknown | null
  parseError: string | null
  isDone: boolean
  extracted: string
  extractError: string | null
}

export type ParsedSseResult = {
  events: ParsedSseEvent[]
  concatenated: string
}

function stringifyExtracted(value: unknown): string {
  if (value === null || value === undefined) {
    return ''
  }
  if (typeof value === 'string') {
    return value
  }
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value)
  }
  return JSON.stringify(value)
}

function extractWithJsonPath(parsed: unknown, path: string): string {
  if (!path.trim()) {
    return ''
  }
  try {
    const matches = JSONPath({
      path,
      json: parsed as object,
      wrap: false,
    })
    if (matches === undefined || matches === null) {
      return ''
    }
    if (Array.isArray(matches)) {
      return matches
        .map((item) => stringifyExtracted(item))
        .filter(Boolean)
        .join('')
    }
    return stringifyExtracted(matches)
  } catch {
    return ''
  }
}

function extractOpenAi(parsed: unknown): string {
  if (!parsed || typeof parsed !== 'object') {
    return ''
  }
  const record = parsed as Record<string, unknown>
  const type = typeof record.type === 'string' ? record.type : ''
  if (type === 'response.output_text.delta' && typeof record.delta === 'string') {
    return record.delta
  }
  if (
    type === 'image_generation.partial_image' ||
    type === 'image_generation.completed'
  ) {
    if (typeof record.url === 'string' && record.url.trim()) {
      return record.url
    }
    if (typeof record.b64_json === 'string' && record.b64_json.length > 0) {
      const kb = (record.b64_json.length / 1024).toFixed(1)
      return `[image payload: ${kb} KB]`
    }
    return `[${type}]`
  }
  const parts = [
    extractWithJsonPath(parsed, '$.choices[0].delta.content'),
    extractWithJsonPath(parsed, '$.choices[0].delta.reasoning_content'),
  ].filter(Boolean)
  return parts.join('')
}

function extractByStrategy(
  parsed: unknown,
  strategy: SseParseStrategy,
  customPath: string
): string {
  switch (strategy) {
    case 'openai':
      return extractOpenAi(parsed)
    case 'gemini':
      return extractWithJsonPath(parsed, '$.candidates[0].content.parts[0].text')
    case 'claude':
      return extractWithJsonPath(parsed, '$.delta.text')
    case 'ollama_generate':
      return extractWithJsonPath(parsed, '$.response')
    case 'ollama_chat':
      return extractWithJsonPath(parsed, '$.message.content')
    case 'custom':
      return extractWithJsonPath(parsed, customPath)
    default:
      return ''
  }
}

export function parseSseDataPayload(chunk: string): string | null {
  const dataLines = chunk
    .split('\n')
    .filter((line) => line.startsWith('data:'))
    .map((line) => line.slice(5).trimStart())
  if (dataLines.length === 0) {
    return null
  }
  return dataLines.join('\n')
}

export function parseSseChunks(text: string): string[] {
  return text.split('\n\n').filter((chunk) => chunk.trim().length > 0)
}

export function parseSseTrace(
  text: string,
  strategy: SseParseStrategy,
  customPath: string
): ParsedSseResult {
  const chunks = parseSseChunks(text)
  const events: ParsedSseEvent[] = []
  const extractedParts: string[] = []

  chunks.forEach((chunk, index) => {
    const dataPayload = parseSseDataPayload(chunk)
    if (dataPayload === null) {
      events.push({
        index: index + 1,
        raw: chunk,
        dataPayload: null,
        parsed: null,
        parseError: null,
        isDone: false,
        extracted: '',
        extractError: null,
      })
      return
    }

    if (dataPayload === '[DONE]') {
      events.push({
        index: index + 1,
        raw: chunk,
        dataPayload,
        parsed: null,
        parseError: null,
        isDone: true,
        extracted: '',
        extractError: null,
      })
      return
    }

    let parsed: unknown = null
    let parseError: string | null = null
    try {
      parsed = JSON.parse(dataPayload)
    } catch (error) {
      parseError = error instanceof Error ? error.message : 'Invalid JSON'
    }

    let extracted = ''
    let extractError: string | null = null
    if (parsed !== null) {
      try {
        extracted = extractByStrategy(parsed, strategy, customPath)
      } catch (error) {
        extractError = error instanceof Error ? error.message : 'Extract failed'
      }
      if (extracted) {
        extractedParts.push(extracted)
      }
    }

    events.push({
      index: index + 1,
      raw: chunk,
      dataPayload,
      parsed,
      parseError,
      isDone: false,
      extracted,
      extractError,
    })
  })

  return {
    events,
    concatenated: extractedParts.join(''),
  }
}

export function formatSseEventsForCopy(events: ParsedSseEvent[]): string {
  return events
    .map((event) => {
      if (event.isDone) {
        return `[${event.index}] data: [DONE]`
      }
      if (event.parseError) {
        return `[${event.index}] ${event.raw}\n(parse error: ${event.parseError})`
      }
      if (event.parsed) {
        return `[${event.index}] data: ${JSON.stringify(event.parsed, null, 2)}`
      }
      return `[${event.index}] ${event.raw}`
    })
    .join('\n\n')
}
