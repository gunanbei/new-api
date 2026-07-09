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
import { useEffect, useMemo, useState } from 'react'
import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

import { previewInflightCleanupCron } from '../api'
import {
  SettingsControlGroup,
  SettingsFormGrid,
} from '../components/settings-form-layout'
import {
  buildCronFromSimple,
  DEFAULT_INFLIGHT_CRON_SIMPLE_CONFIG,
  INFLIGHT_CLEANUP_INTERVAL_MAX,
  INFLIGHT_CLEANUP_INTERVAL_MIN,
  normalizeInflightCronSimpleConfig,
  parseCronToSimple,
  type InflightCronSimpleConfig,
} from './inflight-cleanup-cron'

export type ScheduleFormValues = {
  InflightTaskCleanupScheduleMode: 'interval' | 'cron'
  InflightTaskCleanupIntervalMinutes: number
  InflightTaskCleanupCron: string
}

type InflightCleanupScheduleFieldsProps = {
  form: UseFormReturn<ScheduleFormValues>
  enabled: boolean
}

const WEEKDAY_OPTIONS = [
  { value: 0, labelKey: 'Sunday' },
  { value: 1, labelKey: 'Monday' },
  { value: 2, labelKey: 'Tuesday' },
  { value: 3, labelKey: 'Wednesday' },
  { value: 4, labelKey: 'Thursday' },
  { value: 5, labelKey: 'Friday' },
  { value: 6, labelKey: 'Saturday' },
] as const

const settingsFieldClassName = 'grid gap-2'
const settingsInputClassName = 'h-9 w-full max-w-[220px]'
const settingsCodeClassName =
  'bg-background text-foreground border-border inline-block max-w-full rounded-md border px-2.5 py-1.5 font-mono text-xs'

export function InflightCleanupScheduleFields(
  props: InflightCleanupScheduleFieldsProps
) {
  const { form, enabled } = props
  const { t } = useTranslation()
  const scheduleMode = form.watch('InflightTaskCleanupScheduleMode')
  const cronExpr = form.watch('InflightTaskCleanupCron')
  const [simpleConfig, setSimpleConfig] = useState<InflightCronSimpleConfig>(
    () =>
      parseCronToSimple(cronExpr) ?? {
        ...DEFAULT_INFLIGHT_CRON_SIMPLE_CONFIG,
      }
  )
  const [previewTimezone, setPreviewTimezone] = useState('')
  const [previewRuns, setPreviewRuns] = useState<number[]>([])
  const [previewError, setPreviewError] = useState('')

  useEffect(() => {
    const parsed = parseCronToSimple(cronExpr)
    if (parsed) {
      setSimpleConfig(parsed)
    }
  }, [cronExpr])

  useEffect(() => {
    if (!enabled || scheduleMode !== 'cron') {
      return
    }
    const nextCron = buildCronFromSimple(
      normalizeInflightCronSimpleConfig(simpleConfig)
    )
    if (nextCron !== cronExpr) {
      form.setValue('InflightTaskCleanupCron', nextCron, {
        shouldDirty: true,
        shouldValidate: true,
      })
    }
  }, [cronExpr, enabled, form, scheduleMode, simpleConfig])

  useEffect(() => {
    if (!enabled || scheduleMode !== 'cron' || !cronExpr.trim()) {
      setPreviewTimezone('')
      setPreviewRuns([])
      setPreviewError('')
      return
    }

    let cancelled = false
    const timer = window.setTimeout(async () => {
      try {
        const res = await previewInflightCleanupCron(cronExpr)
        if (cancelled) {
          return
        }
        if (!res.success || !res.data) {
          setPreviewTimezone('')
          setPreviewRuns([])
          setPreviewError(res.message || t('Invalid cron expression'))
          return
        }
        setPreviewTimezone(res.data.timezone)
        setPreviewRuns(res.data.next_runs)
        setPreviewError('')
      } catch {
        if (!cancelled) {
          setPreviewTimezone('')
          setPreviewRuns([])
          setPreviewError(t('Invalid cron expression'))
        }
      }
    }, 300)

    return () => {
      cancelled = true
      window.clearTimeout(timer)
    }
  }, [cronExpr, enabled, scheduleMode, t])

  const generatedCron = useMemo(
    () => buildCronFromSimple(normalizeInflightCronSimpleConfig(simpleConfig)),
    [simpleConfig]
  )

  const updateSimpleConfig = (patch: Partial<InflightCronSimpleConfig>) => {
    setSimpleConfig((current) =>
      normalizeInflightCronSimpleConfig({ ...current, ...patch })
    )
  }

  const timeValue = `${String(simpleConfig.hour).padStart(2, '0')}:${String(
    simpleConfig.minute
  ).padStart(2, '0')}`

  if (!enabled) {
    return null
  }

  return (
    <>
      <FormField
        control={form.control}
        name='InflightTaskCleanupScheduleMode'
        render={({ field }) => (
          <SettingsControlGroup className={settingsFieldClassName}>
            <FormLabel>{t('Inflight log cleanup schedule mode')}</FormLabel>
            <FormDescription>
              {t('Choose a fixed interval or a cron schedule for cleanup scans.')}
            </FormDescription>
            <FormControl>
              <Select
                items={[
                  { value: 'interval', label: t('Fixed interval') },
                  { value: 'cron', label: t('Cron schedule') },
                ]}
                value={field.value}
                onValueChange={(value) => {
                  if (value === 'interval' || value === 'cron') {
                    field.onChange(value)
                  }
                }}
              >
                <SelectTrigger className={settingsInputClassName}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    <SelectItem value='interval'>{t('Fixed interval')}</SelectItem>
                    <SelectItem value='cron'>{t('Cron schedule')}</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </FormControl>
            <FormMessage />
          </SettingsControlGroup>
        )}
      />

      {scheduleMode === 'interval' ? (
        <FormField
          control={form.control}
          name='InflightTaskCleanupIntervalMinutes'
          render={({ field }) => (
            <SettingsControlGroup className={settingsFieldClassName}>
              <FormLabel>
                {t('Inflight log cleanup interval (minutes)')}
              </FormLabel>
              <FormDescription>
                {t(
                  'How often the cron cleanup scans terminal inflight logs.'
                )}
              </FormDescription>
              <FormControl>
                <Input
                  type='number'
                  min={INFLIGHT_CLEANUP_INTERVAL_MIN}
                  max={INFLIGHT_CLEANUP_INTERVAL_MAX}
                  className={cn(settingsInputClassName, 'max-w-[160px]')}
                  {...field}
                  onChange={(event) =>
                    field.onChange(event.currentTarget.valueAsNumber)
                  }
                />
              </FormControl>
              <FormDescription>
                {t('Allowed range: {{min}}-{{max}} minutes (up to 7 days)', {
                  min: INFLIGHT_CLEANUP_INTERVAL_MIN,
                  max: INFLIGHT_CLEANUP_INTERVAL_MAX,
                })}
              </FormDescription>
              <FormMessage />
            </SettingsControlGroup>
          )}
        />
      ) : (
        <FormField
          control={form.control}
          name='InflightTaskCleanupCron'
          render={() => (
            <FormItem>
              <SettingsControlGroup className='grid gap-4'>
                <div className='grid gap-1'>
                  <FormLabel>{t('Cron schedule')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Execution times use the server local timezone. Preview below follows the server clock.'
                    )}
                  </FormDescription>
                </div>

                <SettingsFormGrid className='gap-4 lg:grid-cols-2'>
                  <div className={settingsFieldClassName}>
                    <FormLabel>{t('Schedule preset')}</FormLabel>
                    <Select
                      items={[
                        { value: 'daily', label: t('Daily') },
                        { value: 'weekly', label: t('Weekly') },
                        {
                          value: 'hourly_interval',
                          label: t('Every N hours'),
                        },
                      ]}
                      value={simpleConfig.preset}
                      onValueChange={(value) => {
                        if (
                          value === 'daily' ||
                          value === 'weekly' ||
                          value === 'hourly_interval'
                        ) {
                          updateSimpleConfig({ preset: value })
                        }
                      }}
                    >
                      <SelectTrigger className={settingsInputClassName}>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          <SelectItem value='daily'>{t('Daily')}</SelectItem>
                          <SelectItem value='weekly'>{t('Weekly')}</SelectItem>
                          <SelectItem value='hourly_interval'>
                            {t('Every N hours')}
                          </SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                  </div>

                  {simpleConfig.preset === 'weekly' ? (
                    <div className={settingsFieldClassName}>
                      <FormLabel>{t('Weekday')}</FormLabel>
                      <Select
                        items={WEEKDAY_OPTIONS.map((option) => ({
                          value: String(option.value),
                          label: t(option.labelKey),
                        }))}
                        value={String(simpleConfig.weekday)}
                        onValueChange={(value) => {
                          if (value !== null) {
                            updateSimpleConfig({ weekday: Number(value) })
                          }
                        }}
                      >
                        <SelectTrigger className={settingsInputClassName}>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent alignItemWithTrigger={false}>
                          <SelectGroup>
                            {WEEKDAY_OPTIONS.map((option) => (
                              <SelectItem
                                key={option.value}
                                value={String(option.value)}
                              >
                                {t(option.labelKey)}
                              </SelectItem>
                            ))}
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                    </div>
                  ) : null}

                  {simpleConfig.preset === 'hourly_interval' ? (
                    <div className={settingsFieldClassName}>
                      <FormLabel>{t('Hour interval')}</FormLabel>
                      <Input
                        type='number'
                        min={1}
                        max={23}
                        className={cn(settingsInputClassName, 'max-w-[160px]')}
                        value={simpleConfig.hourInterval}
                        onChange={(event) =>
                          updateSimpleConfig({
                            hourInterval: event.currentTarget.valueAsNumber,
                          })
                        }
                      />
                    </div>
                  ) : (
                    <div className={settingsFieldClassName}>
                      <FormLabel>{t('Time')}</FormLabel>
                      <Input
                        type='time'
                        className={cn(settingsInputClassName, 'max-w-[160px]')}
                        value={timeValue}
                        onChange={(event) => {
                          const [hour, minute] = event.currentTarget.value
                            .split(':')
                            .map((part) => Number(part))
                          updateSimpleConfig({ hour, minute })
                        }}
                      />
                    </div>
                  )}
                </SettingsFormGrid>

                <div className={settingsFieldClassName}>
                  <FormLabel>{t('Generated cron expression')}</FormLabel>
                  <code className={settingsCodeClassName}>{generatedCron}</code>
                </div>

                {previewError ? (
                  <Alert variant='destructive'>
                    <AlertDescription>{previewError}</AlertDescription>
                  </Alert>
                ) : previewRuns.length > 0 ? (
                  <div className='bg-background/60 border-border space-y-2 rounded-md border p-3'>
                    <p className='text-sm font-medium'>
                      {t('Next 6 scheduled runs')}
                    </p>
                    {previewTimezone ? (
                      <p className='text-muted-foreground text-xs'>
                        {t('Server timezone: {{timezone}}', {
                          timezone: previewTimezone,
                        })}
                      </p>
                    ) : null}
                    <ul className='text-muted-foreground space-y-1 font-mono text-xs'>
                      {previewRuns.map((timestamp) => (
                        <li key={timestamp}>
                          {formatTimestampToDate(timestamp, 'seconds')}
                        </li>
                      ))}
                    </ul>
                  </div>
                ) : null}
              </SettingsControlGroup>
              <FormMessage />
            </FormItem>
          )}
        />
      )}
    </>
  )
}
