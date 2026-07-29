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

export type CrossGroupRetryRule = {
  group: string
  http_status_codes: string
}

type CrossGroupRetryRulesFieldProps = {
  groups: string[]
  isLoading: boolean
  value: CrossGroupRetryRule[]
  getError: (index: number) => string | undefined
  onChange: (rules: CrossGroupRetryRule[]) => void
}

export function CrossGroupRetryRulesField(
  props: CrossGroupRetryRulesFieldProps
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
      { group, http_status_codes: '' },
    ])
  }

  return (
    <div className='flex min-w-0 flex-col gap-3'>
      {props.value.length > 0 && (
        <div className='text-muted-foreground hidden min-w-0 gap-3 text-xs font-medium md:grid md:grid-cols-[minmax(10rem,15rem)_minmax(0,1fr)_2.25rem]'>
          <span>{t('Group')}</span>
          <span>{t('HTTP status codes')}</span>
          <span />
        </div>
      )}
      {props.value.map((rule, index) => (
        <div
          key={rule.group || `rule-${index}`}
          className='grid min-w-0 gap-3 md:grid-cols-[minmax(10rem,15rem)_minmax(0,1fr)_auto] md:items-start'
        >
          <div className='border-input bg-muted/30 flex h-9 min-w-0 items-center gap-2 rounded-md border px-2.5 text-sm font-medium'>
            <span className='truncate'>{rule.group}</span>
            {!props.groups.includes(rule.group) && (
              <span className='text-destructive shrink-0 text-xs font-normal'>
                {t('Unavailable')}
              </span>
            )}
          </div>
          <Input
            value={rule.http_status_codes}
            placeholder='429,500-599'
            aria-label={t('Cross-group retry HTTP status codes')}
            aria-invalid={Boolean(props.getError(index)) || undefined}
            onChange={(event) => {
              const next = [...props.value]
              next[index] = {
                ...rule,
                http_status_codes: event.target.value,
              }
              props.onChange(next)
            }}
          />
          {props.getError(index) && (
            <p className='text-destructive text-sm md:col-start-2'>
              {t(props.getError(index) ?? '')}
            </p>
          )}
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
