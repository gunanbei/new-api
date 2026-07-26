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

import {
  DEFAULT_GROUP,
  FAILOVER_STRATEGIES,
  MAX_FAILOVER_GROUPS,
  MIN_FAILOVER_GROUPS,
} from '../constants'
import { type ApiKeyFormData, type ApiKey } from '../types'

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
  tokenCount: 1,
}

export function getApiKeyFormDefaultValues(
  defaultUseAutoGroup: boolean
): ApiKeyFormValues {
  return {
    ...API_KEY_FORM_DEFAULT_VALUES,
    group: defaultUseAutoGroup ? 'auto' : DEFAULT_GROUP,
    cross_group_retry: defaultUseAutoGroup,
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
  }
}

/**
 * Transform API key data to form defaults
 */
export function transformApiKeyToFormDefaults(
  apiKey: ApiKey
): ApiKeyFormValues {
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
    tokenCount: 1,
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
