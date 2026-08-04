/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { Plus, Trash2 } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

export type GroupAutoDisableRule = {
  group: string
  enabled: boolean
  disable_threshold_seconds: number
  http_status_codes: string
  failure_keywords: string
}

type GroupAutoDisableRulesFieldProps = {
  groups: string[]
  isLoading: boolean
  value: GroupAutoDisableRule[]
  getError: (
    index: number,
    field:
      | 'http_status_codes'
      | 'disable_threshold_seconds'
      | 'failure_keywords'
  ) => string | undefined
  onChange: (rules: GroupAutoDisableRule[]) => void
}

export function GroupAutoDisableRulesField(
  props: GroupAutoDisableRulesFieldProps
) {
  const { t } = useTranslation()
  const availableGroups = useMemo(
    () =>
      props.groups.filter(
        (group) =>
          group !== 'auto' && !props.value.some((rule) => rule.group === group)
      ),
    [props.groups, props.value]
  )

  const addRule = (group: string | null) => {
    if (!group || !availableGroups.includes(group)) return
    props.onChange([
      ...props.value,
      {
        group,
        enabled: true,
        disable_threshold_seconds: 0,
        http_status_codes: '',
        failure_keywords: '',
      },
    ])
  }

  return (
    <div className='flex min-w-0 flex-col gap-3'>
      {props.value.map((rule, index) => (
        <div
          key={rule.group || `rule-${index}`}
          className='flex min-w-0 flex-col gap-3 border-b pb-3 last:border-b-0 last:pb-0'
        >
          <div className='grid min-w-0 gap-3 md:grid-cols-[minmax(8rem,12rem)_minmax(10rem,1fr)_minmax(10rem,15rem)_auto] md:items-end'>
            <div className='flex min-w-0 flex-col gap-1'>
              <span className='text-muted-foreground text-xs font-medium'>
                {t('Group')}
              </span>
              <div className='border-input bg-muted/30 flex h-9 min-w-0 items-center gap-2 rounded-md border px-2.5 text-sm font-medium'>
                <span className='truncate'>{rule.group}</span>
                {!props.groups.includes(rule.group) && (
                  <span className='text-destructive shrink-0 text-xs font-normal'>
                    {t('Unavailable')}
                  </span>
                )}
              </div>
            </div>
            <div className='flex flex-col gap-1'>
              <span className='text-muted-foreground text-xs font-medium'>
                {t('Enable group auto-disable')}
              </span>
              <Switch
                checked={rule.enabled}
                aria-label={t('Enable group auto-disable')}
                onCheckedChange={(enabled) => {
                  const next = [...props.value]
                  next[index] = { ...rule, enabled }
                  props.onChange(next)
                }}
              />
            </div>
            <div className='flex min-w-0 flex-col gap-1'>
              <span className='text-muted-foreground text-xs font-medium'>
                {t('Group disable threshold (seconds)')}
              </span>
              <Input
                type='number'
                min={0}
                step={1}
                value={rule.disable_threshold_seconds}
                aria-label={t('Group disable threshold (seconds)')}
                aria-invalid={
                  Boolean(props.getError(index, 'disable_threshold_seconds')) ||
                  undefined
                }
                onChange={(event) => {
                  const value = event.target.valueAsNumber
                  const next = [...props.value]
                  next[index] = {
                    ...rule,
                    disable_threshold_seconds: Number.isFinite(value)
                      ? value
                      : 0,
                  }
                  props.onChange(next)
                }}
              />
              {props.getError(index, 'disable_threshold_seconds') && (
                <p className='text-destructive text-sm'>
                  {t(props.getError(index, 'disable_threshold_seconds') ?? '')}
                </p>
              )}
            </div>
            <Button
              type='button'
              variant='ghost'
              size='icon'
              className='text-muted-foreground hover:text-destructive'
              aria-label={t('Remove')}
              onClick={() =>
                props.onChange(
                  props.value.filter((_, itemIndex) => itemIndex !== index)
                )
              }
            >
              <Trash2 className='size-4' />
            </Button>
          </div>
          <div className='grid min-w-0 gap-3 md:grid-cols-2'>
            <div className='flex min-w-0 flex-col gap-1'>
              <span className='text-muted-foreground text-xs font-medium'>
                {t('Group auto-disable HTTP status codes')}
              </span>
              <Input
                value={rule.http_status_codes}
                placeholder='401,429,500-599'
                aria-label={t('Group auto-disable HTTP status codes')}
                aria-invalid={
                  Boolean(props.getError(index, 'http_status_codes')) ||
                  undefined
                }
                onChange={(event) => {
                  const next = [...props.value]
                  next[index] = {
                    ...rule,
                    http_status_codes: event.target.value,
                  }
                  props.onChange(next)
                }}
              />
              {props.getError(index, 'http_status_codes') && (
                <p className='text-destructive text-sm'>
                  {t(props.getError(index, 'http_status_codes') ?? '')}
                </p>
              )}
            </div>
            <div className='flex min-w-0 flex-col gap-1'>
              <span className='text-muted-foreground text-xs font-medium'>
                {t('Group failure keywords')}
              </span>
              <Textarea
                rows={3}
                value={rule.failure_keywords}
                placeholder={t('one keyword per line')}
                aria-label={t('Group failure keywords')}
                aria-invalid={
                  Boolean(props.getError(index, 'failure_keywords')) ||
                  undefined
                }
                onChange={(event) => {
                  const next = [...props.value]
                  next[index] = {
                    ...rule,
                    failure_keywords: event.target.value,
                  }
                  props.onChange(next)
                }}
              />
              {props.getError(index, 'failure_keywords') && (
                <p className='text-destructive text-sm'>
                  {t(props.getError(index, 'failure_keywords') ?? '')}
                </p>
              )}
            </div>
          </div>
        </div>
      ))}

      <Select
        disabled={props.isLoading || availableGroups.length === 0}
        onValueChange={addRule}
      >
        <SelectTrigger className='w-full justify-center border-dashed'>
          <Plus className='size-4' />
          <span>{t('Add group rule')}</span>
        </SelectTrigger>
        <SelectContent alignItemWithTrigger={false}>
          <SelectGroup>
            {availableGroups.map((group) => (
              <SelectItem key={group} value={group}>
                {group}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </div>
  )
}
