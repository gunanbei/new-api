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
import { Plus, Trash2 } from 'lucide-react'
import { useEffect, useMemo } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { useQuery } from '@tanstack/react-query'
import i18next from 'i18next'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
import { ComboboxInput, type ComboboxInputOption } from '@/components/ui/combobox'
import {
  Form,
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
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'

import { getEnabledModels } from '@/features/channels/api'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const IMAGE_MODEL_REGISTRY_KEY = 'image_playground.model_registry'

const ADAPTER_OPTIONS = [
  { value: 'openai_images', label: 'OpenAI Images' },
  {
    value: 'openai_responses_image_generation',
    label: 'OpenAI Responses Image Generation',
  },
  { value: 'gemini_imagen', label: 'Google Imagen' },
  { value: 'fal', label: 'FAL' },
  { value: 'replicate', label: 'Replicate' },
  { value: 'stability', label: 'Stability AI' },
  { value: 'midjourney', label: 'Midjourney' },
  { value: 'ideogram', label: 'Ideogram' },
  { value: 'flux', label: 'FLUX' },
  { value: 'custom', label: 'Custom' },
] as const

const registryItemSchema = z.object({
  model_name: z.string().trim().min(1, 'Model name is required'),
  enabled: z.boolean(),
  adapter_type: z.string().trim().min(1, 'Adapter type is required'),
  display_vendor: z.string().trim().min(1, 'Display vendor is required'),
})

function createRegistrySchema() {
  return z
    .object({
      version: z.literal(1),
      items: z.array(registryItemSchema),
    })
    .superRefine((data, ctx) => {
      const seen = new Set<string>()
      data.items.forEach((item, index) => {
        const key = item.model_name.trim().toLowerCase()
        if (!key) return
        if (seen.has(key)) {
          ctx.addIssue({
            code: 'custom',
            message: i18next.t(
              'Duplicate model names are not allowed (case-insensitive)'
            ),
            path: ['items', index, 'model_name'],
          })
          return
        }
        seen.add(key)
      })
    })
}

type ImageModelConfigurationFormValues = z.output<
  ReturnType<typeof createRegistrySchema>
>
type ImageModelConfigurationFormInput = z.input<
  ReturnType<typeof createRegistrySchema>
>

type ImageModelConfigurationSectionProps = {
  defaultValue: string
}

function normalizeRegistryValue(value: string): ImageModelConfigurationFormValues {
  const fallback: ImageModelConfigurationFormValues = {
    version: 1,
    items: [],
  }
  const raw = (value ?? '').trim()
  if (!raw) return fallback
  try {
    const parsed = JSON.parse(raw)
    const result = createRegistrySchema().safeParse(parsed)
    if (result.success) return result.data
  } catch {
    /* empty */
  }
  return fallback
}

function normalizeRegistryItems(values: ImageModelConfigurationFormValues) {
  return values.items.map((item) => ({
    model_name: item.model_name.trim(),
    enabled: item.enabled,
    adapter_type: item.adapter_type.trim(),
    display_vendor: item.display_vendor.trim(),
  }))
}

function stringifyRegistry(values: ImageModelConfigurationFormValues) {
  return JSON.stringify(
    {
      version: 1,
      items: normalizeRegistryItems(values),
    },
    null,
    2
  )
}

function getDefaultVendor(adapterType: string) {
  return (
    ADAPTER_OPTIONS.find((item) => item.value === adapterType)?.label ?? 'Custom'
  )
}

export function ImageModelConfigurationSection(
  props: ImageModelConfigurationSectionProps
) {
  const { t, i18n } = useTranslation()
  const updateOption = useUpdateOption()
  const registrySchema = useMemo(() => createRegistrySchema(), [i18n.language])
  const defaultValues = useMemo(
    () => normalizeRegistryValue(props.defaultValue),
    [props.defaultValue]
  )

  const form = useForm<
    ImageModelConfigurationFormInput,
    unknown,
    ImageModelConfigurationFormValues
  >({
    resolver: zodResolver(registrySchema),
    defaultValues,
  })

  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: 'items',
  })

  useEffect(() => {
    form.reset(defaultValues)
  }, [defaultValues, form])

  const enabledModelsQuery = useQuery({
    queryKey: ['channel-enabled-models'],
    queryFn: getEnabledModels,
    staleTime: 5 * 60 * 1000,
  })

  const modelNameOptions = useMemo<ComboboxInputOption[]>(() => {
    const names = [...new Set((enabledModelsQuery.data?.data ?? []).map((item) => item.trim()).filter(Boolean))]
      .sort((a, b) => a.localeCompare(b, undefined, { sensitivity: 'base' }))
    return names.map((name) => ({ value: name, label: name }))
  }, [enabledModelsQuery.data?.data])

  const onSubmit = async (values: ImageModelConfigurationFormValues) => {
    const nextValue = stringifyRegistry(values)
    const currentValue = stringifyRegistry(defaultValues)
    if (nextValue === currentValue) {
      toast.info(t('No changes to save'))
      return
    }

    await updateOption.mutateAsync({
      key: IMAGE_MODEL_REGISTRY_KEY,
      value: nextValue,
    })
  }

  return (
    <SettingsSection title={t('Image Model Configuration')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />

          <div className='flex flex-col gap-4'>
            {fields.map((field, index) => (
              <div
                key={field.id}
                className='rounded-lg border p-4'
              >
                <div className='mb-4 flex items-center justify-between gap-3'>
                  <div className='min-w-0'>
                    <div className='text-sm font-medium'>
                      {t('Image model')} {index + 1}
                    </div>
                  </div>
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    onClick={() => remove(index)}
                  >
                    <Trash2 className='h-4 w-4' />
                    {t('Delete')}
                  </Button>
                </div>

                <div className='grid gap-4 lg:grid-cols-2'>
                  <FormField
                    control={form.control}
                    name={`items.${index}.model_name`}
                    render={({ field: modelField }) => (
                      <FormItem>
                        <FormLabel>{t('Model Name')}</FormLabel>
                        <FormControl>
                          <ComboboxInput
                            options={modelNameOptions}
                            value={modelField.value}
                            onValueChange={modelField.onChange}
                            allowCustomValue
                            placeholder={t('Select or type a model name')}
                            emptyText={t('No model found')}
                          />
                        </FormControl>
                        <FormDescription>
                          {t(
                            'Candidates come from currently enabled channel models, and manual input is also allowed.'
                          )}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name={`items.${index}.adapter_type`}
                    render={({ field: adapterField }) => (
                      <FormItem>
                        <FormLabel>{t('Adapter Type')}</FormLabel>
                        <Select
                          value={adapterField.value}
                          onValueChange={(value) => {
                            adapterField.onChange(value)
                            const vendorPath = `items.${index}.display_vendor` as const
                            if (!form.getValues(vendorPath).trim()) {
                              form.setValue(vendorPath, getDefaultVendor(value), {
                                shouldDirty: true,
                              })
                            }
                          }}
                        >
                          <FormControl>
                            <SelectTrigger>
                              <SelectValue
                                placeholder={t('Select adapter type')}
                              />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            {ADAPTER_OPTIONS.map((option) => (
                              <SelectItem key={option.value} value={option.value}>
                                {option.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name={`items.${index}.display_vendor`}
                    render={({ field: vendorField }) => (
                      <FormItem>
                        <FormLabel>{t('Display Vendor')}</FormLabel>
                        <FormControl>
                          <Input
                            placeholder={t('Example: OpenAI Images')}
                            {...vendorField}
                          />
                        </FormControl>
                        <FormDescription>
                          {t('Shown as vendor or product name in bootstrap metadata.')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name={`items.${index}.enabled`}
                    render={({ field: enabledField }) => (
                      <SettingsSwitchItem className='rounded-md border px-4 py-3'>
                        <SettingsSwitchContent>
                          <FormLabel>{t('Enabled')}</FormLabel>
                          <FormDescription>
                            {t('Only enabled image models are returned by bootstrap.')}
                          </FormDescription>
                        </SettingsSwitchContent>
                        <FormControl>
                          <Switch
                            checked={enabledField.value}
                            onCheckedChange={enabledField.onChange}
                          />
                        </FormControl>
                      </SettingsSwitchItem>
                    )}
                  />
                </div>
              </div>
            ))}

            {fields.length === 0 && (
              <div className='rounded-lg border border-dashed px-4 py-8 text-center text-sm text-muted-foreground'>
                {t('No image models configured')}
              </div>
            )}
          </div>

          <Separator />

          <div>
            <Button
              type='button'
              variant='outline'
              onClick={() =>
                append({
                  model_name: '',
                  enabled: true,
                  adapter_type: 'openai_images',
                  display_vendor: 'OpenAI Images',
                })
              }
            >
              <Plus className='h-4 w-4' />
              {t('Add image model')}
            </Button>
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
