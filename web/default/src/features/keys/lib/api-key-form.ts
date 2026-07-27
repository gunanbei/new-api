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
import type { TFunction } from 'i18next'
import { z } from 'zod'

import { parseQuotaFromDollars, quotaUnitsToDollars } from '@/lib/format'
import { parseHttpStatusCodeRules } from '@/lib/http-status-code-rules'

import {
  DEFAULT_GROUP,
  FAILOVER_STRATEGIES,
  MAX_FAILOVER_GROUPS,
  MIN_FAILOVER_GROUPS,
} from '../constants'
import type { ApiKey, ApiKeyFormData } from '../types'

// ============================================================================
// Form Schema
// ============================================================================

/**
 * @param maxRetryTimes System wide retry cap from /api/status. A failover key may lower
 *   it but never raise it, and the backend rejects anything above it.
 */
export function getApiKeyFormSchema(t: TFunction, maxRetryTimes: number) {
  return z
    .object({
      name: z.string().min(1, t('Please enter a name')),
      remain_quota_dollars: z.number().optional(),
      expired_time: z.date().optional(),
      unlimited_quota: z.boolean(),
      model_limits: z.array(z.string()),
      allow_ips: z.string().optional(),
      group: z.string().optional(),
      cross_group_retry: z.boolean().optional(),
      failover_enabled: z.boolean().optional(),
      failover_groups: z.array(z.string()),
      failover_strategy: z.enum(FAILOVER_STRATEGIES),
      failover_max_retry: z.number().optional(),
      failover_http_status_codes: z.string(),
      stream_response_header_timeout_seconds: z.number(),
      stream_first_content_timeout_seconds: z.number(),
      non_stream_response_header_timeout_seconds: z.number(),
      non_stream_first_content_timeout_seconds: z.number(),
      retry_on_transport_error: z.boolean(),
      retry_on_empty_response: z.boolean(),
      retry_on_invalid_response: z.boolean(),
      retry_on_stream_error: z.boolean(),
      tokenCount: z.number().min(1).optional(),
    })
    .superRefine((data, ctx) => {
      if (
        !data.unlimited_quota &&
        (data.remain_quota_dollars === undefined ||
          data.remain_quota_dollars < 0)
      ) {
        ctx.addIssue({
          code: 'custom',
          path: ['remain_quota_dollars'],
          message: t('Quota must be zero or greater'),
        })
      }

      if (!data.failover_enabled) {
        return
      }

      const statusCodes = parseHttpStatusCodeRules(
        data.failover_http_status_codes
      )
      if (!statusCodes.ok) {
        ctx.addIssue({
          code: 'custom',
          path: ['failover_http_status_codes'],
          message: t('Invalid status code rules: {{rules}}', {
            rules: statusCodes.invalidTokens.join(', '),
          }),
        })
      }

      if (
        data.failover_groups.length < MIN_FAILOVER_GROUPS ||
        data.failover_groups.length > MAX_FAILOVER_GROUPS
      ) {
        ctx.addIssue({
          code: 'custom',
          path: ['failover_groups'],
          message: t('Select between {{min}} and {{max}} groups', {
            min: MIN_FAILOVER_GROUPS,
            max: MAX_FAILOVER_GROUPS,
          }),
        })
      }

      if (maxRetryTimes < 1) {
        ctx.addIssue({
          code: 'custom',
          path: ['failover_enabled'],
          message: t(
            'Retries are disabled system wide, so failover cannot take effect. Ask an administrator to raise the max retry count.'
          ),
        })
        return
      }

      const maxRetry = data.failover_max_retry
      if (maxRetry === undefined || maxRetry < 1 || maxRetry > maxRetryTimes) {
        ctx.addIssue({
          code: 'custom',
          path: ['failover_max_retry'],
          message: t('Enter a value between 1 and {{max}}', {
            max: maxRetryTimes,
          }),
        })
      }

      for (const [path, value, max] of [
        [
          'stream_response_header_timeout_seconds',
          data.stream_response_header_timeout_seconds,
          120,
        ],
        [
          'stream_first_content_timeout_seconds',
          data.stream_first_content_timeout_seconds,
          300,
        ],
        [
          'non_stream_response_header_timeout_seconds',
          data.non_stream_response_header_timeout_seconds,
          120,
        ],
        [
          'non_stream_first_content_timeout_seconds',
          data.non_stream_first_content_timeout_seconds,
          300,
        ],
      ] as const) {
        if (value !== 0 && (value < 1 || value > max)) {
          ctx.addIssue({
            code: 'custom',
            path: [path],
            message: t(
              'Enter 0 to disable, or a value between 1 and {{max}} seconds',
              { max }
            ),
          })
        }
      }
    })
}

export type ApiKeyFormValues = z.infer<ReturnType<typeof getApiKeyFormSchema>>

// ============================================================================
// Form Defaults
// ============================================================================

export const API_KEY_FORM_DEFAULT_VALUES: ApiKeyFormValues = {
  name: '',
  remain_quota_dollars: 10,
  expired_time: undefined,
  unlimited_quota: true,
  model_limits: [],
  allow_ips: '',
  group: DEFAULT_GROUP,
  cross_group_retry: true,
  failover_enabled: false,
  failover_groups: [],
  failover_strategy: 'order',
  failover_max_retry: 1,
  failover_http_status_codes: '',
  stream_response_header_timeout_seconds: 15,
  stream_first_content_timeout_seconds: 45,
  non_stream_response_header_timeout_seconds: 15,
  non_stream_first_content_timeout_seconds: 45,
  retry_on_transport_error: true,
  retry_on_empty_response: true,
  retry_on_invalid_response: true,
  retry_on_stream_error: true,
  tokenCount: 1,
}

export function getApiKeyFormDefaultValues(
  defaultUseAutoGroup: boolean,
  automaticRetryStatusCodes = ''
): ApiKeyFormValues {
  return {
    ...API_KEY_FORM_DEFAULT_VALUES,
    group: defaultUseAutoGroup ? 'auto' : DEFAULT_GROUP,
    cross_group_retry: defaultUseAutoGroup,
    failover_http_status_codes: automaticRetryStatusCodes,
  }
}

// ============================================================================
// Form Data Transformation
// ============================================================================

/**
 * Transform form data to API payload
 */
export function transformFormDataToPayload(
  data: ApiKeyFormValues
): ApiKeyFormData {
  const failoverEnabled = !!data.failover_enabled
  return {
    name: data.name,
    remain_quota: data.unlimited_quota
      ? 0
      : parseQuotaFromDollars(data.remain_quota_dollars || 0),
    expired_time: data.expired_time
      ? Math.floor(data.expired_time.getTime() / 1000)
      : -1,
    unlimited_quota: data.unlimited_quota,
    model_limits_enabled: data.model_limits.length > 0,
    model_limits: data.model_limits.join(','),
    allow_ips: data.allow_ips || '',
    // The group stays on the payload while failover is on so turning it back off
    // restores the single group the user had picked before.
    group: data.group || '',
    cross_group_retry:
      !failoverEnabled && data.group === 'auto' && !!data.cross_group_retry,
    failover_enabled: failoverEnabled,
    failover_groups: failoverEnabled
      ? JSON.stringify(data.failover_groups)
      : '',
    failover_strategy: failoverEnabled ? data.failover_strategy : '',
    failover_max_retry: failoverEnabled ? data.failover_max_retry || 1 : 0,
    failover_rules: failoverEnabled
      ? JSON.stringify({
          http_status_codes: parseHttpStatusCodeRules(
            data.failover_http_status_codes
          ).normalized,
          stream_response_header_timeout_ms:
            data.stream_response_header_timeout_seconds * 1000,
          stream_first_content_timeout_ms:
            data.stream_first_content_timeout_seconds * 1000,
          non_stream_response_header_timeout_ms:
            data.non_stream_response_header_timeout_seconds * 1000,
          non_stream_first_content_timeout_ms:
            data.non_stream_first_content_timeout_seconds * 1000,
          retry_on_transport_error: data.retry_on_transport_error,
          retry_on_empty_response: data.retry_on_empty_response,
          retry_on_invalid_response: data.retry_on_invalid_response,
          retry_on_stream_error: data.retry_on_stream_error,
        })
      : '',
  }
}

/**
 * Transform API key data to form defaults
 */
export function transformApiKeyToFormDefaults(
  apiKey: ApiKey
): ApiKeyFormValues {
  const rules = parseFailoverRules(apiKey.failover_rules)
  return {
    name: apiKey.name,
    remain_quota_dollars: apiKey.unlimited_quota
      ? 0
      : quotaUnitsToDollars(apiKey.remain_quota),
    expired_time:
      apiKey.expired_time > 0
        ? new Date(apiKey.expired_time * 1000)
        : undefined,
    unlimited_quota: apiKey.unlimited_quota,
    model_limits: apiKey.model_limits
      ? apiKey.model_limits.split(',').filter(Boolean)
      : [],
    allow_ips: apiKey.allow_ips || '',
    group: apiKey.group || DEFAULT_GROUP,
    cross_group_retry: !!apiKey.cross_group_retry,
    failover_enabled: !!apiKey.failover_enabled,
    failover_groups: parseFailoverGroups(apiKey.failover_groups),
    failover_strategy: isFailoverStrategy(apiKey.failover_strategy)
      ? apiKey.failover_strategy
      : 'order',
    failover_max_retry: apiKey.failover_max_retry || 1,
    failover_http_status_codes: rules.http_status_codes,
    stream_response_header_timeout_seconds:
      rules.stream_response_header_timeout_ms / 1000,
    stream_first_content_timeout_seconds:
      rules.stream_first_content_timeout_ms / 1000,
    non_stream_response_header_timeout_seconds:
      rules.non_stream_response_header_timeout_ms / 1000,
    non_stream_first_content_timeout_seconds:
      rules.non_stream_first_content_timeout_ms / 1000,
    retry_on_transport_error: rules.retry_on_transport_error,
    retry_on_empty_response: rules.retry_on_empty_response,
    retry_on_invalid_response: rules.retry_on_invalid_response,
    retry_on_stream_error: rules.retry_on_stream_error,
    tokenCount: 1,
  }
}

type FailoverRules = {
  http_status_codes: string
  stream_response_header_timeout_ms: number
  stream_first_content_timeout_ms: number
  non_stream_response_header_timeout_ms: number
  non_stream_first_content_timeout_ms: number
  retry_on_transport_error: boolean
  retry_on_empty_response: boolean
  retry_on_invalid_response: boolean
  retry_on_stream_error: boolean
}

const DEFAULT_FAILOVER_RULES: FailoverRules = {
  http_status_codes: '',
  stream_response_header_timeout_ms: 15000,
  stream_first_content_timeout_ms: 45000,
  non_stream_response_header_timeout_ms: 15000,
  non_stream_first_content_timeout_ms: 45000,
  retry_on_transport_error: true,
  retry_on_empty_response: true,
  retry_on_invalid_response: true,
  retry_on_stream_error: true,
}

function parseFailoverRules(raw: string | null | undefined): FailoverRules {
  if (!raw) return DEFAULT_FAILOVER_RULES
  try {
    const parsed = JSON.parse(raw) as Partial<FailoverRules> & {
      response_header_timeout_ms?: number
      first_content_timeout_ms?: number
    }
    const legacyHeader = parsed.response_header_timeout_ms ?? 15000
    const legacyContent = parsed.first_content_timeout_ms ?? 45000
    return {
      ...DEFAULT_FAILOVER_RULES,
      ...parsed,
      stream_response_header_timeout_ms:
        parsed.stream_response_header_timeout_ms ?? legacyHeader,
      stream_first_content_timeout_ms:
        parsed.stream_first_content_timeout_ms ?? legacyContent,
      non_stream_response_header_timeout_ms:
        parsed.non_stream_response_header_timeout_ms ?? legacyHeader,
      non_stream_first_content_timeout_ms:
        parsed.non_stream_first_content_timeout_ms ?? legacyContent,
    }
  } catch {
    return DEFAULT_FAILOVER_RULES
  }
}

/** Reads the JSON encoded group sequence stored on an API key. */
export function parseFailoverGroups(raw: string | null | undefined): string[] {
  if (!raw) return []
  try {
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((item): item is string => typeof item === 'string')
  } catch {
    return []
  }
}

function isFailoverStrategy(
  value: string | null | undefined
): value is (typeof FAILOVER_STRATEGIES)[number] {
  return (
    !!value &&
    FAILOVER_STRATEGIES.includes(value as (typeof FAILOVER_STRATEGIES)[number])
  )
}
