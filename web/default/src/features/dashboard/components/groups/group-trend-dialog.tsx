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
import { VChart } from '@visactor/react-vchart'
import { ChartColumn, ChartLine } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  formatLatency,
  formatThroughput,
  formatUptimePct,
  getSuccessRateColor,
} from '@/features/performance-metrics/lib/format'
import type { PerfGroupMonitor } from '@/features/performance-metrics/types'
import { useChartTheme } from '@/lib/use-chart-theme'
import { VCHART_OPTION } from '@/lib/vchart'

import {
  GROUP_TREND_METRIC_OPTIONS,
  getRelativeTrendColor,
  type GroupTrendMetric,
} from './group-trend-metric'

type ChartType = 'bar' | 'line'

type GroupTrendDialogProps = {
  group: PerfGroupMonitor | null
  metric: GroupTrendMetric
  averageTtft: number
  averageTps: number
  open: boolean
  onMetricChange: (metric: GroupTrendMetric) => void
  onOpenChange: (open: boolean) => void
}

function getValueField(
  metric: GroupTrendMetric,
  chartType: ChartType
): 'barSuccessRate' | 'successRate' | 'tps' | 'ttft' {
  if (metric === 'availability') {
    return chartType === 'bar' ? 'barSuccessRate' : 'successRate'
  }
  return metric === 'tps' ? 'tps' : 'ttft'
}

function getMetricColor(
  metric: GroupTrendMetric,
  ttft: number,
  tps: number,
  averageTtft: number,
  averageTps: number
): string {
  if (metric === 'tps') return getRelativeTrendColor(tps, averageTps)
  return getRelativeTrendColor(ttft, averageTtft)
}

export function GroupTrendDialog(props: GroupTrendDialogProps) {
  const { t } = useTranslation()
  const { resolvedTheme, themeReady } = useChartTheme()
  const [chartType, setChartType] = useState<ChartType>('line')
  useEffect(() => {
    if (props.group) {
      setChartType(props.metric === 'availability' ? 'bar' : 'line')
    }
  }, [props.group, props.metric])
  const trendData = useMemo(
    () =>
      (props.group?.series ?? [])
        .filter((point) => Number.isFinite(point.ts))
        .sort((left, right) => left.ts - right.ts)
        .map((point) => ({
          time: new Date(point.ts * 1000).toLocaleString(),
          successRate: Math.min(100, Math.max(0, point.success_rate)),
          // Keep zero-percent failures visible in the bar chart while the
          // tooltip continues to report the actual success rate.
          barSuccessRate:
            point.success_rate <= 0 ? 1 : Math.min(100, point.success_rate),
          ttft: Math.max(0, point.avg_ttft_ms),
          tps: Math.max(0, point.avg_tps),
        })),
    [props.group]
  )
  const chartSpec = useMemo(() => {
    const isAvailability = props.metric === 'availability'
    const isTps = props.metric === 'tps'
    const valueField = getValueField(props.metric, chartType)
    const metricLabel = t(
      GROUP_TREND_METRIC_OPTIONS.find((option) => option.value === props.metric)
        ?.labelKey ?? 'Monitoring trend'
    )
    const formatValue = (value: number) => {
      if (isAvailability) return formatUptimePct(value)
      if (isTps) return formatThroughput(value)
      return formatLatency(value)
    }
    const tooltip = {
      title: { value: (datum: { time: string }) => datum.time },
      content: [
        {
          key: metricLabel,
          value: (datum: {
            successRate: number
            tps: number
            ttft: number
          }) => {
            if (isAvailability) return formatUptimePct(datum.successRate)
            if (isTps) return formatThroughput(datum.tps)
            return formatLatency(datum.ttft)
          },
        },
      ],
    }
    const commonSpec = {
      data: [{ id: 'group-success-rate', values: trendData }],
      xField: 'time',
      yField: valueField,
      legends: { visible: false },
      tooltip: {
        mark: tooltip,
        dimension: tooltip,
      },
      axes: [
        {
          orient: 'bottom',
          label: {
            style: { fill: resolvedTheme === 'dark' ? '#cbd5e1' : '#475569' },
            autoHide: true,
            autoLimit: true,
          },
          tick: { visible: false },
        },
        {
          orient: 'left',
          ...(isAvailability ? { min: 0, max: 100 } : { min: 0 }),
          label: {
            formatMethod: (value: number | string) =>
              formatValue(Number(value)),
            style: { fill: resolvedTheme === 'dark' ? '#cbd5e1' : '#475569' },
          },
          grid: {
            visible: true,
            style: {
              lineDash: [3, 3],
              stroke:
                resolvedTheme === 'dark'
                  ? 'rgba(255, 255, 255, 0.12)'
                  : 'rgba(15, 23, 42, 0.12)',
            },
          },
        },
      ],
    }

    if (chartType === 'bar') {
      return {
        ...commonSpec,
        type: 'bar' as const,
        bar: {
          style: {
            fill: (datum: { successRate: number; ttft: number; tps: number }) =>
              isAvailability
                ? getSuccessRateColor(datum.successRate)
                : getMetricColor(
                    props.metric,
                    datum.ttft,
                    datum.tps,
                    props.averageTtft,
                    props.averageTps
                  ),
          },
        },
      }
    }

    return {
      ...commonSpec,
      type: 'line' as const,
      line: {
        style: {
          stroke: '#10b981',
          lineWidth: 2.5,
        },
      },
      point: {
        visible: true,
        style: {
          fill: (datum: { successRate: number; ttft: number; tps: number }) =>
            isAvailability
              ? getSuccessRateColor(datum.successRate)
              : getMetricColor(
                  props.metric,
                  datum.ttft,
                  datum.tps,
                  props.averageTtft,
                  props.averageTps
                ),
          stroke: '#ffffff',
          lineWidth: 1.5,
          size: 5,
        },
      },
    }
  }, [
    chartType,
    props.averageTps,
    props.averageTtft,
    props.metric,
    resolvedTheme,
    t,
    trendData,
  ])
  const metricLabel = t(
    GROUP_TREND_METRIC_OPTIONS.find((option) => option.value === props.metric)
      ?.labelKey ?? 'Monitoring trend'
  )

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Monitoring trend')}
      description={`${t('Group')}: ${props.group?.group ?? '—'}`}
      contentClassName='sm:max-w-5xl'
      contentHeight='min(72vh, 640px)'
      bodyClassName='flex h-full flex-col'
    >
      <div className='flex flex-wrap items-center gap-2'>
        <Select
          items={GROUP_TREND_METRIC_OPTIONS.map((option) => ({
            value: option.value,
            label: t(option.labelKey),
          }))}
          value={props.metric}
          onValueChange={(value) =>
            props.onMetricChange(value as GroupTrendMetric)
          }
        >
          <SelectTrigger size='sm' aria-label={t('Monitoring trend')}>
            <SelectValue>{metricLabel}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {GROUP_TREND_METRIC_OPTIONS.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {t(option.labelKey)}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        <Tabs
          value={chartType}
          onValueChange={(value) => setChartType(value as ChartType)}
        >
          <TabsList>
            <TabsTrigger value='line'>
              <ChartLine className='size-3.5' />
              {t('Line chart')}
            </TabsTrigger>
            <TabsTrigger value='bar'>
              <ChartColumn className='size-3.5' />
              {t('Bar chart')}
            </TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      <div className='min-h-80 flex-1 pt-4'>
        {trendData.length === 0 ? (
          <div className='text-muted-foreground flex h-full items-center justify-center rounded-lg border text-sm'>
            {t('No monitoring trend data')}
          </div>
        ) : (
          themeReady && (
            <VChart
              key={`${props.group?.group}-${chartType}-${resolvedTheme}`}
              spec={{
                ...chartSpec,
                theme: resolvedTheme === 'dark' ? 'dark' : 'light',
                background: 'transparent',
              }}
              option={VCHART_OPTION}
            />
          )
        )}
      </div>
    </Dialog>
  )
}
