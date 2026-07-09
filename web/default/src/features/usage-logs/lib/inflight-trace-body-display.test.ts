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
  formatTraceJsonForDisplay,
  isHeavyTraceJsonBody,
  shouldUseInstantTraceBodyRender,
} from './inflight-trace-body-display'

describe('inflight-trace-body-display', () => {
  test('detects image payloads as heavy bodies', () => {
    assert.equal(
      isHeavyTraceJsonBody('{"data":[{"b64_json":"abc123"}]}', 'application/json'),
      true
    )
    assert.equal(isHeavyTraceJsonBody('{"ok":true}', 'application/json', 'image'), true)
    assert.equal(isHeavyTraceJsonBody('{"ok":true}', 'application/json'), false)
  })

  test('redacts large base64 fields for formatted preview', () => {
    const formatted = formatTraceJsonForDisplay(
      JSON.stringify({ data: [{ b64_json: 'a'.repeat(200) }] })
    )
    assert.match(formatted, /\[b64_json: /)
    assert.doesNotMatch(formatted, /a{200}/)
  })

  test('uses instant rendering while trace is live', () => {
    assert.equal(shouldUseInstantTraceBodyRender({ liveUpdate: true }), true)
    assert.equal(shouldUseInstantTraceBodyRender({ isSse: true }), true)
    assert.equal(shouldUseInstantTraceBodyRender({}), false)
  })
})
