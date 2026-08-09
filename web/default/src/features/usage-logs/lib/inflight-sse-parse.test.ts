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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  appendSseTraceText,
  createIncrementalSseParserState,
  formatSseEventsForCopy,
  parseSseTrace,
} from './inflight-sse-parse'

describe('parseSseTrace', () => {
  test('extracts OpenAI chat completion deltas', () => {
    const body = [
      'data: {"choices":[{"delta":{"content":"Hi"}}]}',
      '',
      'data: {"choices":[{"delta":{"content":" there"}}]}',
      '',
      'data: [DONE]',
      '',
    ].join('\n')

    const result = parseSseTrace(body, 'openai', '')
    assert.equal(result.concatenated, 'Hi there')
    assert.equal(result.events.length, 3)
    assert.equal(result.events[0]?.extracted, 'Hi')
  })

  test('summarizes image stream payloads without dumping base64', () => {
    const body = [
      `data: {"type":"image_generation.partial_image","b64_json":"${'a'.repeat(120)}"}`,
      '',
    ].join('\n')

    const result = parseSseTrace(body, 'openai', '')
    assert.match(result.concatenated, /\[image payload:/)
    assert.doesNotMatch(result.concatenated, /a{120}/)
  })

  test('extracts OpenAI Responses API output text deltas', () => {
    const body = [
      'data: {"type":"response.output_text.delta","delta":"hello"}',
      '',
      'data: {"type":"response.output_text.delta","delta":" world"}',
      '',
    ].join('\n')

    const result = parseSseTrace(body, 'openai', '')
    assert.equal(result.concatenated, 'hello world')
  })

  test('extracts with custom JSONPath', () => {
    const body = 'data: {"message":{"content":"custom"}}\n\n'
    const result = parseSseTrace(body, 'custom', '$.message.content')
    assert.equal(result.concatenated, 'custom')
  })

  test('formats events for copy', () => {
    const body = 'data: {"id":1}\n\n'
    const parsed = parseSseTrace(body, 'openai', '')
    const copied = formatSseEventsForCopy(parsed.events)
    assert.match(copied, /\[1\] data:/)
    assert.match(copied, /"id": 1/)
  })

  test('incrementally appends parsed SSE output', () => {
    const state = createIncrementalSseParserState()
    const first = appendSseTraceText(
      state,
      'data: {"choices":[{"delta":{"content":"Hi"}}]}\n\n',
      'openai',
      ''
    )
    const second = appendSseTraceText(
      state,
      'data: {"choices":[{"delta":{"content":"Hi"}}]}\n\ndata: {"choices":[{"delta":{"content":" there"}}]}\n\n',
      'openai',
      ''
    )
    assert.equal(first.concatenated, 'Hi')
    assert.equal(second.concatenated, 'Hi there')
    assert.equal(second.events.length, 2)
  })

  test('reparses the current prefix when strategy changes', () => {
    const state = createIncrementalSseParserState()
    const body = 'data: {"message":{"content":"custom"}}\n\n'

    appendSseTraceText(state, body, 'openai', '')
    const result = appendSseTraceText(
      state,
      body,
      'custom',
      '$.message.content'
    )

    assert.equal(result.concatenated, 'custom')
    assert.equal(result.events.length, 1)
  })

  test('parses a complete event before its terminating blank line', () => {
    const state = createIncrementalSseParserState()
    const result = appendSseTraceText(
      state,
      'data: {"choices":[{"delta":{"content":"live"}}]}',
      'auto',
      ''
    )

    assert.equal(result.concatenated, 'live')
    assert.equal(result.events.length, 1)
    assert.equal(result.events[0]?.parseError, null)
  })

  test('keeps incomplete JSON buffered until the next snapshot', () => {
    const state = createIncrementalSseParserState()
    const first = appendSseTraceText(state, 'data: {"choices":[', 'auto', '')
    const second = appendSseTraceText(
      state,
      'data: {"choices":[{"delta":{"content":"ok"}}]}\n\n',
      'auto',
      ''
    )

    assert.equal(first.events.length, 0)
    assert.equal(second.concatenated, 'ok')
    assert.equal(second.events.length, 1)
  })

  test('auto-detects provider payloads and normalizes CRLF', () => {
    const cases = [
      [
        'data: {"candidates":[{"content":{"parts":[{"text":"gemini"}]}}]}\r\n\r\n',
        'gemini',
      ],
      ['data: {"delta":{"text":"claude"}}\r\n\r\n', 'claude'],
      ['data: {"response":"generate"}\r\n\r\n', 'generate'],
      ['data: {"message":{"content":"chat"}}\r\n\r\n', 'chat'],
    ] as const

    for (const [body, expected] of cases) {
      assert.equal(parseSseTrace(body, 'auto', '').concatenated, expected)
    }
  })
})
