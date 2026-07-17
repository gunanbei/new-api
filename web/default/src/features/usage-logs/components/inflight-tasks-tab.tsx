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
import type { ColumnDef, Row } from '@tanstack/react-table'
import {
  Eye,
  FolderOpen,
  GitBranch,
  RefreshCw,
  ScrollText,
  Trash2,
} from 'lucide-react'
import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type KeyboardEvent,
} from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  DataTablePage,
  DataTableRow,
  DataTableRowActionMenu,
  TruncatedCell,
  useDataTable,
} from '@/components/data-table'
import { Dialog } from '@/components/dialog'
import { StatusBadge, type StatusBadgeProps } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from '@/components/ui/context-menu'
import { DropdownMenuItem } from '@/components/ui/dropdown-menu'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useMediaQuery } from '@/hooks'
import { useIsAdmin } from '@/hooks/use-admin'
import { useTableUrlState } from '@/hooks/use-table-url-state'
import { api } from '@/lib/api'
import dayjs from '@/lib/dayjs'
import { cn, tryPrettyJson } from '@/lib/utils'

import {
  deleteInflightTraceLocalCacheFiles,
  listInflightTraceLocalCacheFiles,
  readInflightTraceLocalCacheRows,
  type InflightTraceLocalCacheFile,
} from '../lib/inflight-trace-local-cache'
import { getDefaultTimeRange } from '../lib/utils'
import { CompactDateTimeRangePicker } from './compact-date-time-range-picker'
import {
  InflightDetailRow,
  InflightDetailSection,
} from './inflight-detail-primitives'
import { InflightTaskTraceDialog } from './inflight-task-trace-dialog'
import {
  LogsFilterField,
  LogsFilterInput,
  LogsFilterToolbar,
} from './logs-filter-toolbar'

const route = getRouteApi('/_authenticated/usage-logs/$section')
const inflightColumnVisibilityStorageKey =
  'usage-logs:inflight:column-visibility'
const inflightListLiveRefreshMs = 5000

type InflightTaskStatusStep = {
  status: string
  started_at: number
  updated_at: number
  duration_seconds: number
}

type InflightTaskChannelAttempt = {
  retry_index: number
  channel_id?: number
  channel_name?: string
  status?: string
  error?: string
  started_at?: number
  updated_at?: number
}

type InflightTaskDetail = {
  channel_id?: number
  channel_name?: string
  retry_index?: number
  latest_error?: string
  current_stage?: string
  channel_chain?: InflightTaskChannelAttempt[]
  attempts?: InflightTaskAttempt[]
  timeline?: InflightTaskStatusStep[]
}

type InflightTaskAttempt = {
  retry_index: number
  channel_id?: number
  channel_name?: string
  status?: string
  error?: string
  started_at?: number
  updated_at?: number
  timeline?: InflightTaskStatusStep[]
}

type InflightTask = {
  request_id: string
  status: string
  kind: string
  model_name: string
  is_stream: boolean
  created_at: number
  updated_at: number
  has_trace?: boolean
  detail?: InflightTaskDetail
}

type InflightTasksResponse = {
  success: boolean
  message?: string
  data?: {
    items: InflightTask[]
    total: number
    page: number
    page_size: number
    meta?: {
      trace_menu_visible: boolean
    }
  }
}

const MOCK_INFLIGHT_TASKS: InflightTask[] = [
  {
    request_id: 'req_inflight_mock_001',
    status: 'streaming',
    kind: 'chat',
    model_name: 'gpt-4.1',
    is_stream: true,
    created_at: 1751905200,
    updated_at: 1751905228,
    detail: {
      channel_id: 12,
      channel_name: 'OpenAI Primary',
      retry_index: 1,
      current_stage: 'streaming',
      latest_error: 'first attempt upstream timeout',
      timeline: [
        {
          status: 'accepted',
          started_at: 1751905200,
          updated_at: 1751905202,
          duration_seconds: 2,
        },
        {
          status: 'routing',
          started_at: 1751905202,
          updated_at: 1751905205,
          duration_seconds: 3,
        },
        {
          status: 'upstream_pending',
          started_at: 1751905205,
          updated_at: 1751905214,
          duration_seconds: 9,
        },
        {
          status: 'streaming',
          started_at: 1751905214,
          updated_at: 1751905228,
          duration_seconds: 14,
        },
      ],
      channel_chain: [
        {
          retry_index: 0,
          channel_id: 8,
          channel_name: 'Azure Fallback',
          status: 'upstream_pending',
          error: 'upstream timeout after 8s',
          started_at: 1751905205,
          updated_at: 1751905213,
        },
        {
          retry_index: 1,
          channel_id: 12,
          channel_name: 'OpenAI Primary',
          status: 'streaming',
          started_at: 1751905214,
          updated_at: 1751905228,
        },
      ],
      attempts: [
        {
          retry_index: 0,
          channel_id: 8,
          channel_name: 'Azure Fallback',
          status: 'failed',
          error: 'upstream timeout after 8s',
          started_at: 1751905202,
          updated_at: 1751905213,
          timeline: [
            {
              status: 'routing',
              started_at: 1751905202,
              updated_at: 1751905205,
              duration_seconds: 3,
            },
            {
              status: 'upstream_pending',
              started_at: 1751905205,
              updated_at: 1751905213,
              duration_seconds: 8,
            },
            {
              status: 'failed',
              started_at: 1751905213,
              updated_at: 1751905213,
              duration_seconds: 0,
            },
          ],
        },
        {
          retry_index: 1,
          channel_id: 12,
          channel_name: 'OpenAI Primary',
          status: 'streaming',
          started_at: 1751905214,
          updated_at: 1751905228,
          timeline: [
            {
              status: 'routing',
              started_at: 1751905214,
              updated_at: 1751905214,
              duration_seconds: 0,
            },
            {
              status: 'upstream_pending',
              started_at: 1751905214,
              updated_at: 1751905217,
              duration_seconds: 3,
            },
            {
              status: 'streaming',
              started_at: 1751905217,
              updated_at: 1751905228,
              duration_seconds: 11,
            },
          ],
        },
      ],
    },
  },
  {
    request_id: 'req_inflight_mock_002',
    status: 'failed',
    kind: 'image',
    model_name: 'gpt-image-1',
    is_stream: false,
    has_trace: true,
    created_at: 1751905000,
    updated_at: 1751905016,
    detail: {
      channel_id: 21,
      channel_name: 'Image Node',
      retry_index: 0,
      current_stage: 'failed',
      latest_error: 'provider rejected image size',
      timeline: [
        {
          status: 'accepted',
          started_at: 1751905000,
          updated_at: 1751905001,
          duration_seconds: 1,
        },
        {
          status: 'routing',
          started_at: 1751905001,
          updated_at: 1751905003,
          duration_seconds: 2,
        },
        {
          status: 'upstream_pending',
          started_at: 1751905003,
          updated_at: 1751905016,
          duration_seconds: 13,
        },
        {
          status: 'failed',
          started_at: 1751905016,
          updated_at: 1751905016,
          duration_seconds: 0,
        },
      ],
      channel_chain: [
        {
          retry_index: 0,
          channel_id: 21,
          channel_name: 'Image Node',
          status: 'failed',
          error: 'provider rejected image size',
          started_at: 1751905003,
          updated_at: 1751905016,
        },
      ],
      attempts: [
        {
          retry_index: 0,
          channel_id: 21,
          channel_name: 'Image Node',
          status: 'failed',
          error: 'provider rejected image size',
          started_at: 1751905001,
          updated_at: 1751905016,
          timeline: [
            {
              status: 'routing',
              started_at: 1751905001,
              updated_at: 1751905003,
              duration_seconds: 2,
            },
            {
              status: 'upstream_pending',
              started_at: 1751905003,
              updated_at: 1751905016,
              duration_seconds: 13,
            },
            {
              status: 'failed',
              started_at: 1751905016,
              updated_at: 1751905016,
              duration_seconds: 0,
            },
          ],
        },
      ],
    },
  },
]

type InflightFilterDraft = {
  sourceKey: string
  status: string
  kind: string
  stream: string
  model: string
  requestId: string
  channel: string
  startTime?: number
  endTime?: number
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

const retryStateVariant: Record<string, StatusBadgeProps['variant']> = {
  idle: 'neutral',
  retrying: 'warning',
  failed: 'danger',
}

function selectDisplayLabel(
  value: string,
  labels: Record<string, string>,
  fallback: string
) {
  return value ? labels[value] || value : fallback
}

function formatTime(timestamp?: number) {
  if (!timestamp) return '-'
  return dayjs(timestamp * 1000).format('YYYY-MM-DD HH:mm:ss')
}

function getRetryIndex(task: InflightTask) {
  return Math.max(0, task.detail?.retry_index ?? 0)
}

function getAttemptCount(task: InflightTask) {
  const attempts = getAttempts(task)
  const retryTotal = getRetryIndex(task) + 1
  if (attempts.length > 0) {
    return Math.max(attempts.length, retryTotal)
  }
  const chainLength = task.detail?.channel_chain?.length ?? 0
  if (chainLength > 0) {
    return Math.max(chainLength, retryTotal)
  }
  return Math.max(retryTotal, 1)
}

function hasInflightRetries(task: InflightTask) {
  return getRetryIndex(task) > 0 || getAttempts(task).length > 1
}

function getAttemptLabel(
  attempt: InflightTaskAttempt,
  task: InflightTask,
  t: (key: string, options?: Record<string, unknown>) => string
) {
  return t('Attempt {{current}} of {{total}}', {
    current: attempt.retry_index + 1,
    total: getAttemptCount(task),
  })
}

function isFinalFailure(task: InflightTask) {
  return task.status === 'failed'
}

function isInflightTaskTerminal(task: InflightTask) {
  return task.status === 'completed' || task.status === 'failed'
}

function getCurrentStage(task: InflightTask) {
  return task.detail?.current_stage || task.status
}

function getLatestError(task: InflightTask) {
  if (task.detail?.latest_error) {
    return task.detail.latest_error
  }
  const chain = task.detail?.channel_chain ?? []
  for (let i = chain.length - 1; i >= 0; i -= 1) {
    if (chain[i].error) {
      return chain[i].error
    }
  }
  return ''
}

function getRetryState(task: InflightTask) {
  if (isFinalFailure(task)) {
    return 'failed'
  }
  if (getRetryIndex(task) > 0) {
    return 'retrying'
  }
  return 'idle'
}

function getRetryStateLabel(
  task: InflightTask,
  t: (key: string, options?: Record<string, unknown>) => string
) {
  if (isFinalFailure(task)) {
    return t('Final Failure')
  }
  const retryIndex = getRetryIndex(task)
  if (retryIndex > 0) {
    return t('Retry {{count}}', { count: retryIndex })
  }
  return t('No Retry')
}

function getRetryAttemptLabel(
  task: InflightTask,
  t: (key: string, options?: Record<string, unknown>) => string
) {
  return t('Attempt {{current}} of {{total}}', {
    current: getRetryIndex(task) + 1,
    total: getAttemptCount(task),
  })
}

function getAttemptDisplayStatus(
  attempt: InflightTaskAttempt,
  task: InflightTask
) {
  if (
    attempt.retry_index === getRetryIndex(task) &&
    task.status === 'completed' &&
    isInflightTaskTerminal(task)
  ) {
    return 'completed'
  }
  return attempt.status
}

function getAttemptDisplayError(
  attempt: InflightTaskAttempt,
  task: InflightTask
) {
  if (getAttemptDisplayStatus(attempt, task) === 'completed') {
    return ''
  }
  return attempt.error ?? ''
}

function isBetterAttempt(
  candidate: InflightTaskAttempt,
  current: InflightTaskAttempt
) {
  const candidateTerminal =
    candidate.status === 'failed' || candidate.status === 'completed'
  const currentTerminal =
    current.status === 'failed' || current.status === 'completed'
  if (candidateTerminal !== currentTerminal) {
    return candidateTerminal
  }
  const candidateTimeline = candidate.timeline?.length ?? 0
  const currentTimeline = current.timeline?.length ?? 0
  if (candidateTimeline !== currentTimeline) {
    return candidateTimeline > currentTimeline
  }
  return (candidate.updated_at ?? 0) >= (current.updated_at ?? 0)
}

// dedupeAndSortAttempts guarantees a single entry per retry_index rendered in
// ascending retry order, tolerating older backend data that may still carry
// duplicate or out-of-order attempts.
function dedupeAndSortAttempts(
  attempts: InflightTaskAttempt[]
): InflightTaskAttempt[] {
  const byRetry = new Map<number, InflightTaskAttempt>()
  for (const attempt of attempts) {
    const existing = byRetry.get(attempt.retry_index)
    if (!existing || isBetterAttempt(attempt, existing)) {
      byRetry.set(attempt.retry_index, attempt)
    }
  }
  return Array.from(byRetry.values()).sort(
    (a, b) => a.retry_index - b.retry_index
  )
}

function attemptFromChannel(
  attempt: InflightTaskChannelAttempt,
  task: InflightTask
): InflightTaskAttempt {
  return {
    retry_index: attempt.retry_index,
    channel_id: attempt.channel_id,
    channel_name: attempt.channel_name,
    status: attempt.status,
    error: attempt.error,
    started_at: attempt.started_at,
    updated_at: attempt.updated_at,
    timeline:
      attempt.retry_index === getRetryIndex(task)
        ? (task.detail?.timeline ?? [])
        : [],
  }
}

function fillMissingInflightAttempts(
  task: InflightTask,
  resolved: InflightTaskAttempt[]
): InflightTaskAttempt[] {
  const retryIndex = getRetryIndex(task)
  const byRetry = new Map(
    resolved.map((attempt) => [attempt.retry_index, attempt] as const)
  )

  for (const chain of task.detail?.channel_chain ?? []) {
    if (!byRetry.has(chain.retry_index)) {
      byRetry.set(chain.retry_index, attemptFromChannel(chain, task))
    }
  }

  for (let index = 0; index <= retryIndex; index += 1) {
    if (byRetry.has(index)) {
      continue
    }
    const chain = task.detail?.channel_chain?.find(
      (attempt) => attempt.retry_index === index
    )
    if (chain) {
      byRetry.set(index, attemptFromChannel(chain, task))
      continue
    }
    if (
      index === retryIndex &&
      (task.detail?.channel_id || task.detail?.channel_name)
    ) {
      byRetry.set(index, {
        retry_index: index,
        channel_id: task.detail.channel_id,
        channel_name: task.detail.channel_name,
        status: task.status,
        error: task.status === 'completed' ? '' : task.detail.latest_error,
        started_at: task.detail.timeline?.[0]?.started_at ?? task.created_at,
        updated_at: task.updated_at,
        timeline: task.detail.timeline,
      })
    }
  }

  return dedupeAndSortAttempts(Array.from(byRetry.values()))
}

function getAttempts(task: InflightTask): InflightTaskAttempt[] {
  const attempts = task.detail?.attempts ?? []
  let resolved: InflightTaskAttempt[]
  if (attempts.length > 0) {
    resolved = dedupeAndSortAttempts(attempts)
  } else {
    const chain = task.detail?.channel_chain ?? []
    if (chain.length === 0) {
      resolved = []
    } else {
      resolved = dedupeAndSortAttempts(
        chain.map((attempt) => attemptFromChannel(attempt, task))
      )
    }
  }

  return fillMissingInflightAttempts(task, resolved)
}

function getTimelineAccentClass(status: string) {
  if (status === 'failed') {
    return 'bg-red-400/80'
  }
  if (status === 'completed') {
    return 'bg-emerald-400/80'
  }
  return 'bg-amber-400/80'
}

function getRetryPathAccentClass(
  task: InflightTask,
  attempt: InflightTaskChannelAttempt
) {
  if (attempt.status === 'failed') {
    return 'bg-red-400/80'
  }
  if (attempt.retry_index === getRetryIndex(task) && !isFinalFailure(task)) {
    return 'bg-amber-400/80'
  }
  return 'bg-border'
}

function shouldAttemptDefaultOpen(
  attempts: InflightTaskAttempt[],
  retryIndex: number
) {
  if (attempts.length <= 1) {
    return true
  }
  return retryIndex === attempts.at(-1)?.retry_index
}

function getAttemptSummary(
  attempt: InflightTaskAttempt,
  task: InflightTask,
  t: (key: string, options?: Record<string, unknown>) => string
) {
  const summary = [getAttemptLabel(attempt, task, t)]
  if (attempt.channel_name) {
    summary.push(attempt.channel_name)
  }
  if (getAttemptDisplayError(attempt, task)) {
    summary.push(t('Failure Reason'))
  }
  return summary.join(' · ')
}

function buildSourceKey(values: Record<string, unknown>) {
  return [
    values.status,
    values.kind,
    values.stream,
    values.model,
    values.requestId,
    values.channel,
    values.startTime,
    values.endTime,
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
    channel: props.searchParams.channel,
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

async function fetchInflightTaskByRequestId(
  requestId: string,
  params: Record<string, unknown>,
  useMock: boolean
) {
  if (useMock) {
    return (
      MOCK_INFLIGHT_TASKS.find((task) => task.request_id === requestId) ?? null
    )
  }
  const result = await fetchInflightTasks({
    ...params,
    request_id: requestId,
    p: 1,
    page_size: 1,
  })
  if (!result?.success) {
    return null
  }
  return result.data?.items?.[0] ?? null
}

function getMockInflightTasksResponse(
  params: Record<string, unknown>
): NonNullable<InflightTasksResponse['data']> {
  const page = typeof params.p === 'number' ? params.p : 1
  const pageSize = typeof params.page_size === 'number' ? params.page_size : 100
  const status = typeof params.status === 'string' ? params.status : ''
  const kind = typeof params.kind === 'string' ? params.kind : ''
  const stream = typeof params.is_stream === 'string' ? params.is_stream : ''
  const modelName =
    typeof params.model_name === 'string' ? params.model_name.toLowerCase() : ''
  const requestId =
    typeof params.request_id === 'string' ? params.request_id.toLowerCase() : ''
  const channel =
    typeof params.channel === 'string' ? params.channel.toLowerCase() : ''

  const filtered = MOCK_INFLIGHT_TASKS.filter((task) => {
    if (status && task.status !== status) return false
    if (kind && task.kind !== kind) return false
    if (stream === 'true' && !task.is_stream) return false
    if (stream === 'false' && task.is_stream) return false
    if (modelName && !task.model_name.toLowerCase().includes(modelName))
      return false
    if (requestId && !task.request_id.toLowerCase().includes(requestId))
      return false
    if (channel) {
      const currentChannel =
        `${task.detail?.channel_id ?? ''} ${task.detail?.channel_name ?? ''}`.toLowerCase()
      const retryChannels = (task.detail?.channel_chain ?? [])
        .map((attempt) =>
          `${attempt.channel_id ?? ''} ${attempt.channel_name ?? ''}`.toLowerCase()
        )
        .join(' ')
      if (
        !currentChannel.includes(channel) &&
        !retryChannels.includes(channel)
      ) {
        return false
      }
    }
    return true
  })

  const start = Math.max(0, (page - 1) * pageSize)
  const items = filtered.slice(start, start + pageSize)
  return {
    items,
    total: filtered.length,
    page,
    page_size: pageSize,
    meta: {
      trace_menu_visible: true,
    },
  }
}

function getChannelDisplay(task: InflightTask) {
  const channelId = task.detail?.channel_id
  const channelName = task.detail?.channel_name
  const channelIdDisplay = channelId ? `#${channelId}` : '-'
  const channelDisplay =
    channelId && channelName ? `${channelName} #${channelId}` : channelIdDisplay
  return { channelIdDisplay, channelDisplay, channelName }
}

function renderChannelCell(task: InflightTask, t: (key: string) => string) {
  const { channelIdDisplay, channelName } = getChannelDisplay(task)
  const attempts = getAttempts(task)
  const hasRetryChain = hasInflightRetries(task)
  const retryText = attempts
    .map((attempt) =>
      attempt.channel_id
        ? `#${attempt.channel_id}${attempt.channel_name ? ` ${attempt.channel_name}` : ''}`
        : '-'
    )
    .join(' → ')

  if (!task.detail?.channel_id) {
    return <span className='text-muted-foreground/60 text-xs'>-</span>
  }

  return (
    <div className='flex max-w-[180px] flex-col gap-0.5'>
      <div className='relative inline-flex w-fit items-center gap-1'>
        <StatusBadge
          label={channelIdDisplay}
          autoColor={String(task.detail.channel_id)}
          copyText={String(task.detail.channel_id)}
          size='sm'
          showDot={false}
          className='font-mono'
        />
        {hasRetryChain && (
          <Popover>
            <PopoverTrigger
              render={
                <button
                  type='button'
                  className='text-muted-foreground hover:text-foreground inline-flex size-5 shrink-0 items-center justify-center rounded-full transition-colors'
                  aria-label={t('Retry Chain')}
                  onClick={(event) => event.stopPropagation()}
                />
              }
            >
              <GitBranch
                className='size-3.5 text-amber-500'
                aria-hidden='true'
              />
            </PopoverTrigger>
            <PopoverContent side='top' align='start' className='w-64 text-xs'>
              <div className='flex flex-col gap-1'>
                <p className='font-medium'>{t('Retry Chain')}</p>
                <p className='text-muted-foreground font-mono break-all'>
                  {retryText}
                </p>
              </div>
            </PopoverContent>
          </Popover>
        )}
      </div>
      {channelName && (
        <span className='text-muted-foreground/70 truncate !text-xs'>
          {channelName}
        </span>
      )}
    </div>
  )
}

function buildDetailsSummary(task: InflightTask, t: (key: string) => string) {
  const timeline = task.detail?.timeline ?? []
  const totalDuration =
    timeline.length > 0
      ? timeline.reduce(
          (sum, step) => sum + Math.max(0, step.duration_seconds || 0),
          0
        )
      : Math.max(0, task.updated_at - task.created_at)
  const summary = [
    t(statusLabel[task.status] || task.status),
    `${totalDuration}s`,
  ]
  if (
    getRetryIndex(task) > 0 ||
    (task.detail?.channel_chain?.length ?? 0) > 1
  ) {
    summary.push(getRetryStateLabel(task, t))
  }
  if (getLatestError(task)) {
    summary.push(t('Latest Error'))
  }
  return summary.join(' · ')
}

function InflightTaskDetailsDialog(props: {
  task: InflightTask | null
  open: boolean
  onOpenChange: (open: boolean) => void
  isAdmin: boolean
  fetchParams: Record<string, unknown>
  useMock: boolean
}) {
  const { t } = useTranslation()
  const requestId = props.task?.request_id
  const {
    data: liveTask,
    isFetched,
    isLoading,
  } = useQuery({
    queryKey: [
      'inflight-task-detail',
      requestId,
      props.fetchParams,
      props.useMock,
    ],
    queryFn: () =>
      fetchInflightTaskByRequestId(
        requestId!,
        props.fetchParams,
        props.useMock
      ),
    enabled: props.open && !!requestId,
    placeholderData: props.task ?? undefined,
    refetchInterval: (query) => {
      if (!props.open) {
        return false
      }
      const task = query.state.data
      if (task && isInflightTaskTerminal(task)) {
        return false
      }
      return 2000
    },
    refetchIntervalInBackground: false,
  })

  useEffect(() => {
    if (!props.open || props.useMock || !requestId || isLoading || !isFetched) {
      return
    }
    if (liveTask === null) {
      toast.error(
        t('This inflight record does not exist or has been cleaned up')
      )
      props.onOpenChange(false)
    }
  }, [
    props.open,
    props.useMock,
    props.onOpenChange,
    requestId,
    isLoading,
    isFetched,
    liveTask,
    t,
  ])

  const task = isFetched && liveTask === null ? null : (liveTask ?? props.task)

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Details')}
      description={task ? buildDetailsSummary(task, t) : undefined}
      contentClassName='max-h-[calc(100dvh-2rem)] overflow-hidden max-sm:w-screen max-sm:max-w-none max-sm:rounded-none max-sm:p-4 sm:max-w-3xl'
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <Button
          variant='outline'
          onClick={() => props.onOpenChange(false)}
          className='w-full sm:w-auto'
        >
          {t('Close')}
        </Button>
      }
    >
      {task ? (
        <div className='max-h-[calc(100dvh-8.5rem)] space-y-3 overflow-y-auto py-2 pr-1 sm:max-h-[72vh] sm:space-y-4'>
          <InflightTaskDetails task={task} isAdmin={props.isAdmin} />
        </div>
      ) : null}
    </Dialog>
  )
}

function InflightTaskDetails(props: { task: InflightTask; isAdmin: boolean }) {
  const { t } = useTranslation()
  const timeline = props.task.detail?.timeline ?? []
  const attempts = getAttempts(props.task)
  const showRetryAttempts =
    props.isAdmin &&
    attempts.length > 0 &&
    (getRetryIndex(props.task) > 0 || attempts.length > 1)
  const { channelDisplay } = getChannelDisplay(props.task)
  const latestError = getLatestError(props.task)
  const retryState = getRetryState(props.task)
  const currentStage = getCurrentStage(props.task)

  return (
    <div className='w-full min-w-0 space-y-2.5 py-1 sm:space-y-3'>
      <div className='flex flex-wrap items-center gap-2'>
        <StatusBadge
          label={t(statusLabel[props.task.status] || props.task.status)}
          variant={statusVariant[props.task.status] || 'neutral'}
          size='sm'
          copyable={false}
        />
        {getRetryIndex(props.task) > 0 || isFinalFailure(props.task) ? (
          <StatusBadge
            label={getRetryStateLabel(props.task, t)}
            variant={retryStateVariant[retryState] || 'neutral'}
            size='sm'
            copyable={false}
          />
        ) : null}
        {getRetryIndex(props.task) > 0 ? (
          <StatusBadge
            label={getRetryAttemptLabel(props.task, t)}
            variant='neutral'
            size='sm'
            copyable={false}
            showDot={false}
            className='font-mono'
          />
        ) : null}
      </div>

      <div className='min-w-0 space-y-1'>
        <InflightDetailRow
          label={t('Request ID')}
          value={props.task.request_id}
          mono
        />
        <InflightDetailRow
          label={t('Model')}
          value={props.task.model_name || '-'}
        />
        <InflightDetailRow
          label={t('Status')}
          value={t(statusLabel[props.task.status] || props.task.status)}
        />
        <InflightDetailRow
          label={t('Current Stage')}
          value={t(statusLabel[currentStage] || currentStage)}
        />
        {hasInflightRetries(props.task) ? (
          <>
            <InflightDetailRow
              label={t('Current Attempt')}
              value={String(getRetryIndex(props.task) + 1)}
              mono
            />
            <InflightDetailRow
              label={t('Attempts')}
              value={String(getAttemptCount(props.task))}
              mono
            />
          </>
        ) : null}
        {props.isAdmin && props.task.detail?.channel_id ? (
          <InflightDetailRow
            label={t('Current Node')}
            value={channelDisplay}
            mono
          />
        ) : null}
        <InflightDetailRow
          label={t('Type')}
          value={t(kindLabel[props.task.kind] || props.task.kind)}
        />
        <InflightDetailRow
          label={t('Stream')}
          value={t(props.task.is_stream ? 'Yes' : 'No')}
        />
        <InflightDetailRow
          label={t('Started At')}
          value={formatTime(props.task.created_at)}
          mono
        />
        <InflightDetailRow
          label={t('Updated At')}
          value={formatTime(props.task.updated_at)}
          mono
        />
        {isFinalFailure(props.task) ? (
          <InflightDetailRow label={t('Final Failure')} value={t('Yes')} />
        ) : null}
      </div>

      {!showRetryAttempts && timeline.length > 0 && (
        <InflightDetailSection label={t('Timeline')}>
          <div className='space-y-2'>
            {timeline.map((step) => (
              <div
                key={`${step.status}-${step.started_at}-${step.updated_at}`}
                className='border-border/60 bg-background/80 relative space-y-1 rounded-md border p-2 pl-3'
              >
                <span
                  className={cn(
                    'absolute top-2 left-0 h-[calc(100%-1rem)] w-0.5 rounded-full',
                    getTimelineAccentClass(step.status)
                  )}
                  aria-hidden='true'
                />
                <div className='flex flex-wrap items-center gap-2'>
                  <StatusBadge
                    label={t(statusLabel[step.status] || step.status)}
                    variant={statusVariant[step.status] || 'neutral'}
                    size='sm'
                    copyable={false}
                  />
                  <StatusBadge
                    label={`${Math.max(0, step.duration_seconds || 0)}s`}
                    variant='neutral'
                    size='sm'
                    copyable={false}
                    showDot={false}
                    className='font-mono'
                  />
                </div>
                <InflightDetailRow
                  label={t('Started At')}
                  value={formatTime(step.started_at)}
                  mono
                />
                <InflightDetailRow
                  label={t('Updated At')}
                  value={formatTime(step.updated_at)}
                  mono
                />
                <InflightDetailRow
                  label={t('Duration')}
                  value={`${Math.max(0, step.duration_seconds || 0)}s`}
                  mono
                />
              </div>
            ))}
          </div>
        </InflightDetailSection>
      )}

      {showRetryAttempts && (
        <InflightDetailSection label={t('Retry Attempts')}>
          <div className='space-y-2'>
            {attempts.map((attempt) => {
              const displayStatus = getAttemptDisplayStatus(attempt, props.task)
              const displayError = getAttemptDisplayError(attempt, props.task)
              const legacyAttempt: InflightTaskChannelAttempt = {
                retry_index: attempt.retry_index,
                channel_id: attempt.channel_id,
                channel_name: attempt.channel_name,
                status: displayStatus,
                error: displayError,
                started_at: attempt.started_at,
                updated_at: attempt.updated_at,
              }
              const attemptTimeline = attempt.timeline ?? []
              const defaultOpen = shouldAttemptDefaultOpen(
                attempts,
                attempt.retry_index
              )
              return (
                <Collapsible
                  key={`${attempt.retry_index}-${attempt.channel_id ?? 'none'}-${attempt.started_at ?? 0}`}
                  defaultOpen={defaultOpen}
                  className={cn(
                    'rounded-md border border-border/60 bg-background/80',
                    displayError &&
                      'border-red-200/80 bg-red-50/50 dark:border-red-950 dark:bg-red-950/10'
                  )}
                >
                  <CollapsibleTrigger
                    render={
                      <button
                        type='button'
                        className='flex w-full items-start justify-between gap-3 p-3 text-left'
                        aria-label={getAttemptSummary(attempt, props.task, t)}
                      />
                    }
                  >
                    <div className='flex min-w-0 flex-1 flex-col gap-2'>
                      <div className='flex flex-wrap items-center gap-2'>
                        <StatusBadge
                          label={getAttemptLabel(attempt, props.task, t)}
                          variant='neutral'
                          size='sm'
                          copyable={false}
                          showDot={false}
                          className='font-mono'
                        />
                        {attempt.channel_id ? (
                          <StatusBadge
                            label={`#${attempt.channel_id}`}
                            autoColor={String(attempt.channel_id)}
                            copyText={String(attempt.channel_id)}
                            size='sm'
                            showDot={false}
                            className='font-mono'
                          />
                        ) : null}
                        {displayStatus ? (
                          <StatusBadge
                            label={t(
                              statusLabel[displayStatus] || displayStatus
                            )}
                            variant={statusVariant[displayStatus] || 'neutral'}
                            size='sm'
                            copyable={false}
                          />
                        ) : null}
                      </div>
                      <div className='text-muted-foreground text-xs'>
                        {attempt.channel_name || t('Unknown Channel')}
                      </div>
                    </div>
                    <div className='text-muted-foreground shrink-0 text-xs'>
                      {displayError ? t('Failure Reason') : t('Timeline')}
                    </div>
                  </CollapsibleTrigger>
                  <CollapsibleContent className='border-t px-3 py-3'>
                    <div className='space-y-3'>
                      <div
                        className={cn(
                          'relative space-y-1 rounded-md border border-border/60 bg-background/80 p-2 pl-3',
                          displayError &&
                            'border-red-200/80 bg-red-50/50 dark:border-red-950 dark:bg-red-950/10'
                        )}
                      >
                        <span
                          className={cn(
                            'absolute top-2 left-0 h-[calc(100%-1rem)] w-0.5 rounded-full',
                            getRetryPathAccentClass(props.task, legacyAttempt)
                          )}
                          aria-hidden='true'
                        />
                        {attempt.channel_name ? (
                          <InflightDetailRow
                            label={t('Channel')}
                            value={attempt.channel_name}
                          />
                        ) : null}
                        {attempt.started_at ? (
                          <InflightDetailRow
                            label={t('Started At')}
                            value={formatTime(attempt.started_at)}
                            mono
                          />
                        ) : null}
                        {attempt.updated_at ? (
                          <InflightDetailRow
                            label={t('Updated At')}
                            value={formatTime(attempt.updated_at)}
                            mono
                          />
                        ) : null}
                        {displayError ? (
                          <InflightDetailRow
                            label={t('Failure Reason')}
                            value={displayError}
                            muted
                          />
                        ) : null}
                      </div>
                      {attemptTimeline.length > 0 ? (
                        <div className='space-y-2'>
                          {attemptTimeline.map((step) => (
                            <div
                              key={`${attempt.retry_index}-${step.status}-${step.started_at}-${step.updated_at}`}
                              className='border-border/60 bg-background/80 relative space-y-1 rounded-md border p-2 pl-3'
                            >
                              <span
                                className={cn(
                                  'absolute top-2 left-0 h-[calc(100%-1rem)] w-0.5 rounded-full',
                                  getTimelineAccentClass(step.status)
                                )}
                                aria-hidden='true'
                              />
                              <div className='flex flex-wrap items-center gap-2'>
                                <StatusBadge
                                  label={t(
                                    statusLabel[step.status] || step.status
                                  )}
                                  variant={
                                    statusVariant[step.status] || 'neutral'
                                  }
                                  size='sm'
                                  copyable={false}
                                />
                                <StatusBadge
                                  label={`${Math.max(0, step.duration_seconds || 0)}s`}
                                  variant='neutral'
                                  size='sm'
                                  copyable={false}
                                  showDot={false}
                                  className='font-mono'
                                />
                              </div>
                              <InflightDetailRow
                                label={t('Started At')}
                                value={formatTime(step.started_at)}
                                mono
                              />
                              <InflightDetailRow
                                label={t('Updated At')}
                                value={formatTime(step.updated_at)}
                                mono
                              />
                              <InflightDetailRow
                                label={t('Duration')}
                                value={`${Math.max(0, step.duration_seconds || 0)}s`}
                                mono
                              />
                            </div>
                          ))}
                        </div>
                      ) : null}
                    </div>
                  </CollapsibleContent>
                </Collapsible>
              )
            })}
          </div>
        </InflightDetailSection>
      )}

      {latestError ? (
        <InflightDetailSection label={t('Latest Error')} variant='danger'>
          <p className='text-xs wrap-break-word'>{latestError}</p>
        </InflightDetailSection>
      ) : null}
    </div>
  )
}

function useInflightTaskColumns(props: {
  onOpenDetails: (task: InflightTask) => void
  onOpenTrace: (task: InflightTask) => void
  traceMenuVisible: boolean
  isAdmin: boolean
}): ColumnDef<InflightTask>[] {
  const { t } = useTranslation()
  const { isAdmin, onOpenDetails, onOpenTrace, traceMenuVisible } = props

  const showTraceMenu = useCallback(
    (task: InflightTask) =>
      traceMenuVisible &&
      (!isInflightTaskTerminal(task) || task.has_trace === true),
    [traceMenuVisible]
  )

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
      ...(isAdmin
        ? [
            {
              id: 'channel',
              header: t('Channel'),
              accessorFn: (row: InflightTask) => row.detail?.channel_id,
              cell: ({ row }: { row: Row<InflightTask> }) =>
                renderChannelCell(row.original, t),
            } satisfies ColumnDef<InflightTask>,
          ]
        : []),
      {
        id: 'retry',
        header: t('Retry'),
        accessorFn: (row: InflightTask) => getRetryIndex(row),
        cell: ({ row }) => {
          const task = row.original
          const retryState = getRetryState(task)
          return (
            <div className='flex max-w-[170px] flex-col gap-1'>
              <div className='flex flex-wrap items-center gap-1'>
                <StatusBadge
                  label={getRetryStateLabel(task, t)}
                  variant={retryStateVariant[retryState] || 'neutral'}
                  size='sm'
                  copyable={false}
                />
                <StatusBadge
                  label={getRetryAttemptLabel(task, t)}
                  variant='neutral'
                  size='sm'
                  copyable={false}
                  showDot={false}
                  className='font-mono'
                />
              </div>
            </div>
          )
        },
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
        id: 'latest_error',
        header: t('Latest Error'),
        accessorFn: (row: InflightTask) => getLatestError(row),
        cell: ({ row }) => {
          const latestError = getLatestError(row.original)
          if (!latestError) {
            return <span className='text-muted-foreground/60 text-xs'>-</span>
          }
          return (
            <TruncatedCell
              className={cn(
                'max-w-[280px] text-xs',
                isFinalFailure(row.original) && 'text-red-500'
              )}
            >
              {latestError}
            </TruncatedCell>
          )
        },
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
      {
        id: 'actions',
        header: t('Actions'),
        enableHiding: false,
        cell: ({ row }) => (
          <div
            className='flex items-center justify-end gap-1'
            onClick={(event) => event.stopPropagation()}
          >
            <Button
              variant='ghost'
              size='icon-sm'
              onClick={() => onOpenDetails(row.original)}
              aria-label={t('Details')}
            >
              <Eye />
            </Button>
            <DataTableRowActionMenu ariaLabel={t('Actions')}>
              <DropdownMenuItem onClick={() => onOpenDetails(row.original)}>
                <Eye />
                {t('Details')}
              </DropdownMenuItem>
              {showTraceMenu(row.original) ? (
                <DropdownMenuItem onClick={() => onOpenTrace(row.original)}>
                  <ScrollText />
                  {t('Debug Log')}
                </DropdownMenuItem>
              ) : null}
            </DataTableRowActionMenu>
          </div>
        ),
      },
    ],
    [isAdmin, onOpenDetails, onOpenTrace, showTraceMenu, t]
  )
}

function InflightFilterBar<TData>(props: {
  table: ReturnType<typeof useDataTable<TData>>['table']
  isFetching: boolean
  liveRefresh: boolean
  onToggleLiveRefresh: () => void
  onOpenLocalCache: () => void
  isAdmin: boolean
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const searchParams = route.useSearch()

  const searchState = useMemo<InflightFilterDraft>(() => {
    const { start, end } = getDefaultTimeRange()
    const sourceValues = {
      status: searchParams.status,
      kind: searchParams.kind,
      stream: searchParams.stream,
      model: searchParams.model,
      requestId: searchParams.requestId,
      channel: searchParams.channel,
      startTime:
        typeof searchParams.startTime === 'number'
          ? searchParams.startTime
          : start.getTime(),
      endTime:
        typeof searchParams.endTime === 'number'
          ? searchParams.endTime
          : end.getTime(),
    }
    return {
      sourceKey: buildSourceKey(sourceValues),
      status: searchParams.status || '',
      kind: searchParams.kind || '',
      stream: searchParams.stream || '',
      model: searchParams.model || '',
      requestId: searchParams.requestId || '',
      channel: searchParams.channel || '',
      startTime: sourceValues.startTime,
      endTime: sourceValues.endTime,
    }
  }, [
    searchParams.channel,
    searchParams.endTime,
    searchParams.kind,
    searchParams.model,
    searchParams.requestId,
    searchParams.startTime,
    searchParams.status,
    searchParams.stream,
  ])
  const [draft, setDraft] = useState<InflightFilterDraft>(() => searchState)
  const activeDraft =
    draft.sourceKey === searchState.sourceKey ? draft : searchState

  const handleChange = useCallback(
    (
      field: keyof Omit<InflightFilterDraft, 'sourceKey'>,
      value: string | number | undefined
    ) => {
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
        channel: activeDraft.channel || undefined,
        startTime: activeDraft.startTime,
        endTime: activeDraft.endTime,
        page: 1,
      },
    })
    queryClient.invalidateQueries({ queryKey: ['inflight-tasks'] })
  }, [activeDraft, navigate, queryClient])

  const handleReset = useCallback(() => {
    const { start, end } = getDefaultTimeRange()
    setDraft({
      sourceKey: '',
      status: '',
      kind: '',
      stream: '',
      model: '',
      requestId: '',
      channel: '',
      startTime: start.getTime(),
      endTime: end.getTime(),
    })
    navigate({
      to: '/usage-logs/$section',
      params: { section: 'inflight' },
      search: {
        page: 1,
        startTime: start.getTime(),
        endTime: end.getTime(),
      },
    })
    queryClient.invalidateQueries({ queryKey: ['inflight-tasks'] })
  }, [navigate, queryClient])

  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (event.key === 'Enter') handleApply()
    },
    [handleApply]
  )

  const hasExpandedFilters =
    !!activeDraft.model ||
    !!activeDraft.requestId ||
    (props.isAdmin && !!activeDraft.channel)

  const expandedFilterCount = [
    activeDraft.model,
    activeDraft.requestId,
    props.isAdmin ? activeDraft.channel : undefined,
  ].filter(Boolean).length

  const hasActiveFilters =
    !!activeDraft.status ||
    !!activeDraft.kind ||
    !!activeDraft.stream ||
    hasExpandedFilters

  const dateRangeFilter = (
    <LogsFilterField wide>
      <CompactDateTimeRangePicker
        start={
          typeof activeDraft.startTime === 'number'
            ? new Date(activeDraft.startTime)
            : undefined
        }
        end={
          typeof activeDraft.endTime === 'number'
            ? new Date(activeDraft.endTime)
            : undefined
        }
        onChange={({ start, end }) => {
          handleChange('startTime', start?.getTime())
          handleChange('endTime', end?.getTime())
        }}
      />
    </LogsFilterField>
  )

  const statusFilter = (
    <LogsFilterField>
      <Select
        value={activeDraft.status || 'all'}
        onValueChange={(value) =>
          handleChange('status', value === 'all' ? '' : (value ?? ''))
        }
      >
        <SelectTrigger className='h-8'>
          <SelectValue>
            {t(
              selectDisplayLabel(activeDraft.status, statusLabel, 'All Status')
            )}
          </SelectValue>
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
  )

  const kindFilter = (
    <LogsFilterField>
      <Select
        value={activeDraft.kind || 'all'}
        onValueChange={(value) =>
          handleChange('kind', value === 'all' ? '' : (value ?? ''))
        }
      >
        <SelectTrigger className='h-8'>
          <SelectValue>
            {t(selectDisplayLabel(activeDraft.kind, kindLabel, 'All Types'))}
          </SelectValue>
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
  )

  const streamFilter = (
    <LogsFilterField>
      <Select
        value={activeDraft.stream || 'all'}
        onValueChange={(value) =>
          handleChange('stream', value === 'all' ? '' : (value ?? ''))
        }
      >
        <SelectTrigger className='h-8'>
          <SelectValue>
            {t(selectDisplayLabel(activeDraft.stream, streamLabel, 'All'))}
          </SelectValue>
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
  )

  const advancedFilters = (
    <>
      {props.isAdmin ? (
        <LogsFilterField>
          <LogsFilterInput
            placeholder={t('Channel ID')}
            value={activeDraft.channel}
            onChange={(event) => handleChange('channel', event.target.value)}
            onKeyDown={handleKeyDown}
          />
        </LogsFilterField>
      ) : null}
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
      primaryFilters={
        <>
          {dateRangeFilter}
          {statusFilter}
          {kindFilter}
          {streamFilter}
        </>
      }
      advancedFilters={advancedFilters}
      mobilePinnedFilters={dateRangeFilter}
      mobileFilters={
        <>
          {statusFilter}
          {kindFilter}
          {streamFilter}
          {advancedFilters}
        </>
      }
      mobileFilterCount={
        [activeDraft.status, activeDraft.kind, activeDraft.stream].filter(
          Boolean
        ).length + expandedFilterCount
      }
      hasAdvancedActiveFilters={hasExpandedFilters}
      advancedFilterCount={expandedFilterCount}
      hasActiveFilters={hasActiveFilters}
      onReset={handleReset}
      onSearch={handleApply}
      searchLoading={props.isFetching && !props.liveRefresh}
      actionStart={
        <div className='flex items-center gap-2'>
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={props.onOpenLocalCache}
          >
            <FolderOpen />
            {t('Browser debug cache files')}
          </Button>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  type='button'
                  variant={props.liveRefresh ? 'secondary' : 'outline'}
                  size='icon'
                  onClick={props.onToggleLiveRefresh}
                  aria-pressed={props.liveRefresh}
                  aria-label={
                    props.liveRefresh
                      ? t('Stop live refresh')
                      : t('Start live refresh')
                  }
                />
              }
            >
              <RefreshCw
                className={cn('size-4', props.liveRefresh && 'animate-spin')}
              />
            </TooltipTrigger>
            <TooltipContent>
              {props.liveRefresh
                ? t('Stop live refresh')
                : t('Start live refresh')}
            </TooltipContent>
          </Tooltip>
        </div>
      }
    />
  )
}

export function InflightTasksTab() {
  const { t } = useTranslation()
  const isAdmin = useIsAdmin()
  const isMobile = useMediaQuery('(max-width: 640px)')
  const searchParams = route.useSearch()
  const [detailsTask, setDetailsTask] = useState<InflightTask | null>(null)
  const [traceTask, setTraceTask] = useState<InflightTask | null>(null)
  const [liveRefresh, setLiveRefresh] = useState(false)
  const [localCacheOpen, setLocalCacheOpen] = useState(false)
  const [localCacheFiles, setLocalCacheFiles] = useState<
    InflightTraceLocalCacheFile[]
  >([])
  const [selectedLocalCacheFiles, setSelectedLocalCacheFiles] = useState<
    string[]
  >([])
  const [localCacheDetail, setLocalCacheDetail] = useState<{
    fileName: string
    headers: string[]
    rows: Array<{
      requestId: string
      recordedAt: string
      traceJson: string
      formattedTraceJson: string
    }>
  } | null>(null)

  const refreshLocalCacheFiles = useCallback(async () => {
    try {
      setLocalCacheFiles(await listInflightTraceLocalCacheFiles())
    } catch {
      toast.error(t('Failed to load local cache files'))
    }
  }, [t])

  const openLocalCache = useCallback(async () => {
    setLocalCacheOpen(true)
    await refreshLocalCacheFiles()
  }, [refreshLocalCacheFiles])
  const localCacheSize = useMemo(() => {
    const selectedFileNames = new Set(selectedLocalCacheFiles)
    const sizeBytes = localCacheFiles.reduce(
      (total, file) =>
        total +
        (selectedFileNames.size === 0 || selectedFileNames.has(file.fileName)
          ? file.sizeBytes
          : 0),
      0
    )
    return (sizeBytes / 1024).toFixed(2)
  }, [localCacheFiles, selectedLocalCacheFiles])
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

  const onOpenDetails = useCallback((task: InflightTask) => {
    setDetailsTask(task)
  }, [])

  const onOpenTrace = useCallback((task: InflightTask) => {
    setTraceTask(task)
  }, [])

  const detailFetchParams = useMemo(
    () =>
      buildParams({
        page: 1,
        pageSize: 1,
        searchParams,
      }),
    [searchParams]
  )

  const { data, isLoading, isFetching, refetch } = useQuery({
    queryKey: [
      'inflight-tasks',
      pagination.pageIndex + 1,
      pagination.pageSize,
      searchParams,
    ],
    queryFn: async () => {
      if (searchParams.mock === 'inflight') {
        return getMockInflightTasksResponse(
          buildParams({
            page: pagination.pageIndex + 1,
            pageSize: pagination.pageSize,
            searchParams,
          })
        )
      }
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
    refetchOnMount: 'always',
    refetchOnWindowFocus: false,
    refetchInterval: liveRefresh ? inflightListLiveRefreshMs : false,
    refetchIntervalInBackground: false,
  })

  const handleToggleLiveRefresh = useCallback(() => {
    setLiveRefresh((enabled) => {
      const next = !enabled
      if (next) {
        void refetch()
      }
      return next
    })
  }, [refetch])

  const columns = useInflightTaskColumns({
    onOpenDetails,
    onOpenTrace,
    traceMenuVisible: data?.meta?.trace_menu_visible === true,
    isAdmin,
  })

  const { table } = useDataTable({
    data: data?.items ?? [],
    columns,
    columnFilters,
    columnVisibilityStorageKey: inflightColumnVisibilityStorageKey,
    pagination,
    enableRowSelection: false,
    onPaginationChange,
    onColumnFiltersChange,
    manualPagination: true,
    manualFiltering: true,
    totalCount: data?.total ?? 0,
    pageCount: Math.max(
      1,
      Math.ceil((data?.total ?? 0) / Math.max(1, pagination.pageSize))
    ),
    ensurePageInRange,
  })

  return (
    <>
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
          <InflightFilterBar
            table={table}
            isFetching={isFetching}
            liveRefresh={liveRefresh}
            onToggleLiveRefresh={handleToggleLiveRefresh}
            isAdmin={isAdmin}
            onOpenLocalCache={openLocalCache}
          />
        }
        renderRow={(row, helpers) => (
          <ContextMenu key={row.id}>
            <ContextMenuTrigger
              render={
                <DataTableRow
                  row={row}
                  className={cn(
                    isFinalFailure(row.original) &&
                      'bg-red-50/40 dark:bg-red-950/15',
                    !isFinalFailure(row.original) &&
                      getRetryIndex(row.original) > 0 &&
                      'bg-amber-50/30 dark:bg-amber-950/10'
                  )}
                  getColumnClassName={(columnId) =>
                    helpers.getCellClassName(columnId, 'py-3.5')
                  }
                  cellRenderColumns={columns}
                />
              }
            />
            <ContextMenuContent>
              <ContextMenuItem onClick={() => onOpenDetails(row.original)}>
                <Eye />
                {t('Details')}
              </ContextMenuItem>
              {data?.meta?.trace_menu_visible === true &&
              (!isInflightTaskTerminal(row.original) ||
                row.original.has_trace === true) ? (
                <ContextMenuItem onClick={() => onOpenTrace(row.original)}>
                  <ScrollText />
                  {t('Debug Log')}
                </ContextMenuItem>
              ) : null}
            </ContextMenuContent>
          </ContextMenu>
        )}
        tableClassName='[&_[data-slot=table]]:text-[13px] [&_[data-slot=table]_td]:text-[13px] [&_[data-slot=table]_td_*]:text-[13px] [&_[data-slot=table]_th]:text-[13px] [&_[data-slot=table]_th_*]:text-[13px]'
      />
      <Dialog
        open={localCacheOpen}
        onOpenChange={setLocalCacheOpen}
        title={t('Browser debug cache files')}
        description={t('Manage debug archives cached in this browser.')}
        footer={
          <Button variant='outline' onClick={() => setLocalCacheOpen(false)}>
            {t('Close')}
          </Button>
        }
      >
        <div className='flex flex-wrap gap-2'>
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={() =>
              setSelectedLocalCacheFiles(
                localCacheFiles.map((file) => file.fileName)
              )
            }
          >
            {t('Select all')}
          </Button>
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={() =>
              setSelectedLocalCacheFiles((selected) =>
                localCacheFiles
                  .filter((file) => !selected.includes(file.fileName))
                  .map((file) => file.fileName)
              )
            }
          >
            {t('Invert selection')}
          </Button>
          <Button
            type='button'
            variant='destructive'
            size='sm'
            disabled={!selectedLocalCacheFiles.length}
            onClick={async () => {
              await deleteInflightTraceLocalCacheFiles(selectedLocalCacheFiles)
              setSelectedLocalCacheFiles([])
              await refreshLocalCacheFiles()
            }}
          >
            <Trash2 />
            {t('Delete selected')}
          </Button>
          <span className='text-muted-foreground self-center text-sm'>
            {selectedLocalCacheFiles.length > 0
              ? t('Current selected cached files size: {{size}} KB', {
                  size: localCacheSize,
                })
              : t('Current total cached files size: {{size}} KB', {
                  size: localCacheSize,
                })}
          </span>
        </div>
        <div className='mt-4 max-h-80 space-y-2 overflow-auto rounded-md border p-3'>
          {localCacheFiles.length === 0 ? (
            <p className='text-muted-foreground text-sm'>
              {t('No local cache files.')}
            </p>
          ) : (
            localCacheFiles.map((file) => (
              <div
                key={file.fileName}
                className='flex items-center gap-2 text-sm'
              >
                <label className='flex min-w-0 flex-1 items-center gap-2'>
                  <Checkbox
                    checked={selectedLocalCacheFiles.includes(file.fileName)}
                    onCheckedChange={(checked) =>
                      setSelectedLocalCacheFiles((selected) =>
                        checked
                          ? [...selected, file.fileName]
                          : selected.filter((value) => value !== file.fileName)
                      )
                    }
                  />
                  <span className='min-w-0 flex-1 truncate'>{file.fileName}</span>
                </label>
                <div className='flex shrink-0 gap-2'>
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    onClick={async () => {
                      try {
                        const csvRows = await readInflightTraceLocalCacheRows(
                          file.fileName
                        )
                        if (csvRows === null) {
                          toast.error(t('Failed to load local cache files'))
                          return
                        }
                        const [headers = [], ...rows] = csvRows
                        setLocalCacheDetail({
                          fileName: file.fileName,
                          headers,
                          rows: rows.map((row) => ({
                            requestId: row[0] ?? '',
                            recordedAt: row[1] ?? '',
                            traceJson: row[2] ?? '',
                            formattedTraceJson: tryPrettyJson(row[2] ?? ''),
                          })),
                        })
                      } catch {
                        toast.error(t('Failed to load local cache files'))
                      }
                    }}
                  >
                    <Eye />
                    {t('Details')}
                  </Button>
                  <Button
                    type='button'
                    variant='destructive'
                    size='sm'
                    onClick={async () => {
                      await deleteInflightTraceLocalCacheFiles([file.fileName])
                      setSelectedLocalCacheFiles((selected) =>
                        selected.filter((value) => value !== file.fileName)
                      )
                      await refreshLocalCacheFiles()
                    }}
                  >
                    <Trash2 />
                    {t('Delete')}
                  </Button>
                </div>
              </div>
            ))
          )}
        </div>
      </Dialog>
      <Dialog
        open={localCacheDetail !== null}
        onOpenChange={(open) => {
          if (!open) setLocalCacheDetail(null)
        }}
        title={t('Details')}
        description={localCacheDetail?.fileName}
        contentClassName='max-h-[calc(100dvh-2rem)] sm:max-w-6xl'
        footer={
          <Button variant='outline' onClick={() => setLocalCacheDetail(null)}>
            {t('Close')}
          </Button>
        }
      >
        <div className='max-h-[60vh] overflow-auto rounded-md border'>
          <Table className='table-fixed'>
            <colgroup>
              <col className='w-64' />
              <col className='w-36' />
              <col />
            </colgroup>
            <TableHeader className='bg-background sticky top-0 z-10'>
              <TableRow>
                {(localCacheDetail?.headers ?? []).map((header) => (
                  <TableHead key={header}>{header}</TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {(localCacheDetail?.rows ?? []).map((row, index) => (
                <TableRow key={`${row.requestId}-${row.recordedAt}-${index}`}>
                  <TableCell className='truncate font-mono text-xs'>
                    {row.requestId}
                  </TableCell>
                  <TableCell className='font-mono text-xs'>
                    {row.recordedAt}
                  </TableCell>
                  <TableCell className='min-w-0 overflow-hidden'>
                    <Tooltip>
                      <TooltipTrigger
                        render={
                          <button
                            type='button'
                            className='block w-full truncate text-left font-mono text-xs'
                          />
                        }
                      >
                        {row.traceJson}
                      </TooltipTrigger>
                      <TooltipContent
                        side='top'
                        align='end'
                        sideOffset={8}
                        arrowClassName='bg-popover fill-popover'
                        className='bg-popover text-popover-foreground border-border max-h-[40vh] w-[48rem] max-w-[calc(100vw-3rem)] items-start overflow-auto border p-3 shadow-lg'
                      >
                        <pre className='text-popover-foreground min-w-0 max-w-full text-left font-mono text-xs whitespace-pre-wrap break-all'>
                          {row.formattedTraceJson}
                        </pre>
                      </TooltipContent>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </Dialog>
      <InflightTaskDetailsDialog
        task={detailsTask}
        open={detailsTask !== null}
        onOpenChange={(open) => {
          if (!open) setDetailsTask(null)
        }}
        isAdmin={isAdmin}
        fetchParams={detailFetchParams}
        useMock={searchParams.mock === 'inflight'}
      />
      <InflightTaskTraceDialog
        requestId={traceTask?.request_id ?? null}
        open={traceTask !== null}
        onOpenChange={(open) => {
          if (!open) setTraceTask(null)
        }}
        useMock={searchParams.mock === 'inflight'}
        summary={
          traceTask
            ? {
                model_name: traceTask.model_name,
                status: traceTask.status,
                is_stream: traceTask.is_stream,
                has_trace: traceTask.has_trace,
              }
            : undefined
        }
      />
    </>
  )
}
