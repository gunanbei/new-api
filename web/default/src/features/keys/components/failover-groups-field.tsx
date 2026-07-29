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
import { ArrowDown, ArrowUp, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

import { MAX_FAILOVER_GROUPS } from '../constants'
import {
  ApiKeyGroupCombobox,
  type ApiKeyGroupOption,
} from './api-key-group-combobox'

type FailoverGroupsFieldProps = {
  options: ApiKeyGroupOption[]
  value: string[]
  onChange: (groups: string[]) => void
  /** Position numbers are hidden when the strategy ignores the configured order. */
  showOrder: boolean
}

export function FailoverGroupsField(props: FailoverGroupsFieldProps) {
  const { t } = useTranslation()

  // The auto group is itself a group sequence, so nesting it inside a failover
  // sequence has no meaning and the backend rejects it.
  const candidates = props.options.filter(
    (option) => option.value !== 'auto' && !props.value.includes(option.value)
  )
  const reachedLimit = props.value.length >= MAX_FAILOVER_GROUPS

  const move = (index: number, direction: -1 | 1) => {
    const target = index + direction
    if (target < 0 || target >= props.value.length) return
    const next = [...props.value]
    ;[next[index], next[target]] = [next[target], next[index]]
    props.onChange(next)
  }

  return (
    <div className='flex flex-col gap-2'>
      {props.value.length > 0 && (
        <ul className='flex flex-col gap-2'>
          {props.value.map((group, index) => {
            const option = props.options.find((item) => item.value === group)
            return (
              <li
                key={group}
                className='bg-muted/40 flex items-center gap-2 rounded-lg border px-3 py-2'
              >
                {props.showOrder && (
                  <span className='text-muted-foreground w-5 shrink-0 text-xs tabular-nums'>
                    {index + 1}
                  </span>
                )}
                <span className='flex min-w-0 flex-1 flex-col gap-0.5'>
                  <span className='truncate text-sm font-medium'>{group}</span>
                  {option?.crossGroupRetryStatusCodes && (
                    <span className='text-muted-foreground line-clamp-2 text-[11px] leading-4 break-all sm:text-xs'>
                      {t(
                        'Automatic cross-group HTTP status codes: [{{codes}}]',
                        {
                          codes: option.crossGroupRetryStatusCodes,
                        }
                      )}
                    </span>
                  )}
                </span>
                {option?.ratio !== undefined && option.ratio !== '' && (
                  <Badge variant='outline' className='shrink-0 text-[10px]'>
                    {`${option.ratio}x`}
                  </Badge>
                )}
                {!option && (
                  <Badge variant='outline' className='shrink-0 text-[10px]'>
                    {t('Unavailable')}
                  </Badge>
                )}
                <div className='flex shrink-0 items-center'>
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon'
                    className='size-7'
                    disabled={!props.showOrder || index === 0}
                    aria-label={t('Move up')}
                    onClick={() => move(index, -1)}
                  >
                    <ArrowUp className='size-3.5' />
                  </Button>
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon'
                    className='size-7'
                    disabled={
                      !props.showOrder || index === props.value.length - 1
                    }
                    aria-label={t('Move down')}
                    onClick={() => move(index, 1)}
                  >
                    <ArrowDown className='size-3.5' />
                  </Button>
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon'
                    className='text-muted-foreground hover:text-destructive size-7'
                    aria-label={t('Remove')}
                    onClick={() =>
                      props.onChange(props.value.filter((_, i) => i !== index))
                    }
                  >
                    <Trash2 className='size-3.5' />
                  </Button>
                </div>
              </li>
            )
          })}
        </ul>
      )}
      <ApiKeyGroupCombobox
        options={candidates}
        value=''
        onValueChange={(group) => props.onChange([...props.value, group])}
        placeholder={
          reachedLimit
            ? t('Reached the maximum of {{max}} groups', {
                max: MAX_FAILOVER_GROUPS,
              })
            : t('Add a group')
        }
        disabled={reachedLimit || candidates.length === 0}
      />
    </div>
  )
}
