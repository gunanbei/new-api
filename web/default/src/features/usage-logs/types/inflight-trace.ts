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
export type InflightTaskTraceFlags = {
  request_truncated: boolean
  response_truncated: boolean
  response_incomplete: boolean
  in_progress?: boolean
  unsupported_realtime?: boolean
}

export type InflightTraceHTTPPart = {
  method?: string
  path?: string
  query?: string
  protocol?: string
  status_code?: number
  headers?: Record<string, string>
  body?: string
  body_encoding?: 'text' | 'base64' | 'empty'
  body_bytes?: number
  content_type?: string
}

export type InflightTaskTrace = {
  request_id: string
  user_id: number
  status: string
  kind: string
  model_name: string
  is_stream: boolean
  created_at: number
  updated_at: number
  recorded_at: number
  client_request?: InflightTraceHTTPPart
  client_response?: InflightTraceHTTPPart
  flags: InflightTaskTraceFlags
}
