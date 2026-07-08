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
import { RefreshCw } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { StatusBadge, type StatusBadgeProps } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { api } from '@/lib/api'
import dayjs from '@/lib/dayjs'

type InflightTask = {
  request_id: string
  status: string
  kind: string
  model_name: string
  is_stream: boolean
  created_at: number
  updated_at: number
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

async function fetchInflightTasks(): Promise<{
  success: boolean
  message?: string
  data?: InflightTask[]
}> {
  const res = await api.get('/api/log/inflight/self', {
    disableDuplicate: true,
    skipBusinessError: true,
  })
  return res.data
}

function formatTime(timestamp: number) {
  if (!timestamp) return '-'
  return dayjs(timestamp * 1000).format('YYYY-MM-DD HH:mm:ss')
}

export function InflightTasksTab() {
  const { t } = useTranslation()
  const { data, isLoading, isFetching, refetch } = useQuery({
    queryKey: ['inflight-tasks'],
    queryFn: fetchInflightTasks,
    refetchInterval: 5000,
    refetchIntervalInBackground: false,
  })

  const tasks = data?.success ? (data.data ?? []) : []
  const message = !data?.success ? data?.message : ''

  let tableContent: ReactNode = tasks.map((task) => (
    <TableRow key={task.request_id}>
      <TableCell>
        <StatusBadge
          label={t(statusLabel[task.status] || task.status)}
          variant={statusVariant[task.status] || 'neutral'}
          copyable={false}
        />
      </TableCell>
      <TableCell>{t(kindLabel[task.kind] || task.kind)}</TableCell>
      <TableCell className='max-w-[220px] truncate'>
        {task.model_name || '-'}
      </TableCell>
      <TableCell className='font-mono text-xs'>{task.request_id}</TableCell>
      <TableCell>{formatTime(task.created_at)}</TableCell>
      <TableCell>{formatTime(task.updated_at)}</TableCell>
      <TableCell>{task.is_stream ? t('Yes') : t('No')}</TableCell>
    </TableRow>
  ))

  if (isLoading) {
    tableContent = (
      <TableRow>
        <TableCell colSpan={7} className='text-muted-foreground h-24 text-center'>
          {t('Loading...')}
        </TableCell>
      </TableRow>
    )
  } else if (message) {
    tableContent = (
      <TableRow>
        <TableCell colSpan={7} className='text-muted-foreground h-24 text-center'>
          {message}
        </TableCell>
      </TableRow>
    )
  } else if (tasks.length === 0) {
    tableContent = (
      <TableRow>
        <TableCell colSpan={7} className='text-muted-foreground h-24 text-center'>
          {t('No inflight logs.')}
        </TableCell>
      </TableRow>
    )
  }

  return (
    <div className='flex h-full min-h-0 flex-col gap-3'>
      <div className='flex items-center justify-end'>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={() => void refetch()}
          disabled={isFetching}
        >
          <RefreshCw className='size-4' />
          {t('Refresh')}
        </Button>
      </div>

      <div className='min-h-0 flex-1 overflow-auto rounded-md border'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('Status')}</TableHead>
              <TableHead>{t('Type')}</TableHead>
              <TableHead>{t('Model')}</TableHead>
              <TableHead>{t('Request ID')}</TableHead>
              <TableHead>{t('Started At')}</TableHead>
              <TableHead>{t('Updated At')}</TableHead>
              <TableHead>{t('Stream')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>{tableContent}</TableBody>
        </Table>
      </div>
    </div>
  )
}
