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
export type GroupTrendMetric = 'ttft' | 'availability' | 'tps'

export const GROUP_TREND_METRIC_OPTIONS: Array<{
  value: GroupTrendMetric
  labelKey: string
}> = [
  { value: 'ttft', labelKey: 'First token latency' },
  { value: 'availability', labelKey: 'Availability' },
  { value: 'tps', labelKey: 'Throughput short' },
]

export function getRelativeTrendColor(value: number, average: number): string {
  if (!Number.isFinite(average) || average <= 0) {
    return '#f59e0b'
  }
  if (value <= average) return '#10b981'
  if (value <= average * 1.2) return '#f59e0b'
  return '#ef4444'
}
