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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { getRouteApi, useNavigate } from '@tanstack/react-router'
import type { ColumnDef } from '@tanstack/react-table'
import { RefreshCw } from 'lucide-react'
import { useCallback, useMemo, useState, type KeyboardEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  DataTablePage,
  DataTableViewOptions,
  TruncatedCell,
  useDataTable,
} from '@/components/data-table'
import { StatusBadge, type StatusBadgeProps } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useMediaQuery } from '@/hooks'
import { useTableUrlState } from '@/hooks/use-table-url-state'
import { api } from '@/lib/api'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

import { getDefaultTimeRange } from '../lib/utils'
import {
  LogsFilterField,
  LogsFilterInput,
  LogsFilterToolbar,
} from './logs-filter-toolbar'

const route = getRouteApi('/_authenticated/usage-logs/$section')

type InflightTask = {
  request_id: string
  status: string
  kind: string
  model_name: string
  is_stream: boolean
  created_at: number
  updated_at: number
}

type InflightTasksResponse = {
  success: boolean
  message?: string
  data?: {
    items: InflightTask[]
    total: number
    page: number
    page_size: number
  }
}

type InflightFilterDraft = {
  sourceKey: string
  status: string
  kind: string
  stream: string
  model: string
  requestId: string
}

const statusVariant: Record<string, StatusBadgeProps['variant']> = {
  accepted: 'info',
  routing: 'blue',
  upstream_pending: 'warning',
  streaming: 'green',
  completed: 'success',
  failed: 'danger',
}

const statusLabel: Record<string, string> = {
  accepted: 'Accepted',
  routing: 'Routing',
  upstream_pending: 'Waiting upstream',
  streaming: 'Generating',
  completed: 'Completed',
  failed: 'Failed',
}

const kindLabel: Record<string, string> = {
  chat: 'Chat',
  image: 'Image',
  audio: 'Audio',
}

const streamLabel: Record<string, string> = {
  true: 'Yes',
  false: 'No',
}

function formatTime(timestamp: number) {
  if (!timestamp) return '-'
  return dayjs(timestamp * 1000).format('YYYY-MM-DD HH:mm:ss')
}

function buildSourceKey(values: Record<string, unknown>) {
  return [
    values.status,
    values.kind,
    values.stream,
    values.model,
    values.requestId,
  ]
    .map((value) => String(value ?? ''))
    .join('\u001f')
}

function buildParams(props: {
  page: number
  pageSize: number
  searchParams: Record<string, unknown>
}) {
  const { start, end } = getDefaultTimeRange()
  const startTime =
    typeof props.searchParams.startTime === 'number'
      ? props.searchParams.startTime
      : start.getTime()
  const endTime =
    typeof props.searchParams.endTime === 'number'
      ? props.searchParams.endTime
      : end.getTime()
  return {
    p: props.page,
    page_size: props.pageSize,
    status: props.searchParams.status,
    kind: props.searchParams.kind,
    is_stream: props.searchParams.stream,
    model_name: props.searchParams.model,
    request_id: props.searchParams.requestId,
    start_timestamp: Math.floor(startTime / 1000),
    end_timestamp: Math.floor(endTime / 1000),
  }
}

async function fetchInflightTasks(params: Record<string, unknown>) {
  const res = await api.get<InflightTasksResponse>('/api/log/inflight/self', {
    params,
    disableDuplicate: true,
    skipBusinessError: true,
  })
  return res.data
}

function useInflightTaskColumns(): ColumnDef<InflightTask>[] {
  const { t } = useTranslation()

  return useMemo(
    () => [
      {
        accessorKey: 'status',
        header: t('Status'),
        cell: ({ row }) => {
          const status = row.original.status
          return (
            <StatusBadge
              label={t(statusLabel[status] || status)}
              variant={statusVariant[status] || 'neutral'}
              copyable={false}
            />
          )
        },
      },
      {
        accessorKey: 'kind',
        header: t('Type'),
        cell: ({ row }) => t(kindLabel[row.original.kind] || row.original.kind),
      },
      {
        accessorKey: 'model_name',
        header: t('Model'),
        cell: ({ row }) => (
          <TruncatedCell className='max-w-[220px]'>
            {row.original.model_name || '-'}
          </TruncatedCell>
        ),
      },
      {
        accessorKey: 'request_id',
        header: t('Request ID'),
        cell: ({ row }) => (
          <TruncatedCell className='max-w-[240px] font-mono text-xs'>
            {row.original.request_id}
          </TruncatedCell>
        ),
      },
      {
        accessorKey: 'created_at',
        header: t('Started At'),
        cell: ({ row }) => formatTime(row.original.created_at),
      },
      {
        accessorKey: 'updated_at',
        header: t('Updated At'),
        cell: ({ row }) => formatTime(row.original.updated_at),
      },
      {
        accessorKey: 'is_stream',
        header: t('Stream'),
        cell: ({ row }) => t(row.original.is_stream ? 'Yes' : 'No'),
      },
    ],
    [t]
  )
}

function InflightFilterBar<TData>(props: {
  table: ReturnType<typeof useDataTable<TData>>['table']
  isFetching: boolean
  refetch: () => void
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const searchParams = route.useSearch()

  const searchState = useMemo<InflightFilterDraft>(() => {
    const sourceValues = {
      status: searchParams.status,
      kind: searchParams.kind,
      stream: searchParams.stream,
      model: searchParams.model,
      requestId: searchParams.requestId,
    }
    return {
      sourceKey: buildSourceKey(sourceValues),
      status: searchParams.status || '',
      kind: searchParams.kind || '',
      stream: searchParams.stream || '',
      model: searchParams.model || '',
      requestId: searchParams.requestId || '',
    }
  }, [
    searchParams.status,
    searchParams.kind,
    searchParams.stream,
    searchParams.model,
    searchParams.requestId,
  ])
  const [draft, setDraft] = useState<InflightFilterDraft>(() => searchState)
  const activeDraft =
    draft.sourceKey === searchState.sourceKey ? draft : searchState

  const handleChange = useCallback(
    (field: keyof Omit<InflightFilterDraft, 'sourceKey'>, value: string) => {
      setDraft((current) => {
        const base =
          current.sourceKey === searchState.sourceKey ? current : searchState
        return { ...base, sourceKey: searchState.sourceKey, [field]: value }
      })
    },
    [searchState]
  )

  const handleApply = useCallback(() => {
    navigate({
      to: '/usage-logs/$section',
      params: { section: 'inflight' },
      search: {
        status: activeDraft.status || undefined,
        kind: activeDraft.kind || undefined,
        stream: activeDraft.stream || undefined,
        model: activeDraft.model || undefined,
        requestId: activeDraft.requestId || undefined,
        page: 1,
      },
    })
    queryClient.invalidateQueries({ queryKey: ['inflight-tasks'] })
  }, [activeDraft, navigate, queryClient])

  const handleReset = useCallback(() => {
    setDraft({
      sourceKey: '',
      status: '',
      kind: '',
      stream: '',
      model: '',
      requestId: '',
    })
    navigate({
      to: '/usage-logs/$section',
      params: { section: 'inflight' },
      search: { page: 1 },
    })
    queryClient.invalidateQueries({ queryKey: ['inflight-tasks'] })
  }, [navigate, queryClient])

  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (event.key === 'Enter') handleApply()
    },
    [handleApply]
  )

  const hasActiveFilters =
    !!activeDraft.status ||
    !!activeDraft.kind ||
    !!activeDraft.stream ||
    !!activeDraft.model ||
    !!activeDraft.requestId

  const filterFields = (
    <>
      <LogsFilterField>
        <Select
          value={activeDraft.status || 'all'}
          onValueChange={(value) =>
            handleChange('status', value === 'all' ? '' : (value ?? ''))
          }
        >
          <SelectTrigger className='h-8'>
            <SelectValue placeholder={t('All Status')} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value='all'>{t('All Status')}</SelectItem>
            {Object.entries(statusLabel).map(([value, label]) => (
              <SelectItem key={value} value={value}>
                {t(label)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </LogsFilterField>
      <LogsFilterField>
        <Select
          value={activeDraft.kind || 'all'}
          onValueChange={(value) =>
            handleChange('kind', value === 'all' ? '' : (value ?? ''))
          }
        >
          <SelectTrigger className='h-8'>
            <SelectValue placeholder={t('All Types')} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value='all'>{t('All Types')}</SelectItem>
            {Object.entries(kindLabel).map(([value, label]) => (
              <SelectItem key={value} value={value}>
                {t(label)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </LogsFilterField>
      <LogsFilterField>
        <Select
          value={activeDraft.stream || 'all'}
          onValueChange={(value) =>
            handleChange('stream', value === 'all' ? '' : (value ?? ''))
          }
        >
          <SelectTrigger className='h-8'>
            <SelectValue placeholder={t('Stream')} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value='all'>{t('All')}</SelectItem>
            {Object.entries(streamLabel).map(([value, label]) => (
              <SelectItem key={value} value={value}>
                {t(label)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </LogsFilterField>
      <LogsFilterField>
        <LogsFilterInput
          placeholder={t('Model name')}
          value={activeDraft.model}
          onChange={(event) => handleChange('model', event.target.value)}
          onKeyDown={handleKeyDown}
        />
      </LogsFilterField>
      <LogsFilterField wide>
        <LogsFilterInput
          placeholder={t('Request ID')}
          value={activeDraft.requestId}
          onChange={(event) => handleChange('requestId', event.target.value)}
          onKeyDown={handleKeyDown}
        />
      </LogsFilterField>
    </>
  )

  return (
    <LogsFilterToolbar
      table={props.table}
      primaryFilters={filterFields}
      mobilePinnedFilters={filterFields}
      hasActiveFilters={hasActiveFilters}
      onReset={handleReset}
      onSearch={handleApply}
      actionStart={
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                type='button'
                variant='outline'
                size='icon'
                onClick={props.refetch}
                disabled={props.isFetching}
                aria-label={t('Refresh')}
              />
            }
          >
            <RefreshCw
              className={cn('size-4', props.isFetching && 'animate-spin')}
            />
          </TooltipTrigger>
          <TooltipContent>{t('Refresh')}</TooltipContent>
        </Tooltip>
      }
    />
  )
}

export function InflightTasksTab() {
  const { t } = useTranslation()
  const isMobile = useMediaQuery('(max-width: 640px)')
  const searchParams = route.useSearch()
  const {
    columnFilters,
    onColumnFiltersChange,
    pagination,
    onPaginationChange,
    ensurePageInRange,
  } = useTableUrlState({
    search: searchParams,
    navigate: route.useNavigate(),
    pagination: { defaultPage: 1, defaultPageSize: isMobile ? 20 : 100 },
    globalFilter: { enabled: false },
  })
  const columns = useInflightTaskColumns()
  const { data, isLoading, isFetching, refetch } = useQuery({
    queryKey: [
      'inflight-tasks',
      pagination.pageIndex + 1,
      pagination.pageSize,
      searchParams,
    ],
    queryFn: async () => {
      const result = await fetchInflightTasks(
        buildParams({
          page: pagination.pageIndex + 1,
          pageSize: pagination.pageSize,
          searchParams,
        })
      )

      if (!result?.success) {
        toast.error(result?.message || t('Failed to load logs'))
        return { items: [], total: 0, page: 1, page_size: pagination.pageSize }
      }

      return (
        result.data || {
          items: [],
          total: 0,
          page: 1,
          page_size: pagination.pageSize,
        }
      )
    },
    placeholderData: (previousData) => previousData,
    refetchInterval: 5000,
    refetchIntervalInBackground: false,
  })

  const { table } = useDataTable({
    data: data?.items ?? [],
    columns,
    columnFilters,
    pagination,
    enableRowSelection: false,
    onPaginationChange,
    onColumnFiltersChange,
    manualPagination: true,
    manualFiltering: true,
    totalCount: data?.total ?? 0,
    ensurePageInRange,
  })

  return (
    <DataTablePage
      table={table}
      columns={columns}
      isLoading={isLoading}
      isFetching={isFetching}
      emptyTitle={t('No inflight logs.')}
      emptyDescription={t('No inflight logs.')}
      skeletonKeyPrefix='inflight-log-skeleton'
      applyHeaderSize
      toolbar={
        <div className='flex items-center gap-2'>
          <div className='min-w-0 flex-1'>
            <InflightFilterBar
              table={table}
              isFetching={isFetching}
              refetch={() => void refetch()}
            />
          </div>
          <DataTableViewOptions table={table} />
        </div>
      }
      tableClassName='[&_[data-slot=table]]:text-[13px] [&_[data-slot=table]_td]:text-[13px] [&_[data-slot=table]_td_*]:text-[13px] [&_[data-slot=table]_th]:text-[13px] [&_[data-slot=table]_th_*]:text-[13px]'
    />
  )
}
