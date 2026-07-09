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
import { useQuery } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

import { getInflightCleanupSchedule } from '../api'

export function InflightCleanupScheduleInfo(props: { className?: string }) {
  const { t } = useTranslation()
  const { data, isLoading } = useQuery({
    queryKey: ['inflight-cleanup-schedule'],
    queryFn: async () => {
      const res = await getInflightCleanupSchedule()
      if (!res.success || !res.data) {
        return null
      }
      return res.data
    },
    staleTime: 60_000,
    refetchInterval: 60_000,
  })

  if (isLoading) {
    return (
      <div
        className={cn(
          'text-muted-foreground flex items-center gap-1.5 text-xs',
          props.className
        )}
      >
        <Loader2 className='size-3.5 animate-spin' />
        <span>{t('Loading cleanup schedule...')}</span>
      </div>
    )
  }

  if (!data) {
    return null
  }

  const ruleLabel = !data.enabled
    ? t('Automatic inflight log cleanup is disabled')
    : data.schedule_mode === 'cron'
      ? t('Clean terminal inflight logs on schedule: {{expr}}', {
          expr: data.cron_expression,
        })
      : t('Clean terminal inflight logs every {{minutes}} minutes', {
          minutes: data.interval_minutes,
        })

  const nextLabel = data.running
    ? t('Cleanup in progress')
    : data.enabled && data.next_run_at > 0
      ? t('Next cleanup: {{time}}', {
          time: formatTimestampToDate(data.next_run_at, 'seconds'),
        })
      : null

  return (
    <div
      className={cn(
        'bg-muted/20 border-border text-muted-foreground flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5 rounded-lg border px-2.5 py-1 text-xs sm:text-sm',
        props.className
      )}
    >
      <span className='min-w-0'>{ruleLabel}</span>
      {nextLabel ? (
        <>
          <span aria-hidden className='text-border hidden sm:inline'>
            |
          </span>
          <span className='text-foreground/80 min-w-0 font-medium'>
            {nextLabel}
          </span>
        </>
      ) : null}
    </div>
  )
}
