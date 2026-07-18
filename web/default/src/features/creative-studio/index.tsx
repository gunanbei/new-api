import {
  useMutation,
  useQueries,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import {
  ChevronLeft,
  ChevronRight,
  Copy,
  Download,
  History,
  ImageIcon,
  RefreshCw,
  RotateCcw,
  RotateCw,
  Trash2,
  Upload,
  X,
  ZoomIn,
  ZoomOut,
} from 'lucide-react'
import {
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { api } from '@/lib/api'
import { formatDateTimeObject } from '@/lib/time'
import { cn } from '@/lib/utils'

type Capability = {
  id: number
  model_id: number
  operation: string
  asset_kind: string
  input_schema: string
  default_params?: string
}
type CreativeModel = { id: number; display_name: string; model_name: string }
type Publication = { id: number; group_name: string }
type CreativeTask = {
  task_key: string
  display_name: string
  group_name: string
  model_name?: string
  request_model?: string
  source_channel_name?: string
  source_vendor?: string
  operation?: string
  requested_params?: string
  resolved_params?: string
  started_at?: string
  finished_at?: string
  created_at?: string
  status: string
  error_code?: string
  error_message?: string
}
type CreativeAsset = {
  id: number
  user_file_id: number
  role: string
  position: number
  mime_type: string
  file_name: string
  file_url?: string
  preview_base64?: string
  temporary?: boolean
}
type CreativeTaskDetail = { task: CreativeTask; assets: CreativeAsset[] }
type Envelope<T> = { success: boolean; message: string; data: T }
type FieldOption =
  | string
  | number
  | { label: string; value: string | number; aspect_ratio?: string }
type Field = {
  type?: string
  label?: string
  placeholder?: string
  description?: string
  options?: FieldOption[]
  presentation?: 'segmented' | 'size_cards'
  allow_custom?: boolean
  local_only?: boolean
  multiple?: boolean
  max_items?: number
}
type CapabilityOption = { capability: Capability; creativeModel: CreativeModel }
type Bootstrap = {
  models: CreativeModel[]
  capabilities: Record<string, Capability[]>
  publications: Record<string, Publication[]>
  storage_available: boolean
}
type CreativeOutput = { asset: CreativeAsset; fallbackUrl?: string }
type AssetPreview = { taskKey: string; position?: number }
type ReferenceImage = {
  id: string
  file: File
  url: string
  cacheKey?: string
}

async function getCreativeTaskDetail(
  taskKey: string | undefined,
  signal?: AbortSignal
): Promise<CreativeTaskDetail> {
  const response = await api.get<Envelope<CreativeTaskDetail>>(
    `/api/creative/tasks/${taskKey}`,
    {
      signal,
      skipBusinessError: true,
      skipErrorHandler: true,
    }
  )
  if (!response.data.success) throw new Error(response.data.message)
  return response.data.data
}

const promptExamples = [
  'Cyberpunk city nightscape, neon reflections, cinematic lighting, 8k',
  'A Shiba Inu wearing sunglasses, flat illustration style, vibrant colors',
  'Minimalist product photography: a floating perfume bottle, soft light, beige background',
  'Watercolor Jiangnan town at dawn, thin mist, leave blank space',
]

function parseFields(schema: string, operation: string): Record<string, Field> {
  try {
    const parsed = JSON.parse(schema) as { properties?: Record<string, Field> }
    const fields: Record<string, Field> = {
      prompt: { type: 'textarea' },
      ...(parsed.properties ?? parsed),
    }
    if (operation === 'edit' && !fields.image && !fields.images) {
      fields.image = { type: 'file', multiple: true, max_items: 5 }
    }
    return fields
  } catch {
    return operation === 'edit'
      ? {
          prompt: { type: 'textarea' },
          image: { type: 'file', multiple: true, max_items: 5 },
        }
      : { prompt: { type: 'textarea' } }
  }
}

function statusLabel(t: (key: string) => string, status: string): string {
  return t(
    {
      validating: 'Validating',
      dispatching: 'Generating',
      processing: 'Processing',
      importing: 'Uploading',
      succeeded: 'Succeeded',
      failed: 'Failed',
    }[status] ?? status
  )
}

function statusVariant(
  status: string
): 'secondary' | 'destructive' | 'outline' {
  if (status === 'failed') return 'destructive'
  if (status === 'succeeded') return 'secondary'
  return 'outline'
}

function assetUrl(asset: CreativeAsset): string | undefined {
  if (asset.file_url) return asset.file_url
  if (asset.preview_base64) {
    return `data:${asset.mime_type};base64,${asset.preview_base64}`
  }
  return undefined
}

function taskDownloadResolution(task?: CreativeTask): '1K' | '2K' | '4K' {
  const resolution = taskParams(task).download_resolution
  return resolution === '2K' || resolution === '4K' ? resolution : '1K'
}

async function downloadCreativeAsset(
  taskKey: string,
  asset: CreativeAsset,
  resolution: '1K' | '2K' | '4K'
): Promise<void> {
  const source = asset.id
    ? `data:${asset.mime_type};base64,${
        (
          await api.get<Envelope<{ base64: string }>>(
            `/api/creative/tasks/${taskKey}/assets/${asset.id}/base64`
          )
        ).data.data.base64
      }`
    : assetUrl(asset)
  if (!source) throw new Error('Creative asset is unavailable')
  if (!asset.id && !source.startsWith('data:')) {
    const link = document.createElement('a')
    link.href = source
    link.download = asset.file_name
    link.click()
    return
  }
  let blob: Blob
  let fileName = asset.file_name
  if (resolution === '1K') {
    blob = await (await fetch(source)).blob()
  } else {
    const image = new Image()
    image.src = source
    await image.decode()
    if (!image.naturalWidth || !image.naturalHeight) {
      throw new Error('Creative image dimensions are unavailable')
    }
    const target = resolution === '4K' ? 4096 : 2048
    const longestSide = Math.max(image.naturalWidth, image.naturalHeight)
    const scale =
      asset.mime_type === 'image/svg+xml'
        ? target / longestSide
        : Math.max(1, target / longestSide)
    const canvas = document.createElement('canvas')
    canvas.width = Math.max(1, Math.round(image.naturalWidth * scale))
    canvas.height = Math.max(1, Math.round(image.naturalHeight * scale))
    const context = canvas.getContext('2d')
    if (!context) throw new Error('Canvas is unavailable')
    context.drawImage(image, 0, 0, canvas.width, canvas.height)
    const outputType =
      asset.mime_type === 'image/jpeg' || asset.mime_type === 'image/webp'
        ? asset.mime_type
        : 'image/png'
    blob = await new Promise<Blob>((resolve, reject) =>
      canvas.toBlob(
        (value) =>
          value ? resolve(value) : reject(new Error('Unable to render image')),
        outputType,
        0.95
      )
    )
    const extension = outputType.split('/')[1].replace('jpeg', 'jpg')
    fileName = `${fileName.replace(/\.[^.]+$/, '')}-${resolution}.${extension}`
  }
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  link.click()
  setTimeout(() => URL.revokeObjectURL(url), 0)
}

async function creativeAssetFile(
  taskKey: string,
  asset: CreativeAsset
): Promise<File> {
  const base64 = (
    await api.get<Envelope<{ base64: string }>>(
      `/api/creative/tasks/${taskKey}/assets/${asset.id}/base64`
    )
  ).data.data.base64
  const blob = await (
    await fetch(`data:${asset.mime_type};base64,${base64}`)
  ).blob()
  const file = new File([blob], asset.file_name, { type: asset.mime_type })
  if (file.type !== 'image/svg+xml') return file
  const url = URL.createObjectURL(file)
  try {
    const image = new Image()
    image.src = url
    await image.decode()
    if (!image.naturalWidth || !image.naturalHeight) {
      throw new Error('Creative image dimensions are unavailable')
    }
    const scale = Math.min(
      1,
      2048 / Math.max(image.naturalWidth, image.naturalHeight)
    )
    const canvas = document.createElement('canvas')
    canvas.width = Math.max(1, Math.round(image.naturalWidth * scale))
    canvas.height = Math.max(1, Math.round(image.naturalHeight * scale))
    const context = canvas.getContext('2d')
    if (!context) throw new Error('Canvas is unavailable')
    context.drawImage(image, 0, 0, canvas.width, canvas.height)
    const png = await new Promise<Blob>((resolve, reject) =>
      canvas.toBlob(
        (value) =>
          value ? resolve(value) : reject(new Error('Unable to render image')),
        'image/png'
      )
    )
    return new File(
      [png],
      `${file.name.replace(/\.svg$/i, '') || 'creative-image'}.png`,
      { type: 'image/png' }
    )
  } finally {
    URL.revokeObjectURL(url)
  }
}

function outputAssets(assets: CreativeAsset[]): CreativeOutput[] {
  const outputs = new Map<number, CreativeOutput>()
  for (const asset of assets) {
    const url = assetUrl(asset)
    if (asset.role !== 'output' || !url) continue
    const existing = outputs.get(asset.position)
    if (asset.temporary) {
      if (existing && !existing.asset.temporary) {
        existing.fallbackUrl = url
      } else {
        outputs.set(asset.position, { asset })
      }
      continue
    }
    outputs.set(asset.position, {
      asset,
      fallbackUrl: existing ? assetUrl(existing.asset) : undefined,
    })
  }
  return [...outputs.values()]
}

function taskPrompt(task?: CreativeTask): string {
  return String(taskParams(task).prompt ?? '')
}

function taskParams(task?: CreativeTask): Record<string, unknown> {
  const raw = task?.resolved_params ?? task?.requested_params
  if (!raw) return {}
  try {
    return JSON.parse(raw) as Record<string, unknown>
  } catch {
    return {}
  }
}

function displayParam(value: unknown): string {
  if (value === null) return 'null'
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value)
  }
  if (Array.isArray(value)) return value.join(', ')
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

function duration(task?: CreativeTask): string | undefined {
  if (!task?.started_at || !task.finished_at) return undefined
  const seconds = Math.max(
    0,
    (new Date(task.finished_at).getTime() -
      new Date(task.started_at).getTime()) /
      1000
  )
  if (!Number.isFinite(seconds)) return undefined
  const minutes = Math.floor(seconds / 60)
  const remainder = Math.floor(seconds % 60)
  return minutes
    ? `${minutes}:${String(remainder).padStart(2, '0')}`
    : `${remainder}s`
}

function formatCreativeDate(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '-' : formatDateTimeObject(date)
}

function parameterLabel(t: (key: string) => string, key: string): string {
  const labels: Record<string, string> = {
    n: 'Quantity',
    aspect_ratio: 'Aspect ratio',
    download_resolution: 'Download resolution',
    output_format: 'Format',
    size: 'Size',
    quality: 'Quality',
    background: 'Background',
    moderation: 'Moderation',
    transparent_background: 'Transparent background',
  }
  return t(labels[key] ?? key)
}

const preferredParameterKeys = [
  'size',
  'aspect_ratio',
  'download_resolution',
  'quality',
  'output_format',
  'background',
  'transparent_background',
  'moderation',
  'n',
]
const creativeStudioResultVisibilityKey = 'creative-studio-show-result'

function DetailItem(props: { label: string; value: string; wide?: boolean }) {
  return (
    <div className={props.wide ? 'col-span-2 grid gap-1' : 'grid gap-1'}>
      <dt className='text-muted-foreground'>{props.label}</dt>
      <dd className='font-medium break-words'>{props.value}</dd>
    </div>
  )
}

export function CreativeStudio() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [groupName, setGroupName] = useState('')
  const [operation, setOperation] = useState('')
  const [capabilityID, setCapabilityID] = useState<number>()
  const [params, setParams] = useState<Record<string, unknown>>({})
  const [selectedTaskKey, setSelectedTaskKey] = useState<string>()
  const [referenceImages, setReferenceImages] = useState<ReferenceImage[]>([])
  const cachedReferenceFiles = useRef(new Map<string, File>())
  const addingReferenceKeys = useRef(new Set<string>())
  const [addingReferenceKey, setAddingReferenceKey] = useState<string>()
  const [historyOpen, setHistoryOpen] = useState(false)
  const [assetPreview, setAssetPreview] = useState<AssetPreview>()
  const [deleteTaskKeys, setDeleteTaskKeys] = useState<string[]>([])
  const [selectedHistoryTaskKeys, setSelectedHistoryTaskKeys] = useState<
    Set<string>
  >(new Set())
  const [showResult, setShowResult] = useState(
    () =>
      typeof window === 'undefined' ||
      window.localStorage.getItem(creativeStudioResultVisibilityKey) !== 'false'
  )
  const bootstrap = useQuery({
    queryKey: ['creative-bootstrap'],
    queryFn: async () =>
      (await api.get<Envelope<Bootstrap>>('/api/creative/bootstrap')).data.data,
  })
  const tasks = useQuery({
    queryKey: ['creative-tasks'],
    queryFn: async () =>
      (
        await api.get<Envelope<{ items: CreativeTask[] }>>(
          '/api/creative/tasks'
        )
      ).data.data.items,
    refetchInterval: (query) =>
      query.state.data?.some(
        (task) => !['succeeded', 'failed'].includes(task.status)
      )
        ? 4000
        : false,
  })
  const historyTasks = useMemo(
    () =>
      (tasks.data ?? []).filter((task) =>
        ['succeeded', 'failed'].includes(task.status)
      ),
    [tasks.data]
  )
  const activeTasks = useMemo(
    () =>
      (tasks.data ?? []).filter(
        (task) => !['succeeded', 'failed'].includes(task.status)
      ),
    [tasks.data]
  )
  const historyTaskKeys = useMemo(
    () => historyTasks.map((task) => task.task_key),
    [historyTasks]
  )
  const historyDetails = useQueries({
    queries: (historyOpen ? historyTasks : []).map((task) => ({
      queryKey: ['creative-task', task.task_key],
      queryFn: ({ signal }) => getCreativeTaskDetail(task.task_key, signal),
    })),
  })
  const options = useMemo<CapabilityOption[]>(
    () =>
      bootstrap.data?.models.flatMap((creativeModel) =>
        (bootstrap.data.capabilities[String(creativeModel.id)] ?? []).map(
          (capability) => ({ capability, creativeModel })
        )
      ) ?? [],
    [bootstrap.data]
  )
  const groups = useMemo(
    () => [
      ...new Set(
        options.flatMap(({ capability }) =>
          (bootstrap.data?.publications[String(capability.id)] ?? []).map(
            (item) => item.group_name
          )
        )
      ),
    ],
    [bootstrap.data, options]
  )
  const activeGroupName = groups.includes(groupName)
    ? groupName
    : (groups[0] ?? '')
  const groupCapabilities = useMemo(
    () =>
      options.filter(({ capability }) =>
        (bootstrap.data?.publications[String(capability.id)] ?? []).some(
          (item) => item.group_name === activeGroupName
        )
      ),
    [activeGroupName, bootstrap.data, options]
  )
  const operations = useMemo(
    () => [
      ...new Set(
        groupCapabilities.map(({ capability }) => capability.operation)
      ),
    ],
    [groupCapabilities]
  )
  const activeOperation = operations.includes(operation)
    ? operation
    : (operations[0] ?? '')
  const models = useMemo(
    () =>
      groupCapabilities.filter(
        ({ capability }) => capability.operation === activeOperation
      ),
    [activeOperation, groupCapabilities]
  )
  const modelItems = useMemo(
    () =>
      models.map(({ capability, creativeModel }) => ({
        value: String(capability.id),
        label: creativeModel.display_name,
      })),
    [models]
  )
  const activeCapabilityID = models.some(
    ({ capability }) => capability.id === capabilityID
  )
    ? capabilityID
    : models[0]?.capability.id
  const selected = models.find(
    ({ capability }) => capability.id === activeCapabilityID
  )
  const fields = useMemo(
    () =>
      parseFields(selected?.capability.input_schema ?? '{}', activeOperation),
    [activeOperation, selected]
  )
  const activeTaskKey = showResult
    ? (selectedTaskKey ?? tasks.data?.[0]?.task_key)
    : undefined
  const taskDetail = useQuery({
    queryKey: ['creative-task', activeTaskKey],
    queryFn: ({ signal }) => getCreativeTaskDetail(activeTaskKey, signal),
    enabled: Boolean(activeTaskKey),
    refetchInterval: 4000,
  })
  const createTask = useMutation({
    mutationFn: async () => {
      const directReferences = activeOperation === 'edit' ? referenceImages : []
      const request = {
        capability_id: activeCapabilityID,
        group_name: activeGroupName,
        params: Object.fromEntries(
          Object.entries(params).filter(
            ([name]) => fields[name]?.type !== 'file'
          )
        ),
      }
      if (directReferences.length === 0) {
        return (
          await api.post<Envelope<CreativeTask>>('/api/creative/tasks', request)
        ).data.data
      }
      const form = new FormData()
      form.append('request', JSON.stringify(request))
      directReferences.forEach((reference) =>
        form.append('images', reference.file)
      )
      return (
        await api.post<Envelope<CreativeTask>>('/api/creative/tasks', form)
      ).data.data
    },
    onSuccess: (task) => {
      setSelectedTaskKey(task.task_key)
      setShowResult(true)
      queryClient.invalidateQueries({ queryKey: ['creative-tasks'] })
    },
    onError: () => toast.error(t('Failed to create task')),
  })
  const retryTask = useMutation({
    mutationFn: async (taskKey: string) =>
      (
        await api.post<Envelope<CreativeTask>>(
          `/api/creative/tasks/${taskKey}/retry`
        )
      ).data.data,
    onSuccess: (task) => {
      setSelectedTaskKey(task.task_key)
      setShowResult(true)
      setAssetPreview(undefined)
      queryClient.invalidateQueries({ queryKey: ['creative-tasks'] })
      toast.success(t('Retry started'))
    },
    onError: () => toast.error(t('Failed to retry task')),
  })
  const deleteTask = useMutation({
    mutationFn: async (taskKeys: string[]) => {
      await Promise.all(
        taskKeys.map((taskKey) =>
          queryClient.cancelQueries({ queryKey: ['creative-task', taskKey] })
        )
      )
      await Promise.all(
        taskKeys.map((taskKey) => api.delete(`/api/creative/tasks/${taskKey}`))
      )
    },
    onSuccess: (_, taskKeys) => {
      if (selectedTaskKey && taskKeys.includes(selectedTaskKey)) {
        setShowResult(false)
      }
      setAssetPreview(undefined)
      setDeleteTaskKeys([])
      setSelectedHistoryTaskKeys(new Set())
      taskKeys.forEach((taskKey) =>
        queryClient.removeQueries({ queryKey: ['creative-task', taskKey] })
      )
      queryClient.invalidateQueries({ queryKey: ['creative-tasks'] })
      toast.success(t('Deleted successfully'))
    },
    onError: () => toast.error(t('Failed to delete task')),
  })
  const copyBase64 = useMutation({
    mutationFn: async (asset: CreativeAsset) =>
      (
        await api.get<Envelope<{ base64: string }>>(
          `/api/creative/tasks/${taskDetail.data?.task.task_key}/assets/${asset.id}/base64`
        )
      ).data.data.base64,
    onSuccess: async (value) => {
      await navigator.clipboard.writeText(value)
      toast.success(t('Base64 copied to clipboard'))
    },
    onError: () => toast.error(t('Unable to copy Base64')),
  })
  const prompt = String(params.prompt ?? '')
  const outputs = showResult ? outputAssets(taskDetail.data?.assets ?? []) : []
  const editReferenceLimit = useMemo(() => {
    const editCapability = groupCapabilities.find(
      ({ capability }) => capability.operation === 'edit'
    )?.capability
    if (!editCapability) return 0
    const referenceField = Object.values(
      parseFields(editCapability.input_schema, 'edit')
    ).find((field) => field.type === 'file')
    return referenceField
      ? Math.min(
          referenceField.max_items ?? (referenceField.multiple ? 5 : 1),
          5
        )
      : 0
  }, [groupCapabilities])
  const canCreate = Boolean(
    selected &&
    activeGroupName &&
    prompt.trim() &&
    bootstrap.data?.storage_available &&
    (activeOperation !== 'edit' || referenceImages.length > 0) &&
    !createTask.isPending
  )

  const addResultAsReference = async (
    taskKey: string,
    asset: CreativeAsset
  ): Promise<boolean> => {
    const cacheKey = `${taskKey}:${asset.id}`
    if (editReferenceLimit === 0) {
      toast.error(t('No image-to-image model available'))
      return false
    }
    if (
      referenceImages.some((reference) => reference.cacheKey === cacheKey) ||
      addingReferenceKeys.current.has(cacheKey)
    ) {
      return false
    }
    if (
      referenceImages.length + addingReferenceKeys.current.size >=
      editReferenceLimit
    ) {
      toast.error(
        t('Current image-to-image reference image limit has been reached')
      )
      return false
    }
    addingReferenceKeys.current.add(cacheKey)
    setAddingReferenceKey(cacheKey)
    try {
      let file = cachedReferenceFiles.current.get(cacheKey)
      if (!file) {
        file = await creativeAssetFile(taskKey, asset)
        cachedReferenceFiles.current.set(cacheKey, file)
      }
      setReferenceImages((current) => [
        ...current,
        {
          id: crypto.randomUUID(),
          file,
          url: URL.createObjectURL(file),
          cacheKey,
        },
      ])
      setOperation('edit')
      setCapabilityID(undefined)
      setShowResult(false)
      return true
    } catch {
      toast.error(t('Unable to add reference image'))
      return false
    } finally {
      addingReferenceKeys.current.delete(cacheKey)
      setAddingReferenceKey(undefined)
    }
  }

  const removeReferenceImage = (reference: ReferenceImage) => {
    URL.revokeObjectURL(reference.url)
    if (reference.cacheKey) {
      cachedReferenceFiles.current.delete(reference.cacheKey)
    }
    setReferenceImages((current) =>
      current.filter((item) => item.id !== reference.id)
    )
  }

  useEffect(() => {
    setSelectedHistoryTaskKeys(
      (current) =>
        new Set([...current].filter((key) => historyTaskKeys.includes(key)))
    )
  }, [historyTaskKeys])

  useEffect(() => {
    window.localStorage.setItem(
      creativeStudioResultVisibilityKey,
      String(showResult)
    )
  }, [showResult])

  useEffect(() => {
    if (!selected?.capability.default_params) return
    try {
      setParams(
        JSON.parse(selected.capability.default_params) as Record<
          string,
          unknown
        >
      )
    } catch {
      setParams({})
    }
  }, [selected?.capability.default_params, selected?.capability.id])

  if (bootstrap.isLoading) return <div className='p-6'>{t('Loading')}</div>
  if (!bootstrap.data) {
    return <div className='p-6'>{t('No data available')}</div>
  }

  const resetSelection = () => {
    setOperation('')
    setCapabilityID(undefined)
    setParams({})
  }
  return (
    <main className='h-[calc(100dvh-4rem)] overflow-y-auto p-3 sm:p-4 lg:p-6'>
      <Card className='bg-card/95 mx-auto min-h-[calc(100svh-5.5rem)] max-w-[1800px] gap-0 overflow-hidden rounded-2xl py-0 shadow-sm'>
        <header className='flex min-h-16 flex-wrap items-center justify-between gap-3 border-b px-4 py-3 sm:px-5'>
          <div className='flex min-w-0 items-center gap-3'>
            <div className='hidden min-w-0 sm:block'>
              <h1 className='truncate font-semibold'>{t('Creative Studio')}</h1>
              <p className='text-muted-foreground text-xs'>
                {t('Image workspace')}
              </p>
            </div>
            <Select
              items={groups.map((group) => ({ label: group, value: group }))}
              value={activeGroupName}
              onValueChange={(value) => {
                if (value === null) return
                setGroupName(value)
                resetSelection()
              }}
            >
              <SelectTrigger className='w-[min(16rem,calc(100vw-10rem))]'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {groups.map((group) => (
                  <SelectItem key={group} value={group}>
                    {group}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button
              size='icon'
              variant='outline'
              aria-label={t('Refresh')}
              onClick={() => {
                bootstrap.refetch()
                tasks.refetch()
              }}
            >
              <RefreshCw />
            </Button>
          </div>
          <div className='flex items-center gap-2'>
            <Button variant='outline' onClick={() => setHistoryOpen(true)}>
              <History data-icon='inline-start' />
              {t('History works')}
            </Button>
          </div>
        </header>
        <div className='grid min-h-[calc(100svh-9.5rem)] flex-1 lg:grid-cols-[minmax(23rem,31rem)_minmax(0,1fr)]'>
          <section
            className='bg-muted/20 border-b p-4 sm:p-5 lg:max-h-[calc(100svh-9.5rem)] lg:overflow-y-auto lg:border-r lg:border-b-0'
            aria-label={t('Image workspace')}
          >
            <div className='flex min-h-full flex-col gap-5'>
              {operations.length > 1 && (
                <Tabs
                  value={activeOperation}
                  onValueChange={(value) => {
                    setOperation(value)
                    setCapabilityID(undefined)
                    setParams({})
                  }}
                >
                  <TabsList className='grid h-10 w-full grid-cols-2'>
                    {operations.map((item) => (
                      <TabsTrigger key={item} value={item}>
                        {item === 'edit'
                          ? t('Image to image')
                          : t('Text to image')}
                      </TabsTrigger>
                    ))}
                  </TabsList>
                </Tabs>
              )}
              <div className='grid items-start gap-4 sm:grid-cols-2'>
                <label className='grid gap-2 text-sm font-medium'>
                  {t('Choose a model')}
                  <Select
                    disabled={models.length === 0}
                    items={modelItems}
                    value={String(activeCapabilityID ?? '')}
                    onValueChange={(value) => {
                      setCapabilityID(Number(value))
                      setParams({})
                    }}
                  >
                    <SelectTrigger className='w-full'>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {models.map(({ capability, creativeModel }) => (
                        <SelectItem
                          key={capability.id}
                          value={String(capability.id)}
                        >
                          {creativeModel.display_name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </label>
                {fields.n && (
                  <CreativeField
                    name='n'
                    field={fields.n}
                    value={params.n}
                    references={referenceImages}
                    onReferencesChange={setReferenceImages}
                    onRemoveReference={removeReferenceImage}
                    onChange={(value) =>
                      setParams((current) => ({ ...current, n: value }))
                    }
                  />
                )}
              </div>
              <label className='grid gap-2 text-sm font-medium'>
                {t('Prompt')}
                <Textarea
                  className='min-h-40 resize-y'
                  value={prompt}
                  onChange={(event) =>
                    setParams((current) => ({
                      ...current,
                      prompt: event.target.value,
                    }))
                  }
                  placeholder={t(
                    'Example: a cyberpunk hacker cat coding beneath neon lights, cinematic lighting, highly detailed'
                  )}
                />
              </label>
              <div className='grid grid-cols-2 gap-2'>
                {promptExamples.map((example) => (
                  <Button
                    key={example}
                    variant='outline'
                    className='h-auto justify-start px-3 py-2 text-left text-xs whitespace-normal'
                    onClick={() =>
                      setParams((current) => ({
                        ...current,
                        prompt: t(example),
                      }))
                    }
                  >
                    {t(example)}
                  </Button>
                ))}
              </div>
              <div className='grid grid-cols-2 gap-5'>
                {Object.entries(fields)
                  .filter(([name]) => name !== 'prompt' && name !== 'n')
                  .map(([name, field]) => (
                    <div
                      key={name}
                      className={cn(
                        name !== 'quality' &&
                          name !== 'output_format' &&
                          'col-span-2'
                      )}
                    >
                      <CreativeField
                        name={name}
                        field={field}
                        value={params[name]}
                        references={referenceImages}
                        onReferencesChange={setReferenceImages}
                        onRemoveReference={removeReferenceImage}
                        onChange={(value) =>
                          setParams((current) => ({
                            ...current,
                            [name]: value,
                          }))
                        }
                      />
                    </div>
                  ))}
              </div>
              <div className='mt-auto border-t pt-4'>
                <Button
                  size='lg'
                  className='w-full'
                  disabled={!canCreate}
                  onClick={() => createTask.mutate()}
                >
                  {createTask.isPending && <Spinner data-icon='inline-start' />}
                  {createTask.isPending ? t('Generating') : t('Start creating')}
                </Button>
              </div>
            </div>
          </section>
          <section
            className='flex min-w-0 flex-col'
            aria-label={t('My creations')}
          >
            <header className='flex items-center justify-between gap-4 border-b px-4 py-4 sm:px-6'>
              <div>
                <h2 className='font-semibold'>{t('Generation results')}</h2>
                <p className='text-muted-foreground text-sm'>
                  {t('Creation history')}
                </p>
              </div>
              <div className='flex items-center gap-2'>
                {showResult && taskDetail.data && (
                  <Badge variant={statusVariant(taskDetail.data.task.status)}>
                    {statusLabel(t, taskDetail.data.task.status)}
                  </Badge>
                )}
                <Button
                  size='sm'
                  variant='outline'
                  onClick={() => {
                    setSelectedTaskKey(undefined)
                    setShowResult(false)
                  }}
                >
                  {t('Clear')}
                </Button>
              </div>
            </header>
            <div className='flex min-h-96 flex-1 p-4 sm:p-6'>
              {outputs.length > 0 ? (
                <div
                  className={`grid w-full content-start gap-4 ${
                    outputs.length === 1
                      ? 'grid-cols-1'
                      : 'sm:grid-cols-2 xl:grid-cols-3'
                  }`}
                >
                  {outputs.map(({ asset, fallbackUrl }) => {
                    const url = assetUrl(asset)
                    if (!url) return null
                    return (
                      <div
                        key={asset.position}
                        className='relative flex min-w-0 justify-center'
                      >
                        <button
                          type='button'
                          className='border-border bg-muted/20 flex min-h-64 w-full cursor-zoom-in items-center justify-center overflow-hidden rounded-xl border p-2 transition-opacity hover:opacity-90'
                          onClick={() =>
                            activeTaskKey &&
                            setAssetPreview({
                              taskKey: activeTaskKey,
                              position: asset.position,
                            })
                          }
                          aria-label={t('View details')}
                        >
                          <CreativeTaskAssetImage
                            taskKey={activeTaskKey}
                            asset={asset}
                            fallbackSrc={fallbackUrl}
                            alt={asset.file_name}
                            className='max-h-[calc(100svh-16rem)] w-full object-contain'
                          />
                        </button>
                        <div className='absolute right-2 bottom-2 flex gap-1'>
                          <Button
                            size='icon-xs'
                            variant='secondary'
                            aria-label={t('Download')}
                            onClick={() => {
                              if (!activeTaskKey) return
                              void downloadCreativeAsset(
                                activeTaskKey,
                                asset,
                                taskDownloadResolution(taskDetail.data?.task)
                              ).catch(() =>
                                toast.error(t('Unable to download image'))
                              )
                            }}
                          >
                            <Download />
                          </Button>
                          <Button
                            size='xs'
                            variant='secondary'
                            disabled={asset.temporary || copyBase64.isPending}
                            onClick={() => copyBase64.mutate(asset)}
                          >
                            <Copy data-icon='inline-start' />
                            {t('Convert to Base64')}
                          </Button>
                        </div>
                      </div>
                    )
                  })}
                </div>
              ) : (
                <Empty className='m-auto max-w-md border-0'>
                  <EmptyHeader>
                    <EmptyMedia
                      variant='icon'
                      className='border-primary/20 bg-primary/10 text-primary'
                    >
                      <ImageIcon />
                    </EmptyMedia>
                    <EmptyTitle>{t('My creations')}</EmptyTitle>
                    <EmptyDescription>
                      {(showResult && taskDetail.data?.task.error_message) ??
                        t('No creations yet')}
                    </EmptyDescription>
                    {activeTasks.length > 0 && (
                      <div className='mt-4 flex flex-wrap justify-center gap-2'>
                        {activeTasks.map((task) => (
                          <Button
                            key={task.task_key}
                            size='sm'
                            variant='outline'
                            onClick={() => {
                              setSelectedTaskKey(task.task_key)
                              setShowResult(true)
                            }}
                          >
                            <Spinner data-icon='inline-start' />
                            {statusLabel(t, task.status)} · {t('View details')}
                          </Button>
                        ))}
                      </div>
                    )}
                  </EmptyHeader>
                </Empty>
              )}
            </div>
          </section>
        </div>
      </Card>
      <Dialog open={historyOpen} onOpenChange={setHistoryOpen}>
        <DialogContent className='sm:max-w-[calc(100%-2rem)] lg:max-w-4xl'>
          <DialogHeader>
            <DialogTitle>{t('History works')}</DialogTitle>
            <DialogDescription>{t('Creation history')}</DialogDescription>
          </DialogHeader>
          <div className='flex flex-wrap items-center gap-2'>
            <Button
              size='sm'
              variant='outline'
              onClick={() =>
                setSelectedHistoryTaskKeys(new Set(historyTaskKeys))
              }
            >
              {t('Select all')}
            </Button>
            <Button
              size='sm'
              variant='outline'
              onClick={() =>
                setSelectedHistoryTaskKeys(
                  new Set(
                    historyTaskKeys.filter(
                      (key) => !selectedHistoryTaskKeys.has(key)
                    )
                  )
                )
              }
            >
              {t('Invert selection')}
            </Button>
            <Button
              size='sm'
              variant='outline'
              disabled={selectedHistoryTaskKeys.size === 0}
              onClick={() => {
                const downloads: Promise<void>[] = []
                for (const [index, task] of historyTasks.entries()) {
                  if (!selectedHistoryTaskKeys.has(task.task_key)) continue
                  for (const output of outputAssets(
                    historyDetails[index]?.data?.assets ?? []
                  )) {
                    downloads.push(
                      downloadCreativeAsset(
                        task.task_key,
                        output.asset,
                        taskDownloadResolution(
                          historyDetails[index]?.data?.task
                        )
                      )
                    )
                  }
                }
                void Promise.all(downloads).catch(() =>
                  toast.error(t('Unable to download image'))
                )
              }}
            >
              <Download data-icon='inline-start' />
              {t('Download selected')}
            </Button>
            <Button
              size='sm'
              variant='destructive'
              disabled={selectedHistoryTaskKeys.size === 0}
              onClick={() => setDeleteTaskKeys([...selectedHistoryTaskKeys])}
            >
              <Trash2 data-icon='inline-start' />
              {t('Delete selected')}
            </Button>
          </div>
          <div className='grid max-h-[70vh] grid-cols-1 gap-4 overflow-y-auto pr-1 sm:grid-cols-2 lg:grid-cols-3'>
            {historyTasks.map((task, index) => (
              <HistoryTaskCards
                key={task.task_key}
                task={task}
                detail={historyDetails[index]?.data}
                loading={historyDetails[index]?.isLoading ?? false}
                selected={selectedHistoryTaskKeys.has(task.task_key)}
                onSelectedChange={(selected) =>
                  setSelectedHistoryTaskKeys((current) => {
                    const next = new Set(current)
                    if (selected) next.add(task.task_key)
                    else next.delete(task.task_key)
                    return next
                  })
                }
                onPreview={(position) =>
                  setAssetPreview({ taskKey: task.task_key, position })
                }
              />
            ))}
          </div>
        </DialogContent>
      </Dialog>
      <CreativeTaskPreviewDialog
        preview={assetPreview}
        onOpenChange={(open) => !open && setAssetPreview(undefined)}
        onPreview={(position) =>
          setAssetPreview((current) =>
            current ? { ...current, position } : current
          )
        }
        onRetry={(taskKey) => retryTask.mutate(taskKey)}
        retrying={retryTask.isPending}
        onDelete={(taskKey) => setDeleteTaskKeys([taskKey])}
        onEdit={addResultAsReference}
        addingReferenceKey={addingReferenceKey}
      />
      <AlertDialog
        open={deleteTaskKeys.length > 0}
        onOpenChange={(open) => !open && setDeleteTaskKeys([])}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Delete task?')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'Are you sure you want to delete this task? This action cannot be undone.'
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteTask.isPending}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant='destructive'
              disabled={deleteTask.isPending}
              onClick={(event) => {
                event.preventDefault()
                if (deleteTaskKeys.length) deleteTask.mutate(deleteTaskKeys)
              }}
            >
              <Trash2 data-icon='inline-start' />
              {t('Delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </main>
  )
}

function CreativeTaskAssetImage(props: {
  taskKey?: string
  asset: CreativeAsset
  fallbackSrc?: string
  alt: string
  className?: string
}) {
  const svgPreview = useQuery({
    queryKey: ['creative-task-asset-base64', props.taskKey, props.asset.id],
    queryFn: async () =>
      (
        await api.get<Envelope<{ base64: string }>>(
          `/api/creative/tasks/${props.taskKey}/assets/${props.asset.id}/base64`
        )
      ).data.data.base64,
    enabled: Boolean(
      props.taskKey &&
      props.asset.id &&
      props.asset.mime_type === 'image/svg+xml'
    ),
  })
  let src = assetUrl(props.asset)
  if (props.asset.mime_type === 'image/svg+xml') {
    src = svgPreview.data
      ? `data:image/svg+xml;base64,${svgPreview.data}`
      : undefined
  }
  if (!src) return <Spinner />
  return (
    <StableCreativeImage
      src={src}
      fallbackSrc={
        props.asset.mime_type === 'image/svg+xml'
          ? undefined
          : props.fallbackSrc
      }
      alt={props.alt}
      className={props.className}
    />
  )
}

function StableCreativeImage(props: {
  src: string
  fallbackSrc?: string
  alt: string
  className?: string
}) {
  const [displayedSrc, setDisplayedSrc] = useState(
    props.fallbackSrc ?? props.src
  )

  useEffect(() => {
    if (props.src === props.fallbackSrc || props.src.startsWith('data:')) {
      setDisplayedSrc(props.src)
      return
    }
    let cancelled = false
    const image = new Image()
    image.addEventListener('load', () => {
      if (!cancelled) setDisplayedSrc(props.src)
    })
    image.src = props.src
    return () => {
      cancelled = true
    }
  }, [props.fallbackSrc, props.src])

  return <img src={displayedSrc} alt={props.alt} className={props.className} />
}

function HistoryTaskCards(props: {
  task: CreativeTask
  detail?: CreativeTaskDetail
  loading: boolean
  selected: boolean
  onSelectedChange: (selected: boolean) => void
  onPreview: (position?: number) => void
}) {
  const { t } = useTranslation()
  const prompt = taskPrompt(props.task)
  const output = outputAssets(props.detail?.assets ?? [])[0]
  const url = output ? assetUrl(output.asset) : undefined
  let preview: ReactNode
  if (props.loading) {
    preview = (
      <div className='flex aspect-[4/3] items-center justify-center'>
        <Spinner />
      </div>
    )
  } else if (url && output) {
    preview = (
      <CreativeTaskAssetImage
        taskKey={props.task.task_key}
        asset={output.asset}
        fallbackSrc={output.fallbackUrl}
        alt={output.asset.file_name}
        className='aspect-[4/3] w-full object-contain'
      />
    )
  } else {
    preview = (
      <div className='text-muted-foreground flex aspect-[4/3] items-center justify-center'>
        <ImageIcon className='size-7' />
      </div>
    )
  }

  return (
    <div className='relative'>
      <Checkbox
        checked={props.selected}
        onCheckedChange={(value) => props.onSelectedChange(Boolean(value))}
        aria-label={t('Select row')}
        className='bg-background absolute top-3 left-3 z-10'
      />
      <button
        type='button'
        className='border-border bg-muted/20 group focus-visible:ring-ring w-full overflow-hidden rounded-xl border text-left transition-shadow hover:shadow-md focus-visible:ring-2 focus-visible:outline-none'
        onClick={() => props.onPreview(output?.asset.position)}
      >
        {preview}
        <div className='bg-background/90 flex items-center gap-2 border-t px-3 py-2'>
          <span
            className='min-w-0 flex-1 truncate text-sm font-medium'
            title={prompt || props.task.display_name}
          >
            {prompt || props.task.display_name}
          </span>
          <Badge variant={statusVariant(props.task.status)}>
            {statusLabel(t, props.task.status)}
          </Badge>
        </div>
      </button>
    </div>
  )
}

function InteractiveCreativeImage(props: {
  taskKey?: string
  asset?: CreativeAsset
  src?: string
  fallbackSrc?: string
  alt: string
}) {
  const { t } = useTranslation()
  const [zoom, setZoom] = useState(1)
  const viewportRef = useRef<HTMLDivElement>(null)
  const imageRef = useRef<HTMLDivElement>(null)
  const zoomRef = useRef(1)
  const rotationRef = useRef(0)
  const offset = useRef({ x: 0, y: 0 })
  const frame = useRef<number | undefined>(undefined)
  const wheelCommitTimer = useRef<number | undefined>(undefined)
  const dragStart = useRef<
    | {
        x: number
        y: number
        offsetX: number
        offsetY: number
      }
    | undefined
  >(undefined)
  const didDrag = useRef(false)

  const applyTransform = useCallback(() => {
    if (!imageRef.current) return
    imageRef.current.style.transform = `translate3d(${offset.current.x}px, ${offset.current.y}px, 0) scale(${zoomRef.current}) rotate(${rotationRef.current}deg)`
  }, [])

  const scheduleTransform = useCallback(() => {
    if (frame.current !== undefined) return
    frame.current = requestAnimationFrame(() => {
      frame.current = undefined
      applyTransform()
    })
  }, [applyTransform])

  useEffect(() => {
    zoomRef.current = 1
    rotationRef.current = 0
    setZoom(1)
    offset.current = { x: 0, y: 0 }
    if (wheelCommitTimer.current !== undefined) {
      window.clearTimeout(wheelCommitTimer.current)
      wheelCommitTimer.current = undefined
    }
    applyTransform()
  }, [applyTransform, props.asset?.id, props.src])

  const updateZoom = useCallback(
    (nextZoom: number, commit = true) => {
      const clampedZoom = Math.min(3, Math.max(1, nextZoom))
      zoomRef.current = clampedZoom
      if (clampedZoom === 1) {
        offset.current = { x: 0, y: 0 }
      }
      scheduleTransform()
      if (commit) setZoom(clampedZoom)
    },
    [scheduleTransform]
  )

  useEffect(() => {
    const viewport = viewportRef.current
    if (!viewport) return
    const handleWheel = (event: WheelEvent) => {
      if (!event.ctrlKey) return
      event.preventDefault()
      updateZoom(zoomRef.current * Math.exp(-event.deltaY * 0.0015), false)
      if (wheelCommitTimer.current !== undefined) {
        window.clearTimeout(wheelCommitTimer.current)
      }
      wheelCommitTimer.current = window.setTimeout(() => {
        wheelCommitTimer.current = undefined
        setZoom(zoomRef.current)
      }, 80)
    }
    viewport.addEventListener('wheel', handleWheel, { passive: false })
    return () => {
      viewport.removeEventListener('wheel', handleWheel)
      if (wheelCommitTimer.current !== undefined) {
        window.clearTimeout(wheelCommitTimer.current)
      }
    }
  }, [updateZoom])

  useEffect(
    () => () => {
      if (frame.current !== undefined) cancelAnimationFrame(frame.current)
    },
    []
  )

  return (
    <div
      ref={viewportRef}
      className='relative flex max-h-[calc(100svh-14rem)] max-w-full items-center justify-center overflow-hidden'
    >
      <div
        ref={imageRef}
        className={
          zoom > 1
            ? 'cursor-grab touch-none will-change-transform select-none active:cursor-grabbing'
            : 'cursor-zoom-in touch-none will-change-transform select-none'
        }
        onClick={() => {
          if (didDrag.current) {
            didDrag.current = false
            return
          }
          updateZoom(zoomRef.current === 1 ? 1.5 : 1)
        }}
        onPointerDown={(event) => {
          if (zoomRef.current <= 1 || !event.isPrimary) return
          event.currentTarget.setPointerCapture(event.pointerId)
          dragStart.current = {
            x: event.clientX,
            y: event.clientY,
            offsetX: offset.current.x,
            offsetY: offset.current.y,
          }
          didDrag.current = false
        }}
        onDragStart={(event) => event.preventDefault()}
        onPointerMove={(event) => {
          if (!dragStart.current) return
          didDrag.current = true
          offset.current = {
            x: dragStart.current.offsetX + event.clientX - dragStart.current.x,
            y: dragStart.current.offsetY + event.clientY - dragStart.current.y,
          }
          scheduleTransform()
        }}
        onPointerUp={() => {
          dragStart.current = undefined
        }}
        onPointerCancel={() => {
          dragStart.current = undefined
        }}
        role='button'
        tabIndex={0}
        aria-label={t('Zoom in')}
        onKeyDown={(event) => {
          if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault()
            updateZoom(zoomRef.current === 1 ? 1.5 : 1)
          }
        }}
      >
        {props.asset ? (
          <CreativeTaskAssetImage
            taskKey={props.taskKey}
            asset={props.asset}
            fallbackSrc={props.fallbackSrc}
            alt={props.alt}
            className='max-h-[calc(100svh-14rem)] max-w-full object-contain'
          />
        ) : (
          props.src && (
            <StableCreativeImage
              src={props.src}
              fallbackSrc={props.fallbackSrc}
              alt={props.alt}
              className='max-h-[calc(100svh-14rem)] max-w-full object-contain'
            />
          )
        )}
      </div>
      <div className='bg-background/90 absolute right-2 bottom-2 flex gap-1 rounded-md border p-1 shadow-sm'>
        <Button
          size='icon-xs'
          variant='ghost'
          aria-label={t('Zoom out')}
          disabled={zoom <= 1}
          onClick={() => updateZoom(zoomRef.current - 0.25)}
        >
          <ZoomOut />
        </Button>
        <Button
          size='icon-xs'
          variant='ghost'
          aria-label={t('Zoom in')}
          disabled={zoom >= 3}
          onClick={() => updateZoom(zoomRef.current + 0.25)}
        >
          <ZoomIn />
        </Button>
        <Button
          size='icon-xs'
          variant='ghost'
          aria-label={t('Rotate left')}
          onClick={() => {
            rotationRef.current -= 90
            scheduleTransform()
          }}
        >
          <RotateCcw />
        </Button>
        <Button
          size='icon-xs'
          variant='ghost'
          aria-label={t('Rotate right')}
          onClick={() => {
            rotationRef.current += 90
            scheduleTransform()
          }}
        >
          <RotateCw />
        </Button>
        <Button
          size='icon-xs'
          variant='ghost'
          aria-label={t('Reset')}
          onClick={() => {
            zoomRef.current = 1
            rotationRef.current = 0
            setZoom(1)
            offset.current = { x: 0, y: 0 }
            scheduleTransform()
          }}
        >
          <RefreshCw />
        </Button>
      </div>
    </div>
  )
}

function CreativeTaskPreviewDialog(props: {
  preview?: AssetPreview
  onOpenChange: (open: boolean) => void
  onPreview: (position: number) => void
  onRetry: (taskKey: string) => void
  retrying: boolean
  onDelete: (taskKey: string) => void
  onEdit: (taskKey: string, asset: CreativeAsset) => Promise<boolean>
  addingReferenceKey?: string
}) {
  const { t } = useTranslation()
  const onOpenChange = props.onOpenChange
  const [closeCountdown, setCloseCountdown] = useState<number>()
  const detail = useQuery({
    queryKey: ['creative-task', props.preview?.taskKey],
    queryFn: ({ signal }) =>
      getCreativeTaskDetail(props.preview?.taskKey, signal),
    enabled: Boolean(props.preview?.taskKey),
  })
  const outputs = outputAssets(detail.data?.assets ?? []).sort(
    (left, right) => left.asset.position - right.asset.position
  )
  const outputIndex = outputs.findIndex(
    (item) => item.asset.position === props.preview?.position
  )
  const output = outputs[outputIndex]
  const previousOutput = outputs[outputIndex - 1]
  const nextOutput = outputs[outputIndex + 1]
  const url = output ? assetUrl(output.asset) : undefined
  const task = detail.data?.task
  const prompt = taskPrompt(task)
  const params = taskParams(task)
  const taskDuration = duration(task)
  const parameterItems = Object.entries(params)
    .filter(([key]) => !['prompt', 'image', 'images'].includes(key))
    .sort(([left], [right]) => {
      const leftIndex = preferredParameterKeys.indexOf(left)
      const rightIndex = preferredParameterKeys.indexOf(right)
      if (leftIndex === -1 && rightIndex === -1) {
        return left.localeCompare(right)
      }
      if (leftIndex === -1) return 1
      if (rightIndex === -1) return -1
      return leftIndex - rightIndex
    })
  const inputAssets = detail.data?.assets.filter(
    (asset) => asset.role === 'input'
  )
  const adding = Boolean(
    props.preview &&
    output &&
    props.addingReferenceKey === `${props.preview.taskKey}:${output.asset.id}`
  )

  useEffect(() => {
    if (!closeCountdown) return
    const timer = window.setTimeout(() => {
      if (closeCountdown === 1) {
        setCloseCountdown(undefined)
        onOpenChange(false)
        return
      }
      setCloseCountdown((current) => (current ? current - 1 : current))
    }, 1000)
    return () => window.clearTimeout(timer)
  }, [closeCountdown, onOpenChange])

  const copyPrompt = async () => {
    if (!prompt) return
    try {
      await navigator.clipboard.writeText(prompt)
      toast.success(t('Copied to clipboard'))
    } catch {
      toast.error(t('Copy failed'))
    }
  }
  let editLabel = t('Edit')
  if (adding) {
    editLabel = t('Adding reference image')
  }
  if (closeCountdown) {
    editLabel = t('Added. Closing in {{count}} seconds', {
      count: closeCountdown,
    })
  }
  let preview: ReactNode
  if (url) {
    preview = (
      <InteractiveCreativeImage
        taskKey={props.preview?.taskKey}
        asset={output.asset}
        fallbackSrc={output?.fallbackUrl}
        alt={output?.asset.file_name ?? t('Generated image')}
      />
    )
  } else if (task?.status === 'failed') {
    preview = (
      <p className='text-muted-foreground max-w-md text-center text-sm break-words'>
        {task.error_message || t('Failed')}
      </p>
    )
  } else {
    preview = <Spinner />
  }

  return (
    <Dialog open={Boolean(props.preview)} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[calc(100svh-2rem)] overflow-y-auto sm:max-w-[calc(100%-2rem)] lg:max-w-6xl'>
        <DialogHeader>
          <DialogTitle>{t('Image Preview')}</DialogTitle>
          <DialogDescription>
            {detail.data?.task.display_name ?? t('Loading')}
          </DialogDescription>
        </DialogHeader>
        <div className='grid gap-5 lg:grid-cols-[minmax(0,1fr)_18rem]'>
          <div className='bg-muted/30 relative flex min-h-[18rem] items-center justify-center rounded-xl border p-3'>
            {preview}
            {outputs.length > 1 && (
              <>
                <Button
                  size='icon'
                  variant='secondary'
                  className='absolute top-1/2 left-4 -translate-y-1/2'
                  aria-label={t('Previous image')}
                  disabled={!previousOutput}
                  onClick={() =>
                    previousOutput &&
                    props.onPreview(previousOutput.asset.position)
                  }
                >
                  <ChevronLeft />
                </Button>
                <Button
                  size='icon'
                  variant='secondary'
                  className='absolute top-1/2 right-4 -translate-y-1/2'
                  aria-label={t('Next image')}
                  disabled={!nextOutput}
                  onClick={() =>
                    nextOutput && props.onPreview(nextOutput.asset.position)
                  }
                >
                  <ChevronRight />
                </Button>
                <span className='bg-background/90 absolute bottom-4 rounded-md border px-2 py-1 text-xs tabular-nums'>
                  {outputIndex + 1} / {outputs.length}
                </span>
              </>
            )}
          </div>
          {task && (
            <div className='grid content-start gap-5 text-sm'>
              <section className='grid gap-2'>
                <div className='flex items-center justify-between gap-3'>
                  <h3 className='text-muted-foreground font-medium'>
                    {t('Input')}
                  </h3>
                  {prompt && (
                    <Button
                      size='icon-xs'
                      variant='ghost'
                      aria-label={t('Copy prompt')}
                      onClick={() => void copyPrompt()}
                    >
                      <Copy />
                    </Button>
                  )}
                </div>
                {prompt ? (
                  <div className='max-h-48 overflow-y-auto rounded-lg bg-amber-100/70 p-3 leading-6 whitespace-pre-wrap text-amber-900 dark:bg-amber-950/35 dark:text-amber-100'>
                    {prompt}
                  </div>
                ) : (
                  <p className='text-muted-foreground'>-</p>
                )}
                {inputAssets && inputAssets.length > 0 && (
                  <div className='grid gap-2'>
                    <span className='text-muted-foreground text-xs'>
                      {t('Input images')}
                    </span>
                    <div className='flex flex-wrap gap-2'>
                      {inputAssets.map((asset) => {
                        const inputUrl = assetUrl(asset)
                        return inputUrl ? (
                          <img
                            key={asset.id}
                            src={inputUrl}
                            alt={asset.file_name}
                            className='border-border size-14 rounded-md border object-cover'
                          />
                        ) : (
                          <span
                            key={asset.id}
                            className='bg-muted max-w-40 truncate rounded-md px-2 py-1 text-xs'
                          >
                            {asset.file_name}
                          </span>
                        )
                      })}
                    </div>
                  </div>
                )}
              </section>
              <section className='grid gap-2'>
                <h3 className='text-muted-foreground font-medium'>
                  {t('Parameters')}
                </h3>
                <dl className='grid grid-cols-2 gap-x-5 gap-y-4'>
                  <DetailItem
                    label={t('Source')}
                    value={
                      [
                        task.source_vendor,
                        task.source_channel_name &&
                          `${task.source_channel_name} / ${task.request_model ?? task.model_name}`,
                        task.model_name,
                      ]
                        .filter(Boolean)
                        .join(' · ') || task.display_name
                    }
                    wide
                  />
                  <DetailItem label={t('Group')} value={task.group_name} />
                  <DetailItem
                    label={t('Operation')}
                    value={
                      task.operation === 'edit'
                        ? t('Image to image')
                        : t('Text to image')
                    }
                  />
                  <DetailItem
                    label={t('Status')}
                    value={statusLabel(t, task.status)}
                  />
                  {task.error_message && (
                    <DetailItem
                      label={t('Error')}
                      value={task.error_message}
                      wide
                    />
                  )}
                  {parameterItems.map(([key, value]) => (
                    <DetailItem
                      key={key}
                      label={parameterLabel(t, key)}
                      value={displayParam(value)}
                    />
                  ))}
                </dl>
              </section>
              <div className='text-muted-foreground text-xs'>
                {t('Created At')} {formatCreativeDate(task.created_at)}
                {taskDuration && ` · ${t('Duration')} ${taskDuration}`}
              </div>
              {url && task.status !== 'failed' && (
                <Button
                  variant='outline'
                  disabled={!output}
                  onClick={() => {
                    if (!output || !props.preview) return
                    void downloadCreativeAsset(
                      props.preview.taskKey,
                      output.asset,
                      taskDownloadResolution(task)
                    ).catch(() => toast.error(t('Unable to download image')))
                  }}
                >
                  <Download data-icon='inline-start' />
                  {t('Download')}
                </Button>
              )}
              {output && props.preview && task.status !== 'failed' && (
                <Button
                  variant='outline'
                  disabled={adding || closeCountdown !== undefined}
                  onClick={async () => {
                    if (!props.preview) return
                    if (
                      await props.onEdit(props.preview.taskKey, output.asset)
                    ) {
                      toast.success(t('Reference image added'))
                      setCloseCountdown(3)
                    }
                  }}
                >
                  {adding ? (
                    <Spinner data-icon='inline-start' />
                  ) : (
                    <ImageIcon data-icon='inline-start' />
                  )}
                  {editLabel}
                </Button>
              )}
              <div className='grid grid-cols-2 gap-2'>
                <Button
                  variant='outline'
                  disabled={props.retrying}
                  onClick={() =>
                    props.preview && props.onRetry(props.preview.taskKey)
                  }
                >
                  {props.retrying ? (
                    <Spinner data-icon='inline-start' />
                  ) : (
                    <RotateCcw data-icon='inline-start' />
                  )}
                  {t('Redo')}
                </Button>
                <Button
                  variant='destructive'
                  onClick={() =>
                    props.preview && props.onDelete(props.preview.taskKey)
                  }
                >
                  <Trash2 data-icon='inline-start' />
                  {t('Delete')}
                </Button>
              </div>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

function ReferenceImageField(props: {
  name: string
  field: Field
  value: unknown
  references: ReferenceImage[]
  onReferencesChange: (references: ReferenceImage[]) => void
  onRemoveReference: (reference: ReferenceImage) => void
  onChange: (value: unknown) => void
  label: string
}) {
  const { t } = useTranslation()
  const inputID = useId()
  const maxItems = Math.min(
    props.field.max_items ?? (props.field.multiple ? 5 : 1),
    5
  )

  return (
    <div className='grid gap-2 text-sm font-medium'>
      <span>{props.label}</span>
      {props.references.length > 0 && (
        <div className='flex flex-wrap gap-2'>
          {props.references.map((reference) => (
            <ReferenceImageThumbnail
              key={reference.id}
              reference={reference}
              onRemove={() => props.onRemoveReference(reference)}
            />
          ))}
        </div>
      )}
      {props.references.length < maxItems && (
        <>
          <input
            id={inputID}
            className='sr-only'
            type='file'
            accept='image/png,image/jpeg,image/webp,image/gif'
            multiple={props.field.multiple}
            onChange={(event) => {
              const files = [...(event.target.files ?? [])].slice(
                0,
                maxItems - props.references.length
              )
              props.onReferencesChange([
                ...props.references,
                ...files.map((file) => ({
                  id: crypto.randomUUID(),
                  file,
                  url: URL.createObjectURL(file),
                })),
              ])
              event.target.value = ''
            }}
          />
          <Button
            variant='outline'
            onClick={() =>
              document
                .querySelector<HTMLInputElement>(`#${CSS.escape(inputID)}`)
                ?.click()
            }
          >
            <Upload data-icon='inline-start' />
            {t('Upload reference image')}
          </Button>
        </>
      )}
    </div>
  )
}

function ReferenceImageThumbnail(props: {
  reference: ReferenceImage
  onRemove: () => void
}) {
  const { t } = useTranslation()
  const [previewOpen, setPreviewOpen] = useState(false)

  return (
    <div className='bg-background relative size-20 overflow-hidden rounded-lg border'>
      <button
        type='button'
        className='size-full cursor-zoom-in'
        aria-label={t('View details')}
        onClick={() => setPreviewOpen(true)}
      >
        <img
          src={props.reference.url}
          alt={props.reference.file.name}
          className='size-full object-cover'
        />
      </button>
      <Button
        size='icon-xs'
        variant='secondary'
        className='absolute top-1 right-1'
        aria-label={t('Remove')}
        onClick={props.onRemove}
      >
        <X />
      </Button>
      <Dialog open={previewOpen} onOpenChange={setPreviewOpen}>
        <DialogContent className='sm:max-w-3xl'>
          <DialogHeader>
            <DialogTitle>{props.reference.file.name}</DialogTitle>
          </DialogHeader>
          <div className='bg-muted/30 flex min-h-96 items-center justify-center rounded-xl border p-3'>
            <InteractiveCreativeImage
              src={props.reference.url}
              alt={props.reference.file.name}
            />
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function CreativeField(props: {
  name: string
  field: Field
  value: unknown
  references: ReferenceImage[]
  onReferencesChange: (references: ReferenceImage[]) => void
  onRemoveReference: (reference: ReferenceImage) => void
  onChange: (value: unknown) => void
}) {
  const { t } = useTranslation()
  const label = t(props.field.label ?? props.name)
  if (props.field.type === 'file') {
    return <ReferenceImageField {...props} label={label} />
  }
  const options = (props.field.options ?? []).map((option) => {
    if (typeof option === 'object') {
      return {
        label: option.label,
        value: option.value,
        aspect_ratio: option.aspect_ratio,
      }
    }
    return { label: String(option), value: option, aspect_ratio: undefined }
  })
  if (
    options.length > 0 &&
    (props.field.presentation === 'segmented' ||
      props.field.presentation === 'size_cards')
  ) {
    const optionValues = options.map((option) => option.value)
    const customValue =
      props.field.allow_custom &&
      typeof props.value === 'string' &&
      !optionValues.includes(props.value)
        ? props.value
        : undefined
    if (props.field.presentation === 'size_cards') {
      return (
        <div className='grid gap-2 text-sm font-medium'>
          <span>{label}</span>
          <div className='grid grid-cols-2 gap-2 sm:grid-cols-4'>
            {options.map((option) => {
              let aspectClassName = 'size-9'
              if (
                option.aspect_ratio === '16:9' ||
                option.aspect_ratio === '3:2'
              ) {
                aspectClassName = 'h-7 w-12'
              } else if (
                option.aspect_ratio === '9:16' ||
                option.aspect_ratio === '2:3'
              ) {
                aspectClassName = 'h-9 w-6'
              } else if (option.aspect_ratio === 'auto') {
                aspectClassName = 'size-9 border-dashed'
              }
              return (
                <Button
                  key={String(option.value)}
                  variant={props.value === option.value ? 'default' : 'ghost'}
                  className='h-24 flex-col gap-2 border'
                  onClick={() => props.onChange(option.value)}
                >
                  <span
                    aria-hidden='true'
                    className={cn(
                      'border-current/70 block max-h-9 max-w-12 rounded-sm border-2',
                      aspectClassName
                    )}
                  />
                  {t(option.label)}
                </Button>
              )
            })}
            {props.field.allow_custom && (
              <Button
                variant={customValue ? 'default' : 'ghost'}
                className='h-24 flex-col gap-2 border'
                onClick={() => props.onChange(customValue ?? '4:3')}
              >
                <span
                  aria-hidden='true'
                  className='block size-9 rounded-sm border-2 border-dashed border-current/70'
                />
                {t('Custom ratio')}
              </Button>
            )}
          </div>
          {customValue && (
            <Input
              value={customValue}
              inputMode='decimal'
              aria-label={t('Custom ratio')}
              placeholder='4:3'
              onChange={(event) => props.onChange(event.target.value)}
            />
          )}
          {props.field.description && (
            <p className='text-muted-foreground text-xs leading-5 font-normal'>
              {t(props.field.description)}
            </p>
          )}
        </div>
      )
    }
    return (
      <div className='grid gap-2 text-sm font-medium'>
        <span>{label}</span>
        <div className='grid auto-cols-fr grid-flow-col gap-1 rounded-lg border p-1'>
          {options.map((option) => (
            <Button
              key={String(option.value)}
              variant={props.value === option.value ? 'default' : 'ghost'}
              className='w-full'
              onClick={() => props.onChange(option.value)}
            >
              {t(option.label)}
            </Button>
          ))}
        </div>
        {props.field.description && (
          <p className='text-muted-foreground text-xs leading-5 font-normal'>
            {t(props.field.description)}
          </p>
        )}
      </div>
    )
  }
  if (props.field.type === 'select') {
    const selectItems = options.map((option) => ({
      value: String(option.value),
      label: t(option.label),
    }))
    return (
      <label className='grid gap-2 text-sm font-medium'>
        {label}
        <Select
          items={selectItems}
          value={String(props.value ?? '')}
          onValueChange={(value) =>
            props.onChange(
              options.find((option) => String(option.value) === String(value))
                ?.value ?? value
            )
          }
        >
          <SelectTrigger className='w-full'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {options.map((option) => (
              <SelectItem
                key={String(option.value)}
                value={String(option.value)}
              >
                {t(option.label)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {props.field.description && (
          <span className='text-muted-foreground text-xs leading-5 font-normal'>
            {t(props.field.description)}
          </span>
        )}
      </label>
    )
  }
  const numeric = ['number', 'slider', 'ratio'].includes(props.field.type ?? '')
  return (
    <label className='grid gap-2 text-sm font-medium'>
      {label}
      <Input
        type={numeric ? 'number' : 'text'}
        value={String(props.value ?? '')}
        placeholder={props.field.placeholder}
        onChange={(event) =>
          props.onChange(
            numeric ? Number(event.target.value) : event.target.value
          )
        }
      />
    </label>
  )
}
