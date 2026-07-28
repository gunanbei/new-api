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
import {
  ArrowDown,
  ArrowUp,
  ChevronsUpDown,
  CircleAlert,
  CircleX,
  RefreshCw,
} from 'lucide-react'
import { type ReactNode, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { getPerfMetricsGroups } from '@/features/performance-metrics/api'
import {
  formatLatency,
  formatThroughput,
  formatUptimePct,
} from '@/features/performance-metrics/lib/format'
import type { PerfGroupMonitor } from '@/features/performance-metrics/types'
import { UptimeSparkline } from '@/features/pricing/components/model-details-uptime-sparkline'
import { cn } from '@/lib/utils'

const TIME_RANGES = [
  { hours: 6, label: '6h' },
  { hours: 24, label: '24h' },
  { hours: 24 * 7, label: '7d' },
  { hours: 24 * 30, label: '30d' },
] as const

const STATUS_STYLES = {
  available:
    'border-emerald-500/25 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300',
  warning:
    'border-amber-500/25 bg-amber-500/10 text-amber-700 dark:text-amber-300',
  error: 'border-red-500/25 bg-red-500/10 text-red-700 dark:text-red-300',
  unknown: 'border-border bg-muted text-muted-foreground',
} as const

type SortKey =
  | 'ratio'
  | 'cache_hit_rate'
  | 'availability'
  | 'throughput'
  | 'request_count'
type SortDirection = 'asc' | 'desc'
type SortState = {
  key: SortKey
  direction: SortDirection
}

function getSortableGroupValue(group: PerfGroupMonitor, key: SortKey): number {
  switch (key) {
    case 'ratio':
      return Number.isFinite(group.ratio) ? (group.ratio ?? 0) : 0
    case 'cache_hit_rate':
      return Number.isFinite(group.cache_hit_rate) ? group.cache_hit_rate : 0
    case 'availability':
      return Number.isFinite(group.availability) ? group.availability : 0
    case 'throughput':
      return Number.isFinite(group.latest_tps) ? group.latest_tps : 0
    case 'request_count':
      return group.request_count
  }
}

function SortableHeader(props: {
  label: string
  sortKey: SortKey
  activeSortKey: SortKey
  direction: SortDirection
  onSort: (key: SortKey) => void
}) {
  const isActive = props.activeSortKey === props.sortKey
  let SortIcon = ChevronsUpDown
  let ariaSort: 'ascending' | 'descending' | 'none' = 'none'
  if (isActive) {
    SortIcon = props.direction === 'asc' ? ArrowUp : ArrowDown
    ariaSort = props.direction === 'asc' ? 'ascending' : 'descending'
  }

  return (
    <th className='p-0 font-medium' aria-sort={ariaSort}>
      <button
        type='button'
        className={cn(
          'inline-flex w-full items-center gap-1 px-4 py-3 text-left transition-colors hover:text-foreground',
          isActive && 'text-foreground'
        )}
        onClick={() => props.onSort(props.sortKey)}
      >
        {props.label}
        <SortIcon className='size-3.5' aria-hidden='true' />
      </button>
    </th>
  )
}

function StatusBadge(props: { group: PerfGroupMonitor; hours: number }) {
  const { t } = useTranslation()
  const [isPopoverOpen, setIsPopoverOpen] = useState(false)
  const labelKey = {
    available: 'Available',
    warning: 'Warning',
    error: 'Error',
    unknown: 'Unknown',
  }[props.group.status]
  const reasonTexts = (props.group.status_reasons ?? []).map((reason) => {
    if (reason.code === 'latest_bucket_success_rate') {
      return t('Current {{hours}}h success rate is {{rate}}%.', {
        hours: props.hours,
        rate: reason.success_rate.toFixed(2),
      })
    }
    if (reason.code === 'cache_hit_rate_volatility') {
      return t(
        'Cache hit rate fluctuated by over 20% {{count}} times in the last 6 hours. Upstream channels may be switching frequently.',
        { count: reason.fluctuation_count }
      )
    }
    if (reason.code === 'recent_success_rate_warning') {
      return t(
        'Success rate across the latest 20 calls is {{rate}}%. Channel performance is unstable.',
        { rate: reason.success_rate.toFixed(2) }
      )
    }
    if (reason.code === 'channels_disabled') {
      return t(
        'Some channels in this group are disabled. Group availability may fluctuate.'
      )
    }
    if (reason.code === 'recent_success_rate_error') {
      return t(
        'Success rate across the latest 20 calls is only {{rate}}%. Consider switching to another group.',
        { rate: reason.success_rate.toFixed(2) }
      )
    }
    if (reason.code === 'all_channels_disabled') {
      return t(
        'This group is currently unavailable because all of its channels are disabled.'
      )
    }
    if (reason.code === 'insufficient_sampling') {
      return t('Insufficient sampling data.')
    }
    return t(
      'The latest call failed. Check the upstream channel error details.'
    )
  })

  const reasonDetails = reasonTexts.map((reasonText) => (
    <div key={reasonText}>{reasonText}</div>
  ))

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium',
        STATUS_STYLES[props.group.status]
      )}
    >
      <span className='size-1.5 rounded-full bg-current' aria-hidden='true' />
      {t(labelKey)}
      {reasonTexts.length > 0 && (
        <Tooltip open={isPopoverOpen ? false : undefined}>
          <TooltipTrigger render={<span className='inline-flex' />}>
            <Popover open={isPopoverOpen} onOpenChange={setIsPopoverOpen}>
              <PopoverTrigger
                render={
                  <Button
                    variant='ghost'
                    size='icon'
                    className='size-4 text-current hover:bg-transparent hover:text-current'
                    aria-label={reasonTexts.join(' ')}
                  />
                }
              >
                {props.group.status === 'error' ? (
                  <CircleX className='size-3.5' />
                ) : (
                  <CircleAlert className='size-3.5' />
                )}
              </PopoverTrigger>
              <PopoverContent className='max-w-xs whitespace-normal'>
                <div className='space-y-1'>{reasonDetails}</div>
              </PopoverContent>
            </Popover>
          </TooltipTrigger>
          <TooltipContent className='max-w-xs whitespace-normal'>
            <div className='space-y-1'>{reasonDetails}</div>
          </TooltipContent>
        </Tooltip>
      )}
    </span>
  )
}

function GroupTrend(props: { group: PerfGroupMonitor }) {
  const points = useMemo(
    () =>
      props.group.series.slice(-30).map((point) => ({
        date: new Date(point.ts * 1000).toLocaleString(),
        uptime_pct: point.success_rate,
        outage_minutes: 0,
        incidents: 0,
      })),
    [props.group.series]
  )

  return <UptimeSparkline series={points} size='sm' showOverall={false} />
}

function GroupMonitorSkeleton() {
  return (
    <div className='space-y-px'>
      {['first', 'second', 'third'].map((key) => (
        <div key={key} className='grid grid-cols-10 gap-4 px-4 py-4 sm:px-5'>
          {[
            'group',
            'ratio',
            'status',
            'ttft',
            'cache',
            'uptime',
            'tps',
            'requests',
            'trend',
            'latest',
          ].map((cell) => (
            <Skeleton key={cell} className='h-5 w-full' />
          ))}
        </div>
      ))}
    </div>
  )
}

export function GroupMonitor() {
  const { t } = useTranslation()
  const [hours, setHours] = useState(24)
  const [sort, setSort] = useState<SortState>({
    key: 'availability',
    direction: 'desc',
  })
  const groupsQuery = useQuery({
    queryKey: ['perf-metrics-groups', hours],
    queryFn: () => getPerfMetricsGroups(hours),
    staleTime: 30 * 1000,
    refetchInterval: 60 * 1000,
    retry: false,
  })
  const groups = useMemo(
    () =>
      [...(groupsQuery.data?.data ?? [])].sort((left, right) => {
        const leftValue = getSortableGroupValue(left, sort.key)
        const rightValue = getSortableGroupValue(right, sort.key)
        const difference = leftValue - rightValue
        return sort.direction === 'asc' ? difference : -difference
      }),
    [groupsQuery.data, sort]
  )
  const handleSort = (key: SortKey) => {
    setSort((current) => {
      if (current.key !== key) return { key, direction: 'desc' }
      return {
        ...current,
        direction: current.direction === 'asc' ? 'desc' : 'asc',
      }
    })
  }
  let rows: ReactNode
  if (groupsQuery.isLoading) {
    rows = (
      <tr>
        <td colSpan={10}>
          <GroupMonitorSkeleton />
        </td>
      </tr>
    )
  } else if (groups.length === 0) {
    rows = (
      <tr>
        <td
          colSpan={10}
          className='text-muted-foreground px-4 py-12 text-center'
        >
          {t('No group monitoring data')}
        </td>
      </tr>
    )
  } else {
    rows = groups.map((group) => (
      <tr key={group.group} className='hover:bg-muted/30'>
        <td className='px-4 py-3.5 font-mono font-medium sm:px-5'>
          {group.group}
        </td>
        <td className='px-4 py-3.5 font-mono tabular-nums'>
          {group.ratio?.toFixed(2) ?? '—'}
        </td>
        <td className='px-4 py-3.5'>
          <StatusBadge group={group} hours={hours} />
        </td>
        <td className='px-4 py-3.5 font-mono tabular-nums'>
          {formatLatency(group.latest_ttft_ms)}
        </td>
        <td className='px-4 py-3.5 font-mono tabular-nums'>
          {group.input_tokens > 0 ? formatUptimePct(group.cache_hit_rate) : '—'}
        </td>
        <td className='px-4 py-3.5 font-mono tabular-nums'>
          {group.request_count > 0 ? formatUptimePct(group.availability) : '—'}
        </td>
        <td className='px-4 py-3.5 font-mono tabular-nums'>
          {formatThroughput(group.latest_tps)}
        </td>
        <td className='px-4 py-3.5 font-mono tabular-nums'>
          {group.request_count.toLocaleString()}
        </td>
        <td className='px-4 py-3.5'>
          <GroupTrend group={group} />
        </td>
        <td className='text-muted-foreground px-4 py-3.5 font-mono text-xs tabular-nums'>
          {group.last_updated > 0
            ? new Date(group.last_updated * 1000).toLocaleString()
            : '—'}
        </td>
      </tr>
    ))
  }

  return (
    <div className='bg-card overflow-hidden rounded-lg border'>
      <div className='flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3 sm:px-5'>
        <Tabs
          value={String(hours)}
          onValueChange={(value) => setHours(Number(value))}
        >
          <TabsList>
            {TIME_RANGES.map((range) => (
              <TabsTrigger key={range.hours} value={String(range.hours)}>
                {range.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
        <Button
          variant='ghost'
          size='icon'
          className='size-8'
          onClick={() => void groupsQuery.refetch()}
          disabled={groupsQuery.isFetching}
          aria-label={t('Refresh')}
        >
          <RefreshCw
            className={cn('size-4', groupsQuery.isFetching && 'animate-spin')}
          />
        </Button>
      </div>

      <div className='max-h-[34rem] overflow-auto'>
        <table className='w-full min-w-[1120px] text-left text-sm'>
          <thead className='bg-muted/40 text-muted-foreground sticky top-0 z-10 text-xs'>
            <tr className='border-b'>
              <th className='px-4 py-3 font-medium sm:px-5'>{t('Group')}</th>
              <SortableHeader
                label={t('Ratio')}
                sortKey='ratio'
                activeSortKey={sort.key}
                direction={sort.direction}
                onSort={handleSort}
              />
              <th className='px-4 py-3 font-medium'>{t('Status')}</th>
              <th className='px-4 py-3 font-medium'>
                {t('Latest first token')}
              </th>
              <SortableHeader
                label={t('Cache hit rate')}
                sortKey='cache_hit_rate'
                activeSortKey={sort.key}
                direction={sort.direction}
                onSort={handleSort}
              />
              <SortableHeader
                label={t('Availability')}
                sortKey='availability'
                activeSortKey={sort.key}
                direction={sort.direction}
                onSort={handleSort}
              />
              <SortableHeader
                label={t('Throughput')}
                sortKey='throughput'
                activeSortKey={sort.key}
                direction={sort.direction}
                onSort={handleSort}
              />
              <SortableHeader
                label={t('Requests')}
                sortKey='request_count'
                activeSortKey={sort.key}
                direction={sort.direction}
                onSort={handleSort}
              />
              <th className='px-4 py-3 font-medium'>{t('Monitoring trend')}</th>
              <th className='px-4 py-3 font-medium'>
                {t('Latest monitoring')}
              </th>
            </tr>
          </thead>
          <tbody className='divide-y'>{rows}</tbody>
        </table>
      </div>
    </div>
  )
}
