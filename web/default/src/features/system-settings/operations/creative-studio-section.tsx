import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { InformationCircleIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'

import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Field, FieldDescription, FieldGroup, FieldLabel } from '@/components/ui/field'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { api } from '@/lib/api'
import { getEnabledModels } from '@/features/channels/api'

import { SettingsSection } from '../components/settings-section'

type CreativeModel = { id: number; model_name: string; model_key: string; display_name: string; vendor: string; description: string; status: string; sort_order: number }
type Capability = { id: number; model_id: number; category: string; operation: string; asset_kind: string; protocol: string; execution_mode: string; input_schema: string; default_params: string; enabled: boolean; sort_order: number }
type Publication = { id: number; capability_id: number; group_name: string; enabled: boolean; sort_order: number; group_default_params: string }
type Binding = { id: number; publication_id: number; channel_id: number; request_model: string; priority: number; enabled: boolean; validation_status: string; validation_message: string }
type Bootstrap = {
  models: CreativeModel[]
  capabilities: Record<string, Capability[]>
  publications: Record<string, Publication[]>
  bindings: Record<string, Binding[]>
  groups: Record<string, string>
  channels: { id: number; name: string; status: number }[]
  file_channels: { id: number; name: string; type: string; status: string }[]
  settings: { default_file_channel_id: number }
}

const vendorItems = ['OpenAI', 'Anthropic', 'Google', 'Stability AI', 'Midjourney', 'FAL', 'Replicate', 'Ideogram', 'Black Forest Labs', 'Other'].map((value) => ({ label: value, value }))
const protocolItems = ['openai_images', 'openai_responses_image_generation', 'gemini_imagen', 'fal', 'replicate', 'stability', 'midjourney', 'ideogram', 'flux', 'claude_svg', 'custom'].map((value) => ({ label: value, value }))

type Editor =
  | { kind: 'model'; value?: CreativeModel }
  | { kind: 'capability'; modelID: number; value?: Capability }
  | { kind: 'publication'; capabilityID: number; value?: Publication }
  | { kind: 'binding'; publicationID: number; value?: Binding }
  | null

async function getBootstrap(): Promise<Bootstrap> {
  const response = await api.get<{ success: boolean; message: string; data: Bootstrap }>('/api/creative/admin/bootstrap')
  if (!response.data.success) throw new Error(response.data.message)
  return response.data.data
}

function requireSuccess(response: { data: { success?: boolean; message?: string } }) {
  if (response.data.success === false) throw new Error(response.data.message)
  return response
}

export function CreativeStudioSection() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [editor, setEditor] = useState<Editor>(null)
  const { data, isLoading } = useQuery({ queryKey: ['creative-studio'], queryFn: getBootstrap })
  const { data: enabledModels } = useQuery({ queryKey: ['enabled-models'], queryFn: getEnabledModels })
  const refresh = () => queryClient.invalidateQueries({ queryKey: ['creative-studio'] })
  const updateSettings = useMutation({
    mutationFn: async (defaultFileChannelID: number) => requireSuccess(await api.put('/api/creative/admin/settings', { default_file_channel_id: defaultFileChannelID })),
    onSuccess: () => { toast.success(t('Setting updated successfully')); refresh() },
  })
  const remove = useMutation({
    mutationFn: async (url: string) => requireSuccess(await api.delete(url)),
    onSuccess: () => { toast.success(t('Deleted successfully')); refresh() },
  })

  if (isLoading || !data) return <p className='text-muted-foreground text-sm'>{t('Loading creative studio...')}</p>
  const fileChannels = data.file_channels.filter((channel) => channel.status === '1' && channel.type !== '0')
  const fileChannelItems = [{ label: t('Not configured'), value: '0' }, ...fileChannels.map((channel) => ({ label: channel.name, value: String(channel.id) }))]

  return (
    <SettingsSection title={t('Creative Studio Management')}>
      <div className='space-y-6'>
        <div className='rounded-xl border p-4'>
          <h4 className='mb-2 font-medium'>{t('Default work storage')}</h4>
          <Select items={fileChannelItems} value={String(data.settings.default_file_channel_id)} onValueChange={(value) => updateSettings.mutate(Number(value))}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>{fileChannelItems.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}</SelectContent>
          </Select>
        </div>
        <div className='space-y-3'>
          <div className='flex items-center justify-between'><h4 className='font-medium'>{t('Model catalog')}</h4><Button onClick={() => setEditor({ kind: 'model' })}>{t('Add model')}</Button></div>
          <div className='space-y-3'>{data.models.map((creativeModel) => <ModelTree key={creativeModel.id} creativeModel={creativeModel} data={data} t={t} onEdit={setEditor} onDelete={(url) => remove.mutate(url)} />)}</div>
        </div>
      </div>
      {editor && <CreativeEditor editor={editor} data={data} modelNames={enabledModels?.data ?? []} onClose={() => setEditor(null)} onSaved={refresh} />}
    </SettingsSection>
  )
}

function FieldGrid(props: { fields: Array<[React.ReactNode, React.ReactNode | null | undefined, React.ReactNode?, string?]> }) {
  return <dl className='grid gap-x-6 gap-y-2 text-sm sm:grid-cols-2 xl:grid-cols-3'>{props.fields.map(([label, value, description, key]) => <div key={key ?? (typeof label === 'string' ? label : '')} className='min-w-0'><dt className='flex items-center gap-2 font-medium'>{label}{description && <span className='text-muted-foreground text-xs font-normal'>{description}</span>}</dt>{value !== null && <dd className='text-muted-foreground text-xs break-all'>{value === '' || value === undefined ? '—' : value}</dd>}</div>)}</dl>
}

function statusLabel(t: (key: string) => string, status: string) {
  const key = status.charAt(0).toUpperCase() + status.slice(1)
  return t(key)
}

function ModelTree(props: { creativeModel: CreativeModel; data: Bootstrap; t: (key: string, options?: Record<string, unknown>) => string; onEdit: (editor: Editor) => void; onDelete: (url: string) => void }) {
  const capabilities = props.data.capabilities[String(props.creativeModel.id)] ?? []
  return <details open className='rounded-xl border bg-card'><summary className='cursor-pointer list-none px-4 py-3'><div className='flex flex-wrap items-center justify-between gap-3'><div><p className='font-semibold'>{props.creativeModel.display_name}</p><p className='text-muted-foreground text-sm'>{props.t('Model directory')} · {props.creativeModel.model_name}</p></div><div className='flex gap-2' onClick={(event) => event.stopPropagation()}><Button size='sm' variant='outline' onClick={() => props.onEdit({ kind: 'model', value: props.creativeModel })}>{props.t('Edit')}</Button><Button size='sm' variant='outline' onClick={() => props.onEdit({ kind: 'capability', modelID: props.creativeModel.id })}>{props.t('Add capability')}</Button><Button size='sm' variant='destructive' onClick={() => props.onDelete(`/api/creative/admin/models/${props.creativeModel.id}`)}>{props.t('Delete')}</Button></div></div></summary><div className='space-y-3 border-t p-4'><FieldGrid fields={[[props.t('Model name'), props.creativeModel.model_name, <FieldHelp key='model-name-help' label={props.t('Model name')} content={props.t('The upstream model identifier. It must match the model name configured on a channel.')} />], [props.t('Model key'), props.creativeModel.model_key, <FieldHelp key='model-key-help' label={props.t('Model key')} content={props.t('Generated from the model name after trimming and lowercasing. It is the unique internal key and cannot be edited.')} />], [props.t('Display name'), props.creativeModel.display_name, <FieldHelp key='display-name-help' label={props.t('Display name')} content={props.t('A friendly label for administrators and future user-facing lists. It does not change routing.')} />], [props.t('Vendor'), props.creativeModel.vendor, <FieldHelp key='vendor-help' label={props.t('Vendor')} content={props.t('Classifies the provider for display only. Protocol is configured on the capability; channel routing is configured on the binding.')} />], [props.t('Status'), statusLabel(props.t, props.creativeModel.status), <ModelStatusHelp key='status-help' />], [props.t('Sort order'), props.creativeModel.sort_order, <FieldHelp key='sort-order-help' label={props.t('Sort order')} content={props.t('Lower values appear first. Use 0 for the default order.')} />], [props.t('Description'), props.creativeModel.description, <FieldHelp key='description-help' label={props.t('Description')} content={props.t('Optional administrative notes. It does not change routing.')} />]]} />{capabilities.map((capability) => <CapabilityTree key={capability.id} capability={capability} creativeModel={props.creativeModel} data={props.data} t={props.t} onEdit={props.onEdit} onDelete={props.onDelete} />)}</div></details>
}

function JsonPreview(props: { label: string; value: string }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  let formattedValue = props.value
  try {
    formattedValue = JSON.stringify(JSON.parse(props.value), null, 2)
  } catch {}
  const button = <button type='button' className='cursor-pointer text-left font-medium' onClick={() => setOpen(true)}>{props.label}</button>
  return <><Tooltip><TooltipTrigger render={button} /><TooltipContent>{t('Click to view full JSON')}</TooltipContent></Tooltip><Dialog open={open} onOpenChange={setOpen}><DialogContent className='max-w-3xl'><DialogHeader><DialogTitle>{props.label}</DialogTitle></DialogHeader><pre className='bg-muted max-h-[60vh] overflow-auto rounded-lg p-3 text-xs break-words whitespace-pre-wrap'>{formattedValue}</pre><DialogFooter><Button variant='outline' onClick={() => setOpen(false)}>{t('Cancel')}</Button></DialogFooter></DialogContent></Dialog></>
}

function capabilityValue(t: (key: string) => string, field: 'category' | 'operation' | 'assetKind' | 'executionMode', value: string) {
  if (field === 'category') {
    return t(value === 'image' ? 'Image' : 'Video')
  }
  if (field === 'operation') {
    if (value === 'generate') return t('Generate')
    if (value === 'edit') return t('Edit')
    return t('Remix')
  }
  if (field === 'assetKind') {
    if (value === 'raster') return t('Raster image')
    if (value === 'vector') return t('Vector graphic')
    return t('Video')
  }
  if (value === 'sync') return t('Synchronous')
  if (value === 'async') return t('Asynchronous')
  return t('Synchronous artifact')
}

function MediaFormatHelp() {
  const { t } = useTranslation()
  const button = <button type='button' className='text-muted-foreground inline-flex size-3.5 items-center justify-center' aria-label={t('Media format')}><HugeiconsIcon icon={InformationCircleIcon} strokeWidth={2} className='size-3.5' /></button>
  return <Tooltip><TooltipTrigger render={button} /><TooltipContent className='max-w-md'><div className='space-y-1.5'><p>{t('The media format of generated results.')}</p><ul className='list-disc space-y-1 pl-4'><li>{t('Raster: standard bitmap images, recommended for most image generation and editing models.')}</li><li>{t('Vector: SVG graphics that scale without loss, only for models that explicitly support SVG output.')}</li><li>{t('Video: video files, only for video capabilities.')}</li></ul><p>{t('Recommendation: choose raster for standard images; choose vector only for explicit SVG output; choose video for video capabilities.')}</p></div></TooltipContent></Tooltip>
}

function ModelStatusHelp() {
  const { t } = useTranslation()
  const button = <button type='button' className='text-muted-foreground inline-flex size-3.5 items-center justify-center' aria-label={t('Status')}><HugeiconsIcon icon={InformationCircleIcon} strokeWidth={2} className='size-3.5' /></button>
  return <Tooltip><TooltipTrigger render={button} /><TooltipContent className='max-w-md'><div className='space-y-1.5'><p>{t('Marks whether this catalog entry is available in Creative Studio. It does not replace capability, publication, or channel enablement.')}</p><ul className='list-disc space-y-1 pl-4'><li>{t('Enabled: keep this model directory entry available for Creative Studio configuration.')}</li><li>{t('Disabled: hide this directory entry from normal use without deleting its configuration.')}</li></ul></div></TooltipContent></Tooltip>
}

function FieldHelp(props: { label: string; content: string }) {
  const button = <button type='button' className='text-muted-foreground inline-flex size-3.5 items-center justify-center' aria-label={props.label}><HugeiconsIcon icon={InformationCircleIcon} strokeWidth={2} className='size-3.5' /></button>
  return <Tooltip><TooltipTrigger render={button} /><TooltipContent className='max-w-md'>{props.content}</TooltipContent></Tooltip>
}

function CapabilityTree(props: { capability: Capability; creativeModel: CreativeModel; data: Bootstrap; t: (key: string, options?: Record<string, unknown>) => string; onEdit: (editor: Editor) => void; onDelete: (url: string) => void }) {
  const publications = props.data.publications[String(props.capability.id)] ?? []
  return <details className='ml-3 rounded-lg border border-dashed'><summary className='cursor-pointer list-none px-3 py-2'><div className='flex flex-wrap items-center justify-between gap-2'><span className='font-medium'>{props.t('Capability')}:{capabilityValue(props.t, 'category', props.capability.category)}/{capabilityValue(props.t, 'operation', props.capability.operation)}｜{props.capability.protocol}</span><div className='flex gap-2' onClick={(event) => event.stopPropagation()}><Button size='sm' variant='outline' onClick={() => props.onEdit({ kind: 'capability', modelID: props.capability.model_id, value: props.capability })}>{props.t('Edit')}</Button><Button size='sm' variant='outline' onClick={() => props.onEdit({ kind: 'publication', capabilityID: props.capability.id })}>{props.t('Add group')}</Button><Button size='sm' variant='destructive' onClick={() => props.onDelete(`/api/creative/admin/capabilities/${props.capability.id}`)}>{props.t('Delete')}</Button></div></div></summary><div className='space-y-3 border-t p-3'><FieldGrid fields={[[props.t('Category'), capabilityValue(props.t, 'category', props.capability.category), <FieldHelp key='category-help' label={props.t('Category')} content={props.t('Defines whether this capability creates images or videos.')} />], [props.t('Operation'), capabilityValue(props.t, 'operation', props.capability.operation), <FieldHelp key='operation-help' label={props.t('Operation')} content={props.t('Defines the action this capability performs.')} />], [props.t('Media format'), capabilityValue(props.t, 'assetKind', props.capability.asset_kind), <MediaFormatHelp key='media-format-help' />], [props.t('Protocol'), props.capability.protocol, <FieldHelp key='protocol-help' label={props.t('Protocol')} content={props.t('Identifies the upstream request protocol. It is part of this capability’s unique identity.')} />], [props.t('Execution mode'), capabilityValue(props.t, 'executionMode', props.capability.execution_mode), <FieldHelp key='execution-mode-help' label={props.t('Execution mode')} content={props.t('Describes whether the result is returned immediately, later, or as a synchronous artifact.')} />], [props.t('Status'), props.capability.enabled ? props.t('Enabled') : props.t('Disabled'), <FieldHelp key='capability-status-help' label={props.t('Status')} content={props.t('Enables this capability for later publication and routing.')} />], [props.t('Sort order'), props.capability.sort_order, <FieldHelp key='capability-sort-help' label={props.t('Sort order')} content={props.t('Lower values appear first. Use 0 for the default order.')} />], [<JsonPreview key='input-schema' label={props.t('Input schema')} value={props.capability.input_schema} />, null, <FieldHelp key='input-schema-help' label={props.t('Input schema')} content={props.t('JSON object that describes supported inputs.')} />, 'input-schema'], [<JsonPreview key='default-parameters' label={props.t('Default parameters')} value={props.capability.default_params} />, null, <FieldHelp key='default-parameters-help' label={props.t('Default parameters')} content={props.t('JSON object applied before group defaults.')} />, 'default-parameters']]} />{publications.map((publication) => <PublicationTree key={publication.id} publication={publication} capability={props.capability} creativeModel={props.creativeModel} data={props.data} t={props.t} onEdit={props.onEdit} onDelete={props.onDelete} />)}</div></details>
}

function PublicationTree(props: { publication: Publication; capability: Capability; creativeModel: CreativeModel; data: Bootstrap; t: (key: string, options?: Record<string, unknown>) => string; onEdit: (editor: Editor) => void; onDelete: (url: string) => void }) {
  const bindings = props.data.bindings[String(props.publication.id)] ?? []
  return <details className='ml-3 rounded-lg border border-dashed'><summary className='cursor-pointer list-none px-3 py-2'><div className='flex flex-wrap items-center justify-between gap-2'><span className='font-medium'>{props.t('Group')}: {props.publication.group_name}</span><div className='flex gap-2' onClick={(event) => event.stopPropagation()}><Button size='sm' variant='outline' onClick={() => props.onEdit({ kind: 'publication', capabilityID: props.publication.capability_id, value: props.publication })}>{props.t('Edit')}</Button><Button size='sm' variant='outline' onClick={() => props.onEdit({ kind: 'binding', publicationID: props.publication.id })}>{props.t('Group configuration')}</Button><Button size='sm' variant='destructive' onClick={() => props.onDelete(`/api/creative/admin/publications/${props.publication.id}`)}>{props.t('Delete')}</Button></div></div></summary><div className='space-y-3 border-t p-3'><FieldGrid fields={[[props.t('Group'), props.publication.group_name, <FieldHelp key='group-help' label={props.t('Group')} content={props.t('Selects the user group that can see this capability.')} />], [props.t('Status'), props.publication.enabled ? props.t('Enabled') : props.t('Disabled'), <FieldHelp key='publication-status-help' label={props.t('Status')} content={props.t('Controls whether this group publication is available to users.')} />], [props.t('Sort order'), props.publication.sort_order, <FieldHelp key='publication-sort-help' label={props.t('Sort order')} content={props.t('Lower values appear first. Use 0 for the default order.')} />], [<JsonPreview key='group-default-parameters' label={props.t('Group default parameters')} value={props.publication.group_default_params} />, null, <FieldHelp key='group-default-parameters-help' label={props.t('Group default parameters')} content={props.t('JSON object merged after capability defaults.')} />, 'group-default-parameters']]} />{bindings.map((binding) => <BindingCard key={binding.id} binding={binding} creativeModel={props.creativeModel} data={props.data} t={props.t} onEdit={props.onEdit} onDelete={props.onDelete} />)}</div></details>
}

function BindingCard(props: { binding: Binding; creativeModel: CreativeModel; data: Bootstrap; t: (key: string, options?: Record<string, unknown>) => string; onEdit: (editor: Editor) => void; onDelete: (url: string) => void }) {
  const channel = props.data.channels.find((item) => item.id === props.binding.channel_id)
  return <div className='ml-3 rounded-lg bg-muted/30 p-3'><div className='mb-3 flex flex-wrap items-center justify-between gap-2'><span className='font-medium'>{props.t('Channel routing')}: #{props.binding.channel_id} {channel?.name}</span><div className='flex gap-2'><Button size='sm' variant='outline' onClick={() => props.onEdit({ kind: 'binding', publicationID: props.binding.publication_id, value: props.binding })}>{props.t('Edit')}</Button><Button size='sm' variant='destructive' onClick={() => props.onDelete(`/api/creative/admin/bindings/${props.binding.id}`)}>{props.t('Delete')}</Button></div></div><FieldGrid fields={[[props.t('Channel'), channel?.name, <FieldHelp key='channel-help' label={props.t('Channel')} content={props.t('Selects the existing enabled channel used for this group publication.')} />], [props.t('Channel ID'), props.binding.channel_id, <FieldHelp key='channel-id-help' label={props.t('Channel ID')} content={props.t('The internal ID of the selected channel.')} />], [props.t('Requested model'), props.binding.request_model || props.creativeModel.model_name, <FieldHelp key='requested-model-help' label={props.t('Requested model')} content={props.t('Inherited from the model directory in this stage and sent to the selected channel.')} />], [props.t('Priority'), props.binding.priority, <FieldHelp key='priority-help' label={props.t('Priority')} content={props.t('Higher values are preferred when selecting candidates. This does not alter the channel’s global priority.')} />], [props.t('Status'), props.binding.enabled ? props.t('Enabled') : props.t('Disabled'), <FieldHelp key='binding-status-help' label={props.t('Status')} content={props.t('New bindings remain disabled until protocol verification is available.')} />], [props.t('Validation'), statusLabel(props.t, props.binding.validation_status), <FieldHelp key='validation-help' label={props.t('Validation')} content={props.t('Shows the current basic validation result for this channel binding.')} />], [props.t('Validation message'), props.t(props.binding.validation_message), <FieldHelp key='validation-message-help' label={props.t('Validation message')} content={props.t('Details the latest validation result without exposing credentials or upstream response bodies.')} />]]} /></div>
}

function CreativeEditor(props: { editor: Editor; data: Bootstrap; modelNames: string[]; onClose: () => void; onSaved: () => void }) {
  const { t } = useTranslation()
  const [form, setForm] = useState<Record<string, string>>({})
  const editor = props.editor
  const value = editor?.value
  const selected = (value ?? {}) as Partial<CreativeModel & Capability & Publication & Binding>
  const save = useMutation({
    mutationFn: async () => {
      if (!editor) return
      let payload: Record<string, unknown>
      let endpoint: string
      let updateEndpoint: string
      if (editor.kind === 'model') {
        payload = { model_name: form.model_name ?? selected.model_name ?? '', display_name: form.display_name ?? selected.display_name ?? '', vendor: form.vendor ?? selected.vendor ?? '', description: form.description ?? selected.description ?? '', status: form.status ?? selected.status ?? 'enabled', sort_order: Number(form.sort_order ?? selected.sort_order ?? 0) }
        endpoint = '/api/creative/admin/models'
        updateEndpoint = endpoint
      } else if (editor.kind === 'capability') {
        payload = { category: form.category ?? selected.category ?? 'image', operation: form.operation ?? selected.operation ?? 'generate', asset_kind: form.asset_kind ?? selected.asset_kind ?? 'raster', protocol: form.protocol ?? selected.protocol ?? '', execution_mode: form.execution_mode ?? selected.execution_mode ?? 'sync', input_schema: JSON.parse(form.input_schema ?? selected.input_schema ?? '{}'), default_params: JSON.parse(form.default_params ?? selected.default_params ?? '{}'), enabled: (form.enabled ?? String(selected.enabled ?? true)) === 'true', sort_order: Number(form.sort_order ?? selected.sort_order ?? 0) }
        endpoint = `/api/creative/admin/models/${editor.modelID}/capabilities`
        updateEndpoint = '/api/creative/admin/capabilities'
      } else if (editor.kind === 'publication') {
        payload = { group_name: form.group_name ?? selected.group_name ?? '', enabled: (form.enabled ?? String(selected.enabled ?? true)) === 'true', sort_order: Number(form.sort_order ?? selected.sort_order ?? 0), group_default_params: JSON.parse(form.group_default_params ?? selected.group_default_params ?? '{}') }
        endpoint = `/api/creative/admin/capabilities/${editor.capabilityID}/publications`
        updateEndpoint = '/api/creative/admin/publications'
      } else {
        payload = { channel_id: Number(form.channel_id ?? selected.channel_id ?? 0), request_model: '', priority: Number(form.priority ?? selected.priority ?? 0), enabled: false }
        endpoint = `/api/creative/admin/publications/${editor.publicationID}/bindings`
        updateEndpoint = '/api/creative/admin/bindings'
      }
      const response = value ? await api.put(`${updateEndpoint}/${value.id}`, payload) : await api.post(endpoint, payload)
      return requireSuccess(response)
    },
    onSuccess: () => { toast.success(t('Saved successfully')); props.onSaved(); props.onClose() },
  })
  if (!editor) return null
  const set = (name: string) => (event: React.ChangeEvent<HTMLInputElement>) => setForm((current) => ({ ...current, [name]: event.target.value }))
  let title = t('Channel binding')
  if (editor.kind === 'model') {
    title = t('Model')
  } else if (editor.kind === 'capability') {
    title = t('Capability')
  } else if (editor.kind === 'publication') {
    title = t('Publication')
  }
  const groups = Object.keys(props.data.groups).map((group) => ({ value: group, label: props.data.groups[group] || group }))
  const channels = props.data.channels.map((channel) => ({ value: String(channel.id), label: channel.name }))
  const modelItems = [...new Set([...props.modelNames, selected.model_name].filter(Boolean))].map((value) => ({ label: value, value }))
  const selectedModelName = form.model_name ?? selected.model_name ?? ''
  const modelKey = selectedModelName.trim().toLowerCase()
  const modelStatusItems = ['enabled', 'disabled'].map((value) => ({ label: value === 'enabled' ? t('Enabled') : t('Disabled'), value }))
  const booleanItems = ['true', 'false'].map((value) => ({ label: value === 'true' ? t('Enabled') : t('Disabled'), value }))
  const selectedCategory = form.category ?? selected.category ?? 'image'
  const selectedOperation = form.operation ?? selected.operation ?? 'generate'
  const selectedAssetKind = form.asset_kind ?? selected.asset_kind ?? (selectedCategory === 'video' ? 'video' : 'raster')
  const categoryItems = ['image', 'video'].map((value) => ({ label: capabilityValue(t, 'category', value), value }))
  const operationItems = (selectedCategory === 'video' ? ['generate', 'remix'] : ['generate', 'edit']).map((value) => ({ label: capabilityValue(t, 'operation', value), value }))
  const assetKindItems = (selectedCategory === 'video' ? ['video'] : ['raster', 'vector']).map((value) => ({ label: capabilityValue(t, 'assetKind', value), value }))
  const executionModeItems = ['sync', 'async', 'sync_artifact'].map((value) => ({ label: capabilityValue(t, 'executionMode', value), value }))
  const selectedChannelID = Number(form.channel_id ?? selected.channel_id ?? 0)
  const selectedChannel = channels.find((channel) => Number(channel.value) === selectedChannelID)
  let duplicateMessage: string | undefined
  if (editor.kind === 'model') {
    const modelName = (form.model_name ?? selected.model_name ?? '').trim()
    if (modelName && props.data.models.some((creativeModel) => creativeModel.model_key === modelName.toLowerCase() && creativeModel.id !== selected.id)) {
      duplicateMessage = t('{{type}}: {{name}} already exists', { type: t('Model directory'), name: modelName })
    }
  } else if (editor.kind === 'capability') {
    const protocol = form.protocol ?? selected.protocol ?? ''
    if (protocol && (props.data.capabilities[String(editor.modelID)] ?? []).some((capability) => capability.category === selectedCategory && capability.operation === selectedOperation && capability.protocol === protocol && capability.id !== selected.id)) {
      duplicateMessage = t('{{type}}: {{name}} already exists', { type: t('Capability'), name: `${capabilityValue(t, 'category', selectedCategory)}/${capabilityValue(t, 'operation', selectedOperation)}｜${protocol}` })
    }
  } else if (editor.kind === 'publication') {
    const groupName = form.group_name ?? selected.group_name ?? ''
    if (groupName && (props.data.publications[String(editor.capabilityID)] ?? []).some((publication) => publication.group_name === groupName && publication.id !== selected.id)) {
      duplicateMessage = t('{{type}}: {{name}} already exists', { type: t('Group'), name: groupName })
    }
  } else if (selectedChannelID > 0 && (props.data.bindings[String(editor.publicationID)] ?? []).some((binding) => binding.channel_id === selectedChannelID && binding.id !== selected.id)) {
    duplicateMessage = t('{{type}}: {{name}} already exists', { type: t('Channel'), name: selectedChannel?.label ?? `#${selectedChannelID}` })
  }
  let dialogTitle = value ? t('Edit {{name}}', { name: title }) : t('Add {{name}}', { name: title })
  if (editor.kind === 'publication') dialogTitle = t('Select group')
  return <Dialog open onOpenChange={(open) => !open && props.onClose()}><DialogContent className='max-h-[calc(100dvh-2rem)] max-w-lg grid-rows-[auto_minmax(0,1fr)_auto] overflow-hidden'><DialogHeader><DialogTitle>{dialogTitle}</DialogTitle><DialogDescription>{t('Changes are available to administrators only.')}</DialogDescription></DialogHeader><div className='grid min-h-0 gap-3 overflow-y-auto pr-1'>
    {editor.kind === 'model' && <FieldGroup className='gap-3'><Field><FieldLabel htmlFor='creative-model-name'>{t('Model name')}</FieldLabel><Select items={modelItems} defaultValue={selected.model_name} onValueChange={(modelName) => setForm((current) => ({ ...current, model_name: String(modelName), display_name: current.display_name ?? String(modelName) }))}><SelectTrigger className='w-full' id='creative-model-name'><SelectValue /></SelectTrigger><SelectContent>{modelItems.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}</SelectContent></Select><FieldDescription>{t('The upstream model identifier. It must match the model name configured on a channel.')}</FieldDescription></Field><Field><FieldLabel htmlFor='creative-model-key'>{t('Model key')}</FieldLabel><Input id='creative-model-key' value={modelKey} disabled readOnly /><FieldDescription>{t('Generated from the model name after trimming and lowercasing. It is the unique internal key and cannot be edited.')}</FieldDescription></Field><Field><FieldLabel htmlFor='creative-model-display-name'>{t('Display name')}</FieldLabel><Input defaultValue={selected.display_name} id='creative-model-display-name' onChange={set('display_name')} /><FieldDescription>{t('A friendly label for administrators and future user-facing lists. It does not change routing.')}</FieldDescription></Field><Field><FieldLabel htmlFor='creative-model-vendor'>{t('Vendor')}</FieldLabel><Select items={vendorItems} defaultValue={selected.vendor} onValueChange={(vendor) => setForm((current) => ({ ...current, vendor: String(vendor) }))}><SelectTrigger className='w-full' id='creative-model-vendor'><SelectValue /></SelectTrigger><SelectContent>{vendorItems.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}</SelectContent></Select><FieldDescription>{t('Classifies the provider for display only. Protocol is configured on the capability; channel routing is configured on the binding.')}</FieldDescription></Field><Field><FieldLabel htmlFor='creative-model-description'>{t('Description')}</FieldLabel><Textarea defaultValue={selected.description} id='creative-model-description' onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))} rows={3} /><FieldDescription>{t('Optional administrative notes. It does not change routing.')}</FieldDescription></Field><Field><div className='flex items-center gap-2'><FieldLabel htmlFor='creative-model-status'>{t('Status')}</FieldLabel><ModelStatusHelp /></div><Select items={modelStatusItems} value={form.status ?? selected.status ?? 'enabled'} onValueChange={(status) => setForm((current) => ({ ...current, status: String(status) }))}><SelectTrigger className='w-full' id='creative-model-status'><SelectValue /></SelectTrigger><SelectContent>{modelStatusItems.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}</SelectContent></Select></Field><Field><FieldLabel htmlFor='creative-model-sort-order'>{t('Sort order')}</FieldLabel><Input defaultValue={selected.sort_order ?? 0} id='creative-model-sort-order' onChange={set('sort_order')} type='number' /><FieldDescription>{t('Lower values appear first. Use 0 for the default order.')}</FieldDescription></Field></FieldGroup>}
    {editor.kind === 'capability' && <FieldGroup className='gap-3'><Field><FieldLabel htmlFor='creative-capability-category'>{t('Category')}</FieldLabel><Select items={categoryItems} value={selectedCategory} onValueChange={(category) => setForm((current) => { const nextCategory = String(category); const isVideo = nextCategory === 'video'; const operation = current.operation ?? selected.operation; const assetKind = current.asset_kind ?? selected.asset_kind; return { ...current, category: nextCategory, operation: ((isVideo && (operation === 'generate' || operation === 'remix')) || (!isVideo && (operation === 'generate' || operation === 'edit'))) ? operation : 'generate', asset_kind: isVideo ? 'video' : ((assetKind === 'raster' || assetKind === 'vector') && assetKind) || 'raster' } })}><SelectTrigger className='w-full' id='creative-capability-category'><SelectValue /></SelectTrigger><SelectContent>{categoryItems.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}</SelectContent></Select><FieldDescription>{t('Defines whether this capability creates images or videos.')}</FieldDescription></Field><Field><FieldLabel htmlFor='creative-capability-operation'>{t('Operation')}</FieldLabel><Select items={operationItems} value={selectedOperation} onValueChange={(operation) => setForm((current) => ({ ...current, operation: String(operation) }))}><SelectTrigger className='w-full' id='creative-capability-operation'><SelectValue /></SelectTrigger><SelectContent>{operationItems.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}</SelectContent></Select><FieldDescription>{t('Defines the action this capability performs.')}</FieldDescription></Field><Field><div className='flex items-center gap-2'><FieldLabel htmlFor='creative-capability-asset-kind'>{t('Media format')}</FieldLabel><MediaFormatHelp /></div><Select items={assetKindItems} value={selectedAssetKind} onValueChange={(assetKind) => setForm((current) => ({ ...current, asset_kind: String(assetKind) }))}><SelectTrigger className='w-full' id='creative-capability-asset-kind'><SelectValue /></SelectTrigger><SelectContent>{assetKindItems.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}</SelectContent></Select></Field><Field><FieldLabel htmlFor='creative-capability-protocol'>{t('Protocol')}</FieldLabel><Select items={protocolItems} defaultValue={selected.protocol} onValueChange={(protocol) => setForm((current) => ({ ...current, protocol: String(protocol) }))}><SelectTrigger className='w-full' id='creative-capability-protocol'><SelectValue /></SelectTrigger><SelectContent>{protocolItems.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}</SelectContent></Select><FieldDescription>{t('Identifies the upstream request protocol. It is part of this capability’s unique identity.')}</FieldDescription></Field><Field><FieldLabel htmlFor='creative-capability-execution-mode'>{t('Execution mode')}</FieldLabel><Select items={executionModeItems} defaultValue={selected.execution_mode ?? 'sync'} onValueChange={(executionMode) => setForm((current) => ({ ...current, execution_mode: String(executionMode) }))}><SelectTrigger className='w-full' id='creative-capability-execution-mode'><SelectValue /></SelectTrigger><SelectContent>{executionModeItems.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}</SelectContent></Select><FieldDescription>{t('Describes whether the result is returned immediately, later, or as a synchronous artifact.')}</FieldDescription></Field><Field><FieldLabel htmlFor='creative-capability-input-schema'>{t('Input schema')}</FieldLabel><Textarea defaultValue={selected.input_schema ?? '{}'} id='creative-capability-input-schema' onChange={(event) => setForm((current) => ({ ...current, input_schema: event.target.value }))} rows={3} /><FieldDescription>{t('JSON object that describes supported inputs.')}</FieldDescription></Field><Field><FieldLabel htmlFor='creative-capability-default-parameters'>{t('Default parameters')}</FieldLabel><Textarea defaultValue={selected.default_params ?? '{}'} id='creative-capability-default-parameters' onChange={(event) => setForm((current) => ({ ...current, default_params: event.target.value }))} rows={3} /><FieldDescription>{t('JSON object applied before group defaults.')}</FieldDescription></Field><Field><FieldLabel htmlFor='creative-capability-status'>{t('Status')}</FieldLabel><Select items={booleanItems} value={form.enabled ?? String(selected.enabled ?? true)} onValueChange={(enabled) => setForm((current) => ({ ...current, enabled: String(enabled) }))}><SelectTrigger className='w-full' id='creative-capability-status'><SelectValue /></SelectTrigger><SelectContent>{booleanItems.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}</SelectContent></Select><FieldDescription>{t('Enables this capability for later publication and routing.')}</FieldDescription></Field><Field><FieldLabel htmlFor='creative-capability-sort-order'>{t('Sort order')}</FieldLabel><Input defaultValue={selected.sort_order ?? 0} id='creative-capability-sort-order' onChange={set('sort_order')} type='number' /><FieldDescription>{t('Lower values appear first. Use 0 for the default order.')}</FieldDescription></Field></FieldGroup>}
    {editor.kind === 'publication' && <FieldGroup className='gap-3'><Field><FieldLabel htmlFor='creative-publication-group'>{t('Group')}</FieldLabel><Select items={groups} defaultValue={selected.group_name} onValueChange={(groupName) => setForm((current) => ({ ...current, group_name: String(groupName) }))}><SelectTrigger className='w-full' id='creative-publication-group'><SelectValue /></SelectTrigger><SelectContent>{groups.map((group) => <SelectItem key={group.value} value={group.value}>{group.label}</SelectItem>)}</SelectContent></Select><FieldDescription>{t('Selects the user group that can see this capability.')}</FieldDescription></Field><Field><FieldLabel htmlFor='creative-publication-status'>{t('Status')}</FieldLabel><Select items={booleanItems} value={form.enabled ?? String(selected.enabled ?? true)} onValueChange={(enabled) => setForm((current) => ({ ...current, enabled: String(enabled) }))}><SelectTrigger className='w-full' id='creative-publication-status'><SelectValue /></SelectTrigger><SelectContent>{booleanItems.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}</SelectContent></Select><FieldDescription>{t('Controls whether this group publication is available to users.')}</FieldDescription></Field><Field><FieldLabel htmlFor='creative-publication-sort-order'>{t('Sort order')}</FieldLabel><Input defaultValue={selected.sort_order ?? 0} id='creative-publication-sort-order' onChange={set('sort_order')} type='number' /><FieldDescription>{t('Lower values appear first. Use 0 for the default order.')}</FieldDescription></Field><Field><FieldLabel htmlFor='creative-publication-default-parameters'>{t('Group default parameters')}</FieldLabel><Textarea defaultValue={selected.group_default_params ?? '{}'} id='creative-publication-default-parameters' onChange={(event) => setForm((current) => ({ ...current, group_default_params: event.target.value }))} rows={3} /><FieldDescription>{t('JSON object merged after capability defaults.')}</FieldDescription></Field></FieldGroup>}
    {editor.kind === 'binding' && <FieldGroup className='gap-3'><Field><FieldLabel htmlFor='creative-binding-channel'>{t('Channel')}</FieldLabel><Select items={channels} defaultValue={value ? String(selected.channel_id) : undefined} onValueChange={(channelID) => setForm((current) => ({ ...current, channel_id: String(channelID) }))}><SelectTrigger className='w-full' id='creative-binding-channel'><SelectValue placeholder={t('Channel')} /></SelectTrigger><SelectContent>{channels.map((channel) => <SelectItem key={channel.value} value={channel.value}>{channel.label}</SelectItem>)}</SelectContent></Select><FieldDescription>{t('Selects the existing enabled channel used for this group publication.')}</FieldDescription></Field><Field><FieldLabel htmlFor='creative-binding-request-model'>{t('Requested model')}</FieldLabel><Input id='creative-binding-request-model' value={selected.request_model || selected.model_name || ''} disabled readOnly /><FieldDescription>{t('Inherited from the model directory in this stage and sent to the selected channel.')}</FieldDescription></Field><Field><FieldLabel htmlFor='creative-binding-priority'>{t('Priority')}</FieldLabel><Input defaultValue={selected.priority ?? 0} id='creative-binding-priority' onChange={set('priority')} type='number' /><FieldDescription>{t('Higher values are preferred when selecting candidates. This does not alter the channel’s global priority.')}</FieldDescription></Field><Field><FieldLabel>{t('Status')}</FieldLabel><Input value={t('Disabled')} disabled readOnly /><FieldDescription>{t('New bindings remain disabled until protocol verification is available.')}</FieldDescription></Field></FieldGroup>}
    {duplicateMessage && <Alert className='border-amber-500/40 bg-amber-500/10 text-amber-900 dark:text-amber-100'><AlertDescription>{duplicateMessage}</AlertDescription></Alert>}
  </div><DialogFooter><Button variant='outline' onClick={props.onClose}>{t('Cancel')}</Button><Button onClick={() => save.mutate()} disabled={save.isPending || Boolean(duplicateMessage)}>{t('Save')}</Button></DialogFooter></DialogContent></Dialog>
}
