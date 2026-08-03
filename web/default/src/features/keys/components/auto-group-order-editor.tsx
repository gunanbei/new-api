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
import { ArrowDown, ArrowUp, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

type AutoGroupOrderEditorProps = {
  value: string[]
  options: string[]
  maxCount: number
  onChange: (value: string[]) => void
}

export function AutoGroupOrderEditor({
  value,
  options,
  maxCount,
  onChange,
}: AutoGroupOrderEditorProps) {
  const { t } = useTranslation()
  const availableOptions = options.filter(
    (group) => group !== 'auto' && !value.includes(group)
  )
  const atLimit = value.length >= maxCount

  const move = (index: number, direction: -1 | 1) => {
    const nextIndex = index + direction
    if (nextIndex < 0 || nextIndex >= value.length) return

    const next = [...value]
    ;[next[index], next[nextIndex]] = [next[nextIndex], next[index]]
    onChange(next)
  }

  return (
    <div className='space-y-2'>
      <Select
        items={availableOptions.map((group) => ({ value: group, label: group }))}
        disabled={atLimit || availableOptions.length === 0}
        onValueChange={(group) => {
          if (!group || atLimit || value.includes(group)) return
          onChange([...value, group])
        }}
      >
        <SelectTrigger>
          <SelectValue placeholder={t('Add Auto group')} />
        </SelectTrigger>
        <SelectContent alignItemWithTrigger={false}>
          <SelectGroup>
            {availableOptions.map((group) => (
              <SelectItem key={group} value={group}>
                {group}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>

      {value.map((group, index) => (
        <div
          key={group}
          className='flex min-h-9 items-center gap-2 border px-2 py-1.5'
        >
          <span className='min-w-0 flex-1 truncate text-sm'>
            {index + 1}. {group}
          </span>
          <div className='flex shrink-0 gap-1'>
            <Button
              type='button'
              size='icon'
              variant='ghost'
              aria-label={t('Move {{group}} up', { group })}
              disabled={index === 0}
              onClick={() => move(index, -1)}
            >
              <ArrowUp className='size-4' aria-hidden='true' />
            </Button>
            <Button
              type='button'
              size='icon'
              variant='ghost'
              aria-label={t('Move {{group}} down', { group })}
              disabled={index === value.length - 1}
              onClick={() => move(index, 1)}
            >
              <ArrowDown className='size-4' aria-hidden='true' />
            </Button>
            <Button
              type='button'
              size='icon'
              variant='ghost'
              aria-label={t('Remove {{group}}', { group })}
              onClick={() =>
                onChange(value.filter((item) => item !== group))
              }
            >
              <X className='size-4' aria-hidden='true' />
            </Button>
          </div>
        </div>
      ))}
    </div>
  )
}
