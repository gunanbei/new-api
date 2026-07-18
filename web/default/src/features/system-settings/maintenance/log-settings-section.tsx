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
import { zodResolver } from '@hookform/resolvers/zod'
import { InformationCircleIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useForm, type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { DateTimePicker } from '@/components/datetime-picker'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { api } from '@/lib/api'
import dayjs from '@/lib/dayjs'
import { formatTimestampToDate } from '@/lib/format'

import {
  getCurrentInflightLogCleanupTask,
  getInflightTaskStats,
  getSystemTask,
  startInflightLogCleanupTask,
  triggerInflightTraceArchiveUploads,
} from '../api'
import {
  SettingsControlGroup,
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import type { LogCleanupTask } from '../types'
import {
  InflightCleanupScheduleFields,
  type ScheduleFormValues,
} from './inflight-cleanup-schedule-fields'
import {
  INFLIGHT_CLEANUP_INTERVAL_MAX,
  INFLIGHT_CLEANUP_INTERVAL_MIN,
} from './inflight-cleanup-cron'

const logSettingsSchema = z.object({
  LogConsumeEnabled: z.boolean(),
  InflightTaskCleanupRule: z.number().int().min(1).max(2),
  InflightTaskCleanupScheduleMode: z.enum(['interval', 'cron']),
  InflightTaskCleanupIntervalMinutes: z
    .number()
    .int()
    .min(
      INFLIGHT_CLEANUP_INTERVAL_MIN,
      'Inflight log cleanup interval must be between 1 and 10080 minutes'
    )
    .max(
      INFLIGHT_CLEANUP_INTERVAL_MAX,
      'Inflight log cleanup interval must be between 1 and 10080 minutes'
    ),
  InflightTaskCleanupCron: z.string().min(1, 'Invalid cron expression'),
  InflightTaskTraceEnabled: z.boolean(),
  InflightTaskTraceMenuVisible: z.boolean(),
  InflightTaskTraceMaxRequestBytes: z.number().int().min(0),
  InflightTaskTraceMaxResponseBytes: z.number().int().min(0),
  InflightTaskTraceStorageMode: z.enum(['memory', 'disk']),
  InflightTaskTraceArchiveChannelID: z.number().int().min(0),
  InflightTaskTraceArchiveThresholdBytes: z.number().int().min(1),
  InflightTaskTraceArchiveRetentionYears: z.number().int().min(0),
  InflightTaskTraceArchiveRetentionMonths: z.number().int().min(0),
  InflightTaskTraceArchiveRetentionDays: z.number().int().min(0),
  InflightTaskTraceArchiveRetentionHours: z.number().int().min(0),
  InflightTracePath: z.string(),
})

type LogSettingsFormValues = z.infer<typeof logSettingsSchema>

type LogSettingsSectionProps = {
  defaultEnabled: boolean
  defaultInflightTaskCleanupRule: number
  defaultInflightTaskCleanupIntervalMinutes: number
  defaultInflightTaskCleanupScheduleMode: 'interval' | 'cron'
  defaultInflightTaskCleanupCron: string
  defaultInflightTaskTraceEnabled: boolean
  defaultInflightTaskTraceMenuVisible: boolean
  defaultInflightTaskTraceMaxRequestBytes: number
  defaultInflightTaskTraceMaxResponseBytes: number
  defaultInflightTaskTraceStorageMode: 'memory' | 'disk'
  defaultInflightTaskTraceArchiveChannelID: number
  defaultInflightTaskTraceArchiveThresholdBytes: number
  defaultInflightTaskTraceArchiveRetentionYears: number
  defaultInflightTaskTraceArchiveRetentionMonths: number
  defaultInflightTaskTraceArchiveRetentionDays: number
  defaultInflightTaskTraceArchiveRetentionHours: number
  defaultInflightTracePath: string
}

type ServerLogInfo = {
  enabled: boolean
  log_dir: string
  file_count: number
  total_size: number
  oldest_time?: string
  newest_time?: string
}

type InflightTaskStats = {
  user_count: number
  item_count: number
  total_size: number
  in_memory_count: number
  in_memory_size: number
  local_trace_count: number
  local_trace_size: number
  uploaded_trace_count: number
  uploaded_trace_size: number
  trace_directory?: string
  trace_pending_upload_count?: number
  trace_uploading_count?: number
}

type FileUploadChannel = {
  id: number
  name: string
  type: string
  status: string
}

const HOURS_IN_DAY = 24

function formatBytes(bytes: number, decimals = 2): string {
  if (!bytes || Number.isNaN(bytes)) return '0 Bytes'
  if (bytes === 0) return '0 Bytes'
  if (bytes < 0) return `-${formatBytes(-bytes, decimals)}`
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(Math.abs(bytes)) / Math.log(k))
  if (i < 0 || i >= sizes.length) return `${bytes} Bytes`
  return `${Number.parseFloat((bytes / Math.pow(k, i)).toFixed(decimals))} ${
    sizes[i]
  }`
}

function formatKilobytes(bytes: number): string {
  return `${(bytes / 1024).toFixed(2)} KB`
}

function InflightMetricTooltip({ label, children }: { label: string; children: ReactNode }) {
  const [open, setOpen] = useState(false)

  return (
    <Tooltip open={open} onOpenChange={setOpen}>
      <TooltipTrigger
        closeOnClick={false}
        render={
          <button
            type='button'
            className='inline-flex size-3.5 items-center justify-center'
            aria-label={label}
            onClick={() => setOpen(true)}
          >
            <HugeiconsIcon icon={InformationCircleIcon} strokeWidth={2} className='size-3.5' />
          </button>
        }
      />
      <TooltipContent>
        <div className='space-y-1'>{children}</div>
      </TooltipContent>
    </Tooltip>
  )
}

const getDateHoursAgo = (hours: number) => {
  const date = new Date()
  date.setHours(date.getHours() - hours)
  return date
}

const getDateDaysAgo = (days: number) => getDateHoursAgo(days * HOURS_IN_DAY)

const quickSelectOptions = [
  {
    label: '24 hours ago',
    getValue: () => getDateHoursAgo(24),
  },
  {
    label: '7 days ago',
    getValue: () => getDateDaysAgo(7),
  },
  {
    label: '30 days ago',
    getValue: () => getDateDaysAgo(30),
  },
]

function isActiveLogCleanupTask(task: LogCleanupTask | null) {
  return task?.status === 'pending' || task?.status === 'running'
}

export function LogSettingsSection({
  defaultEnabled,
  defaultInflightTaskCleanupRule,
  defaultInflightTaskCleanupIntervalMinutes,
  defaultInflightTaskCleanupScheduleMode,
  defaultInflightTaskCleanupCron,
  defaultInflightTaskTraceEnabled,
  defaultInflightTaskTraceMenuVisible,
  defaultInflightTaskTraceMaxRequestBytes,
  defaultInflightTaskTraceMaxResponseBytes,
  defaultInflightTaskTraceStorageMode,
  defaultInflightTaskTraceArchiveChannelID,
  defaultInflightTaskTraceArchiveThresholdBytes,
  defaultInflightTaskTraceArchiveRetentionYears,
  defaultInflightTaskTraceArchiveRetentionMonths,
  defaultInflightTaskTraceArchiveRetentionDays,
  defaultInflightTaskTraceArchiveRetentionHours,
  defaultInflightTracePath,
}: LogSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm<LogSettingsFormValues>({
    resolver: zodResolver(logSettingsSchema),
    defaultValues: {
      LogConsumeEnabled: defaultEnabled,
      InflightTaskCleanupRule: defaultInflightTaskCleanupRule,
      InflightTaskCleanupIntervalMinutes:
        defaultInflightTaskCleanupIntervalMinutes,
      InflightTaskCleanupScheduleMode: defaultInflightTaskCleanupScheduleMode,
      InflightTaskCleanupCron: defaultInflightTaskCleanupCron,
      InflightTaskTraceEnabled: defaultInflightTaskTraceEnabled,
      InflightTaskTraceMenuVisible: defaultInflightTaskTraceMenuVisible,
      InflightTaskTraceMaxRequestBytes: defaultInflightTaskTraceMaxRequestBytes,
      InflightTaskTraceMaxResponseBytes:
        defaultInflightTaskTraceMaxResponseBytes,
      InflightTaskTraceStorageMode: defaultInflightTaskTraceStorageMode,
      InflightTaskTraceArchiveChannelID: defaultInflightTaskTraceArchiveChannelID,
      InflightTaskTraceArchiveThresholdBytes: defaultInflightTaskTraceArchiveThresholdBytes,
      InflightTaskTraceArchiveRetentionYears: defaultInflightTaskTraceArchiveRetentionYears,
      InflightTaskTraceArchiveRetentionMonths: defaultInflightTaskTraceArchiveRetentionMonths,
      InflightTaskTraceArchiveRetentionDays: defaultInflightTaskTraceArchiveRetentionDays,
      InflightTaskTraceArchiveRetentionHours: defaultInflightTaskTraceArchiveRetentionHours,
      InflightTracePath: defaultInflightTracePath,
    },
  })

  const [purgeDate, setPurgeDate] = useState<Date | undefined>(() =>
    getDateDaysAgo(30)
  )
  const [isStartingLogCleanup, setIsStartingLogCleanup] = useState(false)
  const [logCleanupTask, setLogCleanupTask] = useState<LogCleanupTask | null>(
    null
  )
  const [showConfirmDialog, setShowConfirmDialog] = useState(false)
  const [inflightTaskStats, setInflightTaskStats] =
    useState<InflightTaskStats | null>(null)
  const [serverLogInfo, setServerLogInfo] = useState<ServerLogInfo | null>(null)
  const [serverLogCleanupMode, setServerLogCleanupMode] = useState('by_count')
  const [serverLogCleanupValue, setServerLogCleanupValue] = useState(10)
  const [serverLogCleanupLoading, setServerLogCleanupLoading] = useState(false)
  const [archiveChannels, setArchiveChannels] = useState<FileUploadChannel[]>([])

  const archiveUploadInProgress =
    (inflightTaskStats?.trace_uploading_count ?? 0) > 0

  const fetchServerLogInfo = useCallback(async () => {
    try {
      const res = await api.get('/api/performance/logs')
      if (res.data.success) setServerLogInfo(res.data.data)
    } catch {
      /* ignore */
    }
  }, [])

  useEffect(() => {
    form.reset({
      LogConsumeEnabled: defaultEnabled,
      InflightTaskCleanupRule: defaultInflightTaskCleanupRule,
      InflightTaskCleanupIntervalMinutes:
        defaultInflightTaskCleanupIntervalMinutes,
      InflightTaskCleanupScheduleMode: defaultInflightTaskCleanupScheduleMode,
      InflightTaskCleanupCron: defaultInflightTaskCleanupCron,
      InflightTaskTraceEnabled: defaultInflightTaskTraceEnabled,
      InflightTaskTraceMenuVisible: defaultInflightTaskTraceMenuVisible,
      InflightTaskTraceMaxRequestBytes: defaultInflightTaskTraceMaxRequestBytes,
      InflightTaskTraceMaxResponseBytes:
        defaultInflightTaskTraceMaxResponseBytes,
      InflightTaskTraceStorageMode: defaultInflightTaskTraceStorageMode,
      InflightTaskTraceArchiveChannelID: defaultInflightTaskTraceArchiveChannelID,
      InflightTaskTraceArchiveThresholdBytes: defaultInflightTaskTraceArchiveThresholdBytes,
      InflightTaskTraceArchiveRetentionYears: defaultInflightTaskTraceArchiveRetentionYears,
      InflightTaskTraceArchiveRetentionMonths: defaultInflightTaskTraceArchiveRetentionMonths,
      InflightTaskTraceArchiveRetentionDays: defaultInflightTaskTraceArchiveRetentionDays,
      InflightTaskTraceArchiveRetentionHours: defaultInflightTaskTraceArchiveRetentionHours,
      InflightTracePath: defaultInflightTracePath,
    })
  }, [
    defaultEnabled,
    defaultInflightTaskCleanupIntervalMinutes,
    defaultInflightTaskCleanupRule,
    defaultInflightTaskCleanupScheduleMode,
    defaultInflightTaskCleanupCron,
    defaultInflightTaskTraceEnabled,
    defaultInflightTaskTraceMaxRequestBytes,
    defaultInflightTaskTraceMaxResponseBytes,
    defaultInflightTaskTraceMenuVisible,
    defaultInflightTaskTraceStorageMode,
    defaultInflightTaskTraceArchiveChannelID,
    defaultInflightTaskTraceArchiveThresholdBytes,
    defaultInflightTaskTraceArchiveRetentionYears,
    defaultInflightTaskTraceArchiveRetentionMonths,
    defaultInflightTaskTraceArchiveRetentionDays,
    defaultInflightTaskTraceArchiveRetentionHours,
    defaultInflightTracePath,
    form,
  ])

  useEffect(() => {
    fetchServerLogInfo()
  }, [fetchServerLogInfo])

  useEffect(() => {
    api
      .get<{ success: boolean; data?: FileUploadChannel[] }>('/api/file-upload-channel/', {
        params: { status: '1' },
      })
      .then((res) => setArchiveChannels(res.data.data ?? []))
      .catch(() => setArchiveChannels([]))
  }, [])

  useEffect(() => {
    let cancelled = false

    async function fetchCurrentLogCleanupTask() {
      try {
        const res = await getCurrentInflightLogCleanupTask()
        if (!cancelled && res.success && res.data) {
          setLogCleanupTask(res.data)
        }
      } catch {
        /* ignore */
      }
    }

    async function fetchStats() {
      try {
        const res = await getInflightTaskStats()
        if (!cancelled && res.success && res.data) {
          setInflightTaskStats(res.data)
        }
      } catch {
        /* ignore */
      }
    }

    fetchCurrentLogCleanupTask()
    fetchStats()

    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    if (!archiveUploadInProgress) return

    let cancelled = false
    const interval = window.setInterval(async () => {
      try {
        const res = await getInflightTaskStats()
        if (!cancelled && res.success && res.data) {
          setInflightTaskStats(res.data)
        }
      } catch {
        /* ignore */
      }
    }, 1000)

    return () => {
      cancelled = true
      window.clearInterval(interval)
    }
  }, [archiveUploadInProgress])

  const purgeTimestamp = useMemo(() => {
    if (!purgeDate) return null
    return Math.floor(purgeDate.getTime() / 1000)
  }, [purgeDate])

  const formattedPurgeDate = useMemo(() => {
    if (!purgeDate) return ''
    return formatTimestampToDate(purgeDate.getTime(), 'milliseconds')
  }, [purgeDate])

  const logCleanupActive = isActiveLogCleanupTask(logCleanupTask)
  const logCleanupState = logCleanupTask?.state
  const logCleanupProgress = Math.min(
    100,
    Math.max(0, logCleanupState?.progress ?? 0)
  )
  const logCleanupProcessed = logCleanupState?.processed ?? 0
  const logCleanupTotal = logCleanupState?.total ?? 0
  const logCleanupTaskId = logCleanupTask?.task_id

  useEffect(() => {
    if (!logCleanupTaskId || !logCleanupActive) return

    let cancelled = false
    const interval = window.setInterval(async () => {
      try {
        const res = await getSystemTask(logCleanupTaskId)
        if (cancelled || !res.success || !res.data) return

        setLogCleanupTask(res.data)
        if (!isActiveLogCleanupTask(res.data)) {
          if (res.data.status === 'succeeded') {
            const count =
              res.data.result?.deleted_count ?? res.data.state?.processed ?? 0
            toast.success(
              count > 0
                ? t('{{count}} log entries removed.', { count })
                : t('No log entries matched the selected time.')
            )
          } else if (res.data.status === 'failed') {
            toast.error(res.data.error || t('Failed to clean logs'))
          }
        }
      } catch {
        /* keep polling */
      }
    }, 1000)

    return () => {
      cancelled = true
      window.clearInterval(interval)
    }
  }, [logCleanupActive, logCleanupTaskId, t])

  const onSubmit = async (values: LogSettingsFormValues) => {
    if (values.LogConsumeEnabled !== defaultEnabled) {
      await updateOption.mutateAsync({
        key: 'LogConsumeEnabled',
        value: values.LogConsumeEnabled,
      })
    }
    if (values.InflightTaskCleanupRule !== defaultInflightTaskCleanupRule) {
      await updateOption.mutateAsync({
        key: 'InflightTaskCleanupRule',
        value: values.InflightTaskCleanupRule,
      })
    }
    if (
      values.InflightTaskCleanupIntervalMinutes !==
      defaultInflightTaskCleanupIntervalMinutes
    ) {
      await updateOption.mutateAsync({
        key: 'InflightTaskCleanupIntervalMinutes',
        value: values.InflightTaskCleanupIntervalMinutes,
      })
    }
    if (
      values.InflightTaskCleanupScheduleMode !==
      defaultInflightTaskCleanupScheduleMode
    ) {
      await updateOption.mutateAsync({
        key: 'InflightTaskCleanupScheduleMode',
        value: values.InflightTaskCleanupScheduleMode,
      })
    }
    if (values.InflightTaskCleanupCron !== defaultInflightTaskCleanupCron) {
      await updateOption.mutateAsync({
        key: 'InflightTaskCleanupCron',
        value: values.InflightTaskCleanupCron,
      })
    }
    if (values.InflightTaskTraceEnabled !== defaultInflightTaskTraceEnabled) {
      await updateOption.mutateAsync({
        key: 'InflightTaskTraceEnabled',
        value: values.InflightTaskTraceEnabled,
      })
    }
    if (
      values.InflightTaskTraceMenuVisible !==
      defaultInflightTaskTraceMenuVisible
    ) {
      await updateOption.mutateAsync({
        key: 'InflightTaskTraceMenuVisible',
        value: values.InflightTaskTraceMenuVisible,
      })
    }
    if (
      values.InflightTaskTraceMaxRequestBytes !==
      defaultInflightTaskTraceMaxRequestBytes
    ) {
      await updateOption.mutateAsync({
        key: 'InflightTaskTraceMaxRequestBytes',
        value: values.InflightTaskTraceMaxRequestBytes,
      })
    }
    if (
      values.InflightTaskTraceMaxResponseBytes !==
      defaultInflightTaskTraceMaxResponseBytes
    ) {
      await updateOption.mutateAsync({
        key: 'InflightTaskTraceMaxResponseBytes',
        value: values.InflightTaskTraceMaxResponseBytes,
      })
    }
    for (const [key, value, defaultValue] of [
      ['InflightTaskTraceStorageMode', values.InflightTaskTraceStorageMode, defaultInflightTaskTraceStorageMode],
      ['InflightTaskTraceArchiveChannelID', values.InflightTaskTraceArchiveChannelID, defaultInflightTaskTraceArchiveChannelID],
      ['InflightTaskTraceArchiveThresholdBytes', values.InflightTaskTraceArchiveThresholdBytes, defaultInflightTaskTraceArchiveThresholdBytes],
      ['InflightTaskTraceArchiveRetentionYears', values.InflightTaskTraceArchiveRetentionYears, defaultInflightTaskTraceArchiveRetentionYears],
      ['InflightTaskTraceArchiveRetentionMonths', values.InflightTaskTraceArchiveRetentionMonths, defaultInflightTaskTraceArchiveRetentionMonths],
      ['InflightTaskTraceArchiveRetentionDays', values.InflightTaskTraceArchiveRetentionDays, defaultInflightTaskTraceArchiveRetentionDays],
      ['InflightTaskTraceArchiveRetentionHours', values.InflightTaskTraceArchiveRetentionHours, defaultInflightTaskTraceArchiveRetentionHours],
    ] as const) {
      if (value !== defaultValue) await updateOption.mutateAsync({ key, value })
    }
    if (values.InflightTracePath !== defaultInflightTracePath) {
      await updateOption.mutateAsync({
        key: 'performance_setting.inflight_trace_path',
        value: values.InflightTracePath,
      })
    }
  }

  const handleRequestCleanLogs = () => {
    if (!purgeTimestamp) {
      toast.error(t('Select a timestamp before clearing logs.'))
      return
    }

    setShowConfirmDialog(true)
  }

  const handleCleanLogs = async () => {
    if (!purgeTimestamp) {
      toast.error(t('Select a timestamp before clearing logs.'))
      return
    }

    setIsStartingLogCleanup(true)
    try {
      const res = await startInflightLogCleanupTask(purgeTimestamp)
      if (!res.success) {
        throw new Error(res.message || t('Failed to clean logs'))
      }
      if (!res.data) {
        throw new Error(t('Failed to clean logs'))
      }
      setLogCleanupTask(res.data)
      setShowConfirmDialog(false)
      toast.success(t('Inflight log cleanup task started.'))
      const statsRes = await getInflightTaskStats()
      if (statsRes.success && statsRes.data) {
        setInflightTaskStats(statsRes.data)
      }
    } catch (error) {
      const message =
        error instanceof Error ? error.message : t('Failed to clean logs')
      toast.error(message)
    } finally {
      setIsStartingLogCleanup(false)
    }
  }

  const cleanupServerLogFiles = async () => {
    if (
      !serverLogCleanupValue ||
      Number.isNaN(serverLogCleanupValue) ||
      serverLogCleanupValue < 1
    ) {
      toast.error(t('Please enter a valid number'))
      return
    }

    setServerLogCleanupLoading(true)
    try {
      const res = await api.delete(
        `/api/performance/logs?mode=${serverLogCleanupMode}&value=${serverLogCleanupValue}`
      )
      if (res.data.success) {
        const { deleted_count, freed_bytes } = res.data.data
        toast.success(
          t('Cleaned up {{count}} log files, freed {{size}}', {
            count: deleted_count,
            size: formatBytes(freed_bytes),
          })
        )
      } else {
        toast.error(res.data.message || t('Cleanup failed'))
      }
      fetchServerLogInfo()
    } catch {
      toast.error(t('Cleanup failed'))
    } finally {
      setServerLogCleanupLoading(false)
    }
  }

  const triggerArchiveUpload = async () => {
    try {
      const res = await triggerInflightTraceArchiveUploads()
      if (!res.success) throw new Error(res.message)
      toast.success(t('Archive upload started.'))
      const statsRes = await getInflightTaskStats()
      if (statsRes.success && statsRes.data) setInflightTaskStats(statsRes.data)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('Failed to start archive upload'))
    }
  }

  return (
    <SettingsSection title={t('Log Maintenance')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save log settings'
          />
          <FormField
            control={form.control}
            name='LogConsumeEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Record quota usage')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Track per-request consumption to power usage analytics. Keeping this on increases database writes.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
                <FormMessage />
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='InflightTaskCleanupRule'
            render={({ field }) => (
              <SettingsControlGroup className='flex flex-wrap items-center justify-between gap-4'>
                <div className='min-w-0'>
                  <FormLabel>{t('Inflight log cleanup rule')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Choose whether terminal inflight logs are cleaned automatically.'
                    )}
                  </FormDescription>
                </div>
                <FormControl>
                  <Select
                    items={[
                      { value: '1', label: t('Do not clean') },
                      { value: '2', label: t('Clean terminal logs') },
                    ]}
                    value={String(field.value)}
                    onValueChange={(value) => {
                      if (value !== null) field.onChange(Number(value))
                    }}
                  >
                    <SelectTrigger className='w-[220px]'>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent alignItemWithTrigger={false}>
                      <SelectGroup>
                        <SelectItem value='1'>{t('Do not clean')}</SelectItem>
                        <SelectItem value='2'>
                          {t('Clean terminal logs')}
                        </SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormMessage className='w-full' />
              </SettingsControlGroup>
            )}
          />
          <InflightCleanupScheduleFields
            form={
              form as unknown as UseFormReturn<ScheduleFormValues>
            }
            enabled={form.watch('InflightTaskCleanupRule') === 2}
          />
          <Separator />
          <SettingsControlGroup className='grid gap-3'>
            <div>
              <h4 className='text-sm font-medium'>
                {t('Inflight Debug Log')}
              </h4>
              <p className='text-muted-foreground text-sm'>
                {t(
                  'Inflight debug logs may contain sensitive data and increase Redis usage.'
                )}
              </p>
            </div>
            <FormField
              control={form.control}
              name='InflightTaskTraceEnabled'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Record inflight debug logs')}</FormLabel>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                  <FormMessage />
                </SettingsSwitchItem>
              )}
            />
            <FormField
              control={form.control}
              name='InflightTaskTraceMenuVisible'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Show debug log menu item')}</FormLabel>
                    <FormDescription>
                      {t('Debug log menu requires recording to be enabled.')}
                    </FormDescription>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                      disabled={!form.watch('InflightTaskTraceEnabled')}
                    />
                  </FormControl>
                  <FormMessage />
                </SettingsSwitchItem>
              )}
            />
            <FormField
              control={form.control}
              name='InflightTaskTraceStorageMode'
              render={({ field }) => (
                <SettingsControlGroup className='flex flex-wrap items-center justify-between gap-4'>
                  <div className='min-w-0'>
                    <FormLabel>{t('Debug log body storage')}</FormLabel>
                    <FormDescription>{t('Store request and response bodies in memory or per-user CSV files.')}</FormDescription>
                  </div>
                  <FormControl>
                    <Select
                      items={[
                        { value: 'memory', label: t('Memory') },
                        { value: 'disk', label: t('Disk CSV') },
                      ]}
                      value={field.value}
                      onValueChange={(value) => value && field.onChange(value)}
                    >
                      <SelectTrigger className='w-[220px]'><SelectValue /></SelectTrigger>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectItem value='memory'>{t('Memory')}</SelectItem>
                        <SelectItem value='disk'>{t('Disk CSV')}</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormControl>
                  {field.value === 'memory' ? <Alert className='w-full'><AlertDescription>{t('Memory storage increases Redis usage.')}</AlertDescription></Alert> : null}
                </SettingsControlGroup>
              )}
            />
            {form.watch('InflightTaskTraceStorageMode') === 'disk' ? (
              <>
                <FormField control={form.control} name='InflightTaskTraceArchiveChannelID' render={({ field }) => (
                  <SettingsControlGroup className='flex flex-wrap items-center justify-between gap-4'>
                    <FormLabel>{t('Archive storage channel')}</FormLabel>
                    <FormControl><Select items={archiveChannels.map((channel) => ({ value: String(channel.id), label: channel.name }))} value={String(field.value || '')} onValueChange={(value) => field.onChange(Number(value))}><SelectTrigger className='w-[320px]'><SelectValue /></SelectTrigger><SelectContent alignItemWithTrigger={false}>{archiveChannels.map((channel) => <SelectItem key={channel.id} value={String(channel.id)}>{channel.name}</SelectItem>)}</SelectContent></Select></FormControl>
                    <FormMessage className='w-full' />
                  </SettingsControlGroup>
                )} />
                <FormField control={form.control} name='InflightTaskTraceArchiveThresholdBytes' render={({ field }) => (
                  <SettingsControlGroup className='flex flex-wrap items-center justify-between gap-4'><FormLabel>{t('Archive upload threshold (KB)')}</FormLabel><FormControl><Input type='number' min={1} step='any' className='w-[220px]' value={field.value / 1024} onChange={(event) => field.onChange(Math.round(event.currentTarget.valueAsNumber * 1024))} /></FormControl><FormMessage className='w-full' /></SettingsControlGroup>
                )} />
                <FormField control={form.control} name='InflightTracePath' render={({ field }) => (
                  <SettingsControlGroup className='flex flex-wrap items-center justify-between gap-4'>
                    <div className='min-w-0'>
                      <FormLabel>{t('Inflight trace directory')}</FormLabel>
                      <FormDescription>{t('Stores disk-mode inflight trace CSV archives separately from request-body cache files.')}</FormDescription>
                    </div>
                    <FormControl><Input className='w-[320px] max-w-full' placeholder={t('Leave empty to use the disk cache directory')} {...field} /></FormControl>
                    <FormMessage className='w-full' />
                  </SettingsControlGroup>
                )} />
                <SettingsControlGroup className='flex flex-wrap items-center justify-between gap-4'>
                  <div className='min-w-0'>
                    <FormLabel>{t('Archive retention')}</FormLabel>
                    <FormDescription>{t('Set all values to 0 to keep archives permanently.')}</FormDescription>
                  </div>
                  <div className='grid max-w-2xl gap-3 sm:grid-cols-4'>
                    {([
                      ['InflightTaskTraceArchiveRetentionYears', t('Years')],
                      ['InflightTaskTraceArchiveRetentionMonths', t('Months')],
                      ['InflightTaskTraceArchiveRetentionDays', t('Days')],
                      ['InflightTaskTraceArchiveRetentionHours', t('Hours')],
                    ] as const).map(([name, label]) => (
                      <div key={name} className='grid gap-1'>
                        <Label>{label}</Label>
                        <Input type='number' min={0} {...form.register(name, { valueAsNumber: true })} />
                      </div>
                    ))}
                  </div>
                </SettingsControlGroup>
              </>
            ) : null}
            <FormField
              control={form.control}
              name='InflightTaskTraceMaxRequestBytes'
              render={({ field }) => (
                <SettingsControlGroup className='flex flex-wrap items-center justify-between gap-4'>
                  <div className='min-w-0'>
                    <FormLabel>
                      {t('Max inflight debug log request size (KB)')}
                    </FormLabel>
                    <FormDescription>{t('0 means no extra limit')}</FormDescription>
                  </div>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      className='w-[200px]'
                      {...field}
                      value={field.value / 1024}
                      onChange={(event) =>
                        field.onChange(
                          Math.round(event.currentTarget.valueAsNumber * 1024)
                        )
                      }
                    />
                  </FormControl>
                  <FormMessage className='w-full' />
                </SettingsControlGroup>
              )}
            />
            <FormField
              control={form.control}
              name='InflightTaskTraceMaxResponseBytes'
              render={({ field }) => (
                <SettingsControlGroup className='flex flex-wrap items-center justify-between gap-4'>
                  <div className='min-w-0'>
                    <FormLabel>
                      {t('Max inflight debug log response size (KB)')}
                    </FormLabel>
                    <FormDescription>{t('0 means no extra limit')}</FormDescription>
                  </div>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      className='w-[200px]'
                      {...field}
                      value={field.value / 1024}
                      onChange={(event) =>
                        field.onChange(
                          Math.round(event.currentTarget.valueAsNumber * 1024)
                        )
                      }
                    />
                  </FormControl>
                  <FormMessage className='w-full' />
                </SettingsControlGroup>
              )}
            />
          </SettingsControlGroup>
          <SettingsControlGroup className='grid gap-2'>
            <FormLabel>{t('Current inflight log usage')}</FormLabel>
            <FormDescription>
              {t('View inflight log storage across memory, local CSV files, and archived cloud files.')}
            </FormDescription>
            <TooltipProvider>
              <div className='grid gap-2 text-sm md:grid-cols-3'>
              <div className='rounded-md border p-3'>
                <div className='text-muted-foreground text-xs'>
                  {t('User count')}
                </div>
                <div className='font-medium'>
                  {inflightTaskStats?.user_count ?? '-'}
                </div>
              </div>
              {inflightTaskStats?.trace_directory ? (
                <div className='rounded-md border p-3 md:col-span-2'>
                  <div className='text-muted-foreground text-xs'>
                    {t('Inflight trace directory')}
                  </div>
                  <div className='mt-1 break-all font-mono text-xs'>
                    {inflightTaskStats.trace_directory}
                  </div>
                </div>
              ) : null}
              <div className='rounded-md border p-3'>
                <div className='text-muted-foreground flex items-center gap-1 text-xs'>
                  {t('Inflight CSV files')}
                  {(inflightTaskStats?.local_trace_count ?? 0) + (inflightTaskStats?.uploaded_trace_count ?? 0) > 0 ? (
                    <InflightMetricTooltip label={t('Inflight CSV files')}>
                      {inflightTaskStats?.uploaded_trace_count ? <div>{t('Uploaded archives')}: {inflightTaskStats.uploaded_trace_count}</div> : null}
                      {inflightTaskStats?.local_trace_count ? <div>{t('Pending archives')}: {inflightTaskStats.local_trace_count}</div> : null}
                    </InflightMetricTooltip>
                  ) : null}
                </div>
                <div className='font-medium'>
                  {inflightTaskStats ? inflightTaskStats.local_trace_count + inflightTaskStats.uploaded_trace_count : '-'}
                </div>
              </div>
              <div className='rounded-md border p-3'>
                <div className='text-muted-foreground flex items-center gap-1 text-xs'>
                  {t('Inflight CSV size (KB)')}
                  {(inflightTaskStats?.local_trace_size ?? 0) + (inflightTaskStats?.uploaded_trace_size ?? 0) > 0 ? (
                    <InflightMetricTooltip label={t('Inflight CSV size (KB)')}>
                      {inflightTaskStats?.local_trace_size ? <div>{t('Retained local CSV size (KB)')}: {formatKilobytes(inflightTaskStats.local_trace_size)}</div> : null}
                      {inflightTaskStats?.uploaded_trace_size ? <div>{t('Archived cloud CSV size (KB)')}: {formatKilobytes(inflightTaskStats.uploaded_trace_size)}</div> : null}
                    </InflightMetricTooltip>
                  ) : null}
                </div>
                <div className='font-medium'>
                  {formatKilobytes((inflightTaskStats?.local_trace_size ?? 0) + (inflightTaskStats?.uploaded_trace_size ?? 0))}
                </div>
              </div>
              <div className='rounded-md border p-3'>
                <div className='text-muted-foreground text-xs'>
                  {t('Pending archive uploads')}
                </div>
                <div className='flex items-center justify-between gap-2 font-medium'>
                  <span>{inflightTaskStats?.trace_pending_upload_count ?? '-'}</span>
                  <Button type='button' size='sm' variant='outline' onClick={triggerArchiveUpload} disabled={!inflightTaskStats?.trace_pending_upload_count || archiveUploadInProgress}>
                    {t('Upload pending archives')}
                  </Button>
                </div>
              </div>
              <div className='rounded-md border p-3'>
                <div className='text-muted-foreground flex items-center gap-1 text-xs'>
                  {t('Inflight log entries')}
                  {(inflightTaskStats?.item_count ?? 0) > 0 ? (
                    <InflightMetricTooltip label={t('Inflight log entries')}>
                      {inflightTaskStats?.in_memory_count ? <div>{t('Memory-only records')}: {inflightTaskStats.in_memory_count} {t('records')}</div> : null}
                      {inflightTaskStats?.local_trace_count ? <div>{t('Local storage')}: {inflightTaskStats.local_trace_count} {t('records')}</div> : null}
                      {inflightTaskStats?.uploaded_trace_count ? <div>{t('Uploaded cloud archives')}: {inflightTaskStats.uploaded_trace_count} {t('records')}</div> : null}
                    </InflightMetricTooltip>
                  ) : null}
                </div>
                <div className='font-medium'>
                  {inflightTaskStats?.item_count ?? '-'}
                </div>
              </div>
              <div className='rounded-md border p-3'>
                <div className='text-muted-foreground flex items-center gap-1 text-xs'>
                  {t('Total inflight log size')}
                  {(inflightTaskStats?.total_size ?? 0) > 0 ? (
                    <InflightMetricTooltip label={t('Total inflight log size')}>
                      {inflightTaskStats?.in_memory_size ? <div>{t('Memory-only records')}: {formatKilobytes(inflightTaskStats.in_memory_size)}</div> : null}
                      {inflightTaskStats?.local_trace_size ? <div>{t('Local storage')}: {formatKilobytes(inflightTaskStats.local_trace_size)}</div> : null}
                      {inflightTaskStats?.uploaded_trace_size ? <div>{t('Uploaded cloud archives')}: {formatKilobytes(inflightTaskStats.uploaded_trace_size)}</div> : null}
                    </InflightMetricTooltip>
                  ) : null}
                </div>
                <div className='font-medium'>
                  {formatKilobytes(inflightTaskStats?.total_size ?? 0)}
                </div>
              </div>
              </div>
            </TooltipProvider>
          </SettingsControlGroup>

          <SettingsControlGroup className='space-y-3'>
            <div>
              <h4 className='text-sm font-medium'>
                {t('Clean inflight logs')}
              </h4>
              <p className='text-muted-foreground text-sm'>
                  {t(
                    'Remove terminal inflight logs and completed CSV archives updated before the selected timestamp.'
                  )}
              </p>
            </div>
            <DateTimePicker value={purgeDate} onChange={setPurgeDate} />
            <div className='flex flex-wrap gap-3'>
              {quickSelectOptions.map((option) => (
                <Button
                  key={option.label}
                  type='button'
                  variant='outline'
                  onClick={() => setPurgeDate(option.getValue())}
                >
                  {t(option.label)}
                </Button>
              ))}
              <Button
                type='button'
                variant='destructive'
                onClick={handleRequestCleanLogs}
                disabled={isStartingLogCleanup || logCleanupActive}
              >
                {isStartingLogCleanup || logCleanupActive
                  ? t('Cleaning...')
                  : t('Clean logs')}
              </Button>
            </div>
            {logCleanupTask && (
              <div className='rounded-md border p-3'>
                <div className='mb-2 flex items-center justify-between gap-3 text-sm'>
                  <span className='font-medium'>
                    {t('Inflight log cleanup progress')}
                  </span>
                  <span className='text-muted-foreground tabular-nums'>
                    {logCleanupProgress}%
                  </span>
                </div>
                <Progress value={logCleanupProgress} />
                <div className='text-muted-foreground mt-2 text-xs'>
                  {t('{{processed}} of {{total}} inflight logs processed.', {
                    processed: logCleanupProcessed,
                    total: logCleanupTotal,
                  })}
                </div>
                {logCleanupTask.status === 'failed' && logCleanupTask.error && (
                  <div className='text-destructive mt-2 text-xs'>
                    {logCleanupTask.error}
                  </div>
                )}
              </div>
            )}
          </SettingsControlGroup>
        </SettingsForm>
      </Form>

      <Separator />

      <div className='space-y-4'>
        <div>
          <h4 className='font-medium'>{t('Server Log Management')}</h4>
          <p className='text-muted-foreground mt-1 text-xs'>
            {t(
              'Manage server log files. Log files accumulate over time; regular cleanup is recommended to free disk space.'
            )}
          </p>
        </div>

        {serverLogInfo !== null &&
          (serverLogInfo.enabled ? (
            <div className='space-y-4'>
              <div className='rounded-lg border p-4'>
                <div className='grid grid-cols-2 gap-2 text-sm md:grid-cols-4'>
                  <div>
                    <span className='text-muted-foreground'>
                      {t('Log Directory')}:
                    </span>{' '}
                    <span className='font-mono text-xs'>
                      {serverLogInfo.log_dir}
                    </span>
                  </div>
                  <div>
                    <span className='text-muted-foreground'>
                      {t('Log File Count')}:
                    </span>{' '}
                    {serverLogInfo.file_count}
                  </div>
                  <div>
                    <span className='text-muted-foreground'>
                      {t('Total Log Size')}:
                    </span>{' '}
                    {formatBytes(serverLogInfo.total_size)}
                  </div>
                  {serverLogInfo.oldest_time && serverLogInfo.newest_time && (
                    <div>
                      <span className='text-muted-foreground'>
                        {t('Date Range')}:
                      </span>{' '}
                      {dayjs(serverLogInfo.oldest_time).format('YYYY-MM-DD')} ~{' '}
                      {dayjs(serverLogInfo.newest_time).format('YYYY-MM-DD')}
                    </div>
                  )}
                </div>
              </div>

              <div className='flex flex-wrap items-end gap-3'>
                <div className='grid gap-1.5'>
                  <Label className='text-xs'>{t('Cleanup Mode')}</Label>
                  <Select
                    items={[
                      { value: 'by_count', label: t('Retain last N files') },
                      { value: 'by_days', label: t('Retain last N days') },
                    ]}
                    value={serverLogCleanupMode}
                    onValueChange={(value) =>
                      value !== null && setServerLogCleanupMode(value)
                    }
                  >
                    <SelectTrigger className='w-[160px]'>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent alignItemWithTrigger={false}>
                      <SelectGroup>
                        <SelectItem value='by_count'>
                          {t('Retain last N files')}
                        </SelectItem>
                        <SelectItem value='by_days'>
                          {t('Retain last N days')}
                        </SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </div>
                <div className='grid gap-1.5'>
                  <Label className='text-xs'>
                    {serverLogCleanupMode === 'by_count'
                      ? t('Files to Retain')
                      : t('Days to Retain')}
                  </Label>
                  <Input
                    type='number'
                    min={1}
                    max={serverLogCleanupMode === 'by_count' ? 1000 : 3650}
                    value={serverLogCleanupValue}
                    onChange={(event) =>
                      setServerLogCleanupValue(Number(event.target.value))
                    }
                    className='w-[120px]'
                  />
                </div>
                <AlertDialog>
                  <AlertDialogTrigger
                    render={
                      <Button
                        type='button'
                        variant='destructive'
                        size='sm'
                        disabled={serverLogCleanupLoading}
                      />
                    }
                  >
                    {serverLogCleanupLoading
                      ? t('Cleaning...')
                      : t('Clean Up Log Files')}
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>
                        {t('Confirm log file cleanup?')}
                      </AlertDialogTitle>
                      <AlertDialogDescription>
                        {serverLogCleanupMode === 'by_count'
                          ? t(
                              'Only the last {{value}} log files will be retained; the rest will be deleted.',
                              {
                                value: serverLogCleanupValue,
                              }
                            )
                          : t(
                              'Log files older than {{value}} days will be deleted.',
                              {
                                value: serverLogCleanupValue,
                              }
                            )}
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>{t('Cancel')}</AlertDialogCancel>
                      <AlertDialogAction
                        variant='destructive'
                        onClick={cleanupServerLogFiles}
                      >
                        {t('Confirm Cleanup')}
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            </div>
          ) : (
            <Alert>
              <AlertDescription>
                {t(
                  'Server logging is not enabled (log directory not configured)'
                )}
              </AlertDescription>
            </Alert>
          ))}
      </div>

      <AlertDialog open={showConfirmDialog} onOpenChange={setShowConfirmDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Confirm log cleanup')}</AlertDialogTitle>
            <AlertDialogDescription>
              {formattedPurgeDate
                ? t(
                    'This will permanently remove terminal inflight logs updated before {{date}}.',
                    { date: formattedPurgeDate }
                  )
                : t(
                    'This will permanently remove terminal inflight logs before the selected timestamp.'
                  )}{' '}
              {t('This action cannot be undone.')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isStartingLogCleanup}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant='destructive'
              onClick={handleCleanLogs}
              disabled={isStartingLogCleanup}
            >
              {isStartingLogCleanup ? t('Cleaning...') : t('Delete logs')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SettingsSection>
  )
}
