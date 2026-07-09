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
export type InflightCronPreset = 'daily' | 'weekly' | 'hourly_interval'

export type InflightCronSimpleConfig = {
  preset: InflightCronPreset
  hour: number
  minute: number
  weekday: number
  hourInterval: number
}

export const INFLIGHT_CLEANUP_INTERVAL_MIN = 1
export const INFLIGHT_CLEANUP_INTERVAL_MAX = 10080

export const DEFAULT_INFLIGHT_CRON_SIMPLE_CONFIG: InflightCronSimpleConfig = {
  preset: 'daily',
  hour: 3,
  minute: 0,
  weekday: 1,
  hourInterval: 6,
}

export function buildCronFromSimple(config: InflightCronSimpleConfig) {
  switch (config.preset) {
    case 'daily':
      return `${config.minute} ${config.hour} * * *`
    case 'weekly':
      return `${config.minute} ${config.hour} * * ${config.weekday}`
    case 'hourly_interval':
      return `0 */${config.hourInterval} * * *`
    default:
      return '0 3 * * *'
  }
}

export function parseCronToSimple(
  expr: string
): InflightCronSimpleConfig | null {
  const normalized = expr.trim().replace(/\s+/g, ' ')
  const daily = /^(\d{1,2}) (\d{1,2}) \* \* \*$/.exec(normalized)
  if (daily) {
    return {
      preset: 'daily',
      minute: Number(daily[1]),
      hour: Number(daily[2]),
      weekday: 1,
      hourInterval: 6,
    }
  }
  const weekly = /^(\d{1,2}) (\d{1,2}) \* \* (\d{1,2})$/.exec(normalized)
  if (weekly) {
    return {
      preset: 'weekly',
      minute: Number(weekly[1]),
      hour: Number(weekly[2]),
      weekday: Number(weekly[3]),
      hourInterval: 6,
    }
  }
  const hourly = /^0 \*\/(\d{1,2}) \* \* \*$/.exec(normalized)
  if (hourly) {
    return {
      preset: 'hourly_interval',
      minute: 0,
      hour: 0,
      weekday: 1,
      hourInterval: Number(hourly[1]),
    }
  }
  return null
}

export function normalizeInflightCronSimpleConfig(
  config: InflightCronSimpleConfig
): InflightCronSimpleConfig {
  return {
    preset: config.preset,
    hour: Math.min(23, Math.max(0, config.hour)),
    minute: Math.min(59, Math.max(0, config.minute)),
    weekday: Math.min(6, Math.max(0, config.weekday)),
    hourInterval: Math.min(23, Math.max(1, config.hourInterval)),
  }
}
