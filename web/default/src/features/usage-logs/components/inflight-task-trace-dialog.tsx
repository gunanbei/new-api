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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import axios from 'axios'
import { Download, Loader2 } from 'lucide-react'
import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  CodeBlock,
  CodeBlockCopyButton,
} from '@/components/ai-elements/code-block'
import { Dialog } from '@/components/dialog'
import { StatusBadge } from '@/components/status-badge'
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
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { api } from '@/lib/api'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

import {
  cacheInflightTraceArchive,
  readInflightTraceArchiveRecord,
  supportsInflightTraceLocalCache,
} from '../lib/inflight-trace-local-cache'

import {
  InflightDetailRow,
  InflightDetailSection,
} from './inflight-detail-primitives'
import type {
  InflightTaskTrace,
  InflightTraceHTTPPart,
} from '../types/inflight-trace'
import {
  formatJsonText,
  formatTraceJsonForDisplay,
  isHeavyTraceJsonBody,
  shouldUseInstantTraceBodyRender,
} from '../lib/inflight-trace-body-display'
import {
  appendSseTraceText,
  createIncrementalSseParserState,
  formatSseEventsForCopy,
  parseSseTrace,
  type IncrementalSseParserState,
  type SseParseStrategy,
} from '../lib/inflight-sse-parse'

type InflightTaskTraceDialogProps = {
  requestId: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
  useMock: boolean
  summary?: {
    model_name?: string
    status?: string
    is_stream?: boolean
    has_trace?: boolean
  }
}

type BodyViewMode = 'formatted' | 'raw'
type SseViewMode = 'raw' | 'events' | 'parsed'

const archiveAutoDownloadStorageKey = 'usage-logs:inflight-trace:auto-download'

function isTerminalInflightStatus(status?: string) {
  return status === 'completed' || status === 'failed'
}

const SSE_STRATEGY_OPTIONS: Array<{
  value: SseParseStrategy
  labelKey: string
}> = [
  { value: 'openai', labelKey: 'OpenAI API compatible format' },
  { value: 'gemini', labelKey: 'Gemini API compatible format' },
  { value: 'claude', labelKey: 'Claude API compatible format' },
  { value: 'ollama_generate', labelKey: 'Ollama API compatible format (Generate)' },
  { value: 'ollama_chat', labelKey: 'Ollama API compatible format (Chat)' },
  { value: 'custom', labelKey: 'Custom JSONPath extraction' },
]

function formatBytes(bytes?: number) {
  if (!bytes) return '0 B'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
}

function decodeTraceBody(part?: InflightTraceHTTPPart) {
  if (!part?.body || part.body_encoding === 'empty') {
    return new Uint8Array()
  }
  if (part.body_encoding === 'base64') {
    const binary = atob(part.body)
    const bytes = new Uint8Array(binary.length)
    for (let i = 0; i < binary.length; i += 1) {
      bytes[i] = binary.charCodeAt(i)
    }
    return bytes
  }
  return new TextEncoder().encode(part.body)
}

function isTraceBodyBinary(part?: InflightTraceHTTPPart) {
  if (!part?.body || part.body_encoding === 'empty') {
    return false
  }
  if (part.body_encoding === 'base64') {
    return true
  }
  return isBinaryContentType(part.content_type)
}

function getTraceBodyPlainText(
  part: InflightTraceHTTPPart | undefined,
  options?: { formattedJson?: boolean }
) {
  if (!part?.body || part.body_encoding !== 'text') {
    return ''
  }
  if (options?.formattedJson && isJsonContentType(part.content_type)) {
    return formatJsonText(part.body)
  }
  return part.body
}

function downloadTraceBody(
  part: InflightTraceHTTPPart | undefined,
  filenameBase: string,
  content?: string
) {
  if (!part?.body) {
    return
  }

  if (isTraceBodyBinary(part)) {
    const bytes = decodeTraceBody(part)
    const blob = new Blob([bytes], {
      type: part.content_type || 'application/octet-stream',
    })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = `${filenameBase}.bin`
    anchor.click()
    URL.revokeObjectURL(url)
    return
  }

  const text =
    content ??
    getTraceBodyPlainText(part, {
      formattedJson: isJsonContentType(part.content_type),
    })
  const blob = new Blob([text], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = `${filenameBase}.txt`
  anchor.click()
  URL.revokeObjectURL(url)
}

function isJsonContentType(contentType?: string) {
  return Boolean(contentType?.includes('application/json'))
}

function isMultipartContentType(contentType?: string) {
  return Boolean(contentType?.includes('multipart/'))
}

function isSseContentType(contentType?: string) {
  return Boolean(contentType?.includes('text/event-stream'))
}

function isBinaryContentType(contentType?: string) {
  if (!contentType) return false
  return (
    contentType.startsWith('image/') ||
    contentType.startsWith('audio/') ||
    contentType.includes('octet-stream') ||
    contentType.includes('application/pdf')
  )
}

function buildHttpLine(part?: InflightTraceHTTPPart, response = false) {
  if (!part) return '-'
  if (response) {
    const status = part.status_code || 0
    return `HTTP ${status}`
  }
  const query = part.query ? `?${part.query}` : ''
  return `${part.method || 'GET'} ${part.path || '/'}${query}`
}

function HeadersTable(props: {
  headers?: Record<string, string>
  onCopy: () => void
  copyLabel: string
}) {
  const { t } = useTranslation()
  const entries = Object.entries(props.headers ?? {})
  return (
    <div className='space-y-2'>
      <div className='flex items-center justify-between gap-2'>
        <span className='text-sm font-medium'>{props.copyLabel}</span>
        <Button type='button' variant='outline' size='sm' onClick={props.onCopy}>
          {t('Copy Headers')}
        </Button>
      </div>
      {entries.length === 0 ? (
        <p className='text-muted-foreground text-xs'>-</p>
      ) : (
        <div className='border-border overflow-x-auto rounded-md border'>
          <table className='w-full text-xs'>
            <thead>
              <tr className='bg-muted/40 border-b'>
                <th className='px-3 py-2 text-left font-medium'>Header</th>
                <th className='px-3 py-2 text-left font-medium'>Value</th>
              </tr>
            </thead>
            <tbody>
              {entries.map(([key, value]) => (
                <tr key={key} className='border-b last:border-b-0'>
                  <td className='px-3 py-2 font-mono break-all'>{key}</td>
                  <td className='px-3 py-2 font-mono break-all'>{value}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

const traceInfoAlertClassName = 'border-border bg-muted/30'
const tracePreClassName =
  'border-border bg-muted/30 max-h-[50vh] overflow-auto rounded-md border p-3 text-xs whitespace-pre-wrap'

function TruncationAlert(props: {
  truncated?: boolean
  bodyBytes?: number
}) {
  const { t } = useTranslation()
  if (!props.truncated) return null
  return (
    <Alert className={traceInfoAlertClassName}>
      <AlertDescription>
        {t('Content truncated to first {{size}}', {
          size: formatBytes(props.bodyBytes),
        })}
      </AlertDescription>
    </Alert>
  )
}

function TraceBodyPanel(props: {
  part?: InflightTraceHTTPPart
  truncated?: boolean
  filenameBase: string
  bodyLabel: string
  liveUpdate?: boolean
  traceKind?: string
}) {
  const { t } = useTranslation()
  const { copyToClipboard } = useCopyToClipboard()
  const preRef = useRef<HTMLPreElement>(null)
  const incrementalSseRef = useRef<IncrementalSseParserState | null>(null)
  const textBody = props.part?.body_encoding === 'text' ? props.part.body || '' : ''
  const contentType = props.part?.content_type
  const isJson = isJsonContentType(contentType)
  const isMultipart = isMultipartContentType(contentType)
  const isSse = isSseContentType(contentType)
  const isBinary = isTraceBodyBinary(props.part)
  const heavyBody = useMemo(
    () => isHeavyTraceJsonBody(textBody, contentType, props.traceKind),
    [contentType, props.traceKind, textBody]
  )
  const [viewMode, setViewMode] = useState<BodyViewMode>(() =>
    heavyBody ? 'raw' : 'formatted'
  )
  const [sseViewMode, setSseViewMode] = useState<SseViewMode>('raw')
  const [sseStrategy, setSseStrategy] = useState<SseParseStrategy>('openai')
  const [customJsonPath, setCustomJsonPath] = useState('$.choices[0].delta.content')
  const useInstantPre = shouldUseInstantTraceBodyRender({
    liveUpdate: props.liveUpdate,
    isSse,
  })

  useEffect(() => {
    if (props.liveUpdate) {
      setViewMode('raw')
    } else {
      incrementalSseRef.current = null
    }
  }, [props.liveUpdate])

  const parsedSse = useMemo(() => {
    if (!isSse || !textBody) {
      return null
    }
    if (props.liveUpdate) {
      if (!incrementalSseRef.current) {
        incrementalSseRef.current = createIncrementalSseParserState()
      }
      return appendSseTraceText(
        incrementalSseRef.current,
        textBody,
        sseStrategy,
        customJsonPath
      )
    }
    return parseSseTrace(textBody, sseStrategy, customJsonPath)
  }, [customJsonPath, isSse, props.liveUpdate, sseStrategy, textBody])

  const displayText = useMemo(() => {
    if (!textBody) return ''
    if (useInstantPre) {
      return textBody
    }
    if (viewMode === 'formatted' && isJson) {
      return heavyBody ? formatTraceJsonForDisplay(textBody) : formatJsonText(textBody)
    }
    return textBody
  }, [heavyBody, isJson, textBody, useInstantPre, viewMode])

  useEffect(() => {
    if (!props.liveUpdate || !preRef.current) {
      return
    }
    preRef.current.scrollTop = preRef.current.scrollHeight
  }, [parsedSse?.concatenated, props.liveUpdate, sseViewMode, textBody])

  const sseEventsToRender = useMemo(() => {
    if (!parsedSse?.events.length) {
      return []
    }
    if (props.liveUpdate) {
      return parsedSse.events.slice(-12)
    }
    return parsedSse.events
  }, [parsedSse?.events, props.liveUpdate])

  const currentCopyText = useMemo(() => {
    if (!textBody) {
      return ''
    }
    if (isSse) {
      if (sseViewMode === 'raw') {
        return textBody
      }
      if (sseViewMode === 'parsed') {
        return parsedSse?.concatenated || ''
      }
      if (parsedSse) {
        return formatSseEventsForCopy(parsedSse.events)
      }
      return textBody
    }
    return displayText || textBody
  }, [displayText, isSse, parsedSse, sseViewMode, textBody])

  let textBodyContent: ReactNode = null
  if (!isBinary && !isSse && textBody) {
    if (useInstantPre || viewMode === 'raw' || heavyBody) {
      textBodyContent = (
        <pre ref={preRef} className={tracePreClassName}>
          {displayText}
        </pre>
      )
    } else if (isJson && viewMode === 'formatted') {
      textBodyContent = (
        <CodeBlock
          code={displayText}
          language='json'
          collapsedLines={12}
          maxExpandedLines={32}
          enableCollapse
        >
          <CodeBlockCopyButton />
        </CodeBlock>
      )
    } else {
      textBodyContent = (
        <pre ref={preRef} className={tracePreClassName}>
          {displayText}
        </pre>
      )
    }
  }

  if (!isBinary && isSse && sseViewMode === 'raw' && textBody) {
    textBodyContent = (
      <pre ref={preRef} className={tracePreClassName}>
        {textBody}
      </pre>
    )
  }

  if (!isBinary && isSse && sseViewMode === 'parsed') {
    textBodyContent = (
      <pre ref={preRef} className={tracePreClassName}>
        {parsedSse?.concatenated || '-'}
      </pre>
    )
  }

  return (
    <div className='space-y-3'>
      <TruncationAlert truncated={props.truncated} bodyBytes={props.part?.body_bytes} />
      <div className='flex flex-wrap items-center gap-2'>
        <span className='text-sm font-medium'>{props.bodyLabel}</span>
        <div className='ml-auto flex flex-wrap items-center justify-end gap-2'>
          {isJson && !isSse ? (
            <>
              <Button
                type='button'
                size='sm'
                variant={viewMode === 'formatted' ? 'default' : 'outline'}
                onClick={() => setViewMode('formatted')}
              >
                {t('Formatted JSON')}
              </Button>
              <Button
                type='button'
                size='sm'
                variant={viewMode === 'raw' ? 'default' : 'outline'}
                onClick={() => setViewMode('raw')}
              >
                {t('Raw')}
              </Button>
            </>
          ) : null}
          {isSse ? (
            <>
              <Button
                type='button'
                size='sm'
                variant={sseViewMode === 'raw' ? 'default' : 'outline'}
                onClick={() => setSseViewMode('raw')}
              >
                {t('Raw SSE')}
              </Button>
              <Button
                type='button'
                size='sm'
                variant={sseViewMode === 'events' ? 'default' : 'outline'}
                onClick={() => setSseViewMode('events')}
              >
                {t('Event List')}
              </Button>
              <Button
                type='button'
                size='sm'
                variant={sseViewMode === 'parsed' ? 'default' : 'outline'}
                onClick={() => setSseViewMode('parsed')}
              >
                {t('Parsed output')}
              </Button>
              <Select
                items={SSE_STRATEGY_OPTIONS.map((option) => ({
                  value: option.value,
                  label: t(option.labelKey),
                }))}
                value={sseStrategy}
                onValueChange={(value) => {
                  if (value) {
                    setSseStrategy(value as SseParseStrategy)
                  }
                }}
              >
                <SelectTrigger className='h-8 w-[min(100%,220px)] text-xs'>
                  <SelectValue placeholder={t('SSE parse strategy')} />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {SSE_STRATEGY_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {t(option.labelKey)}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              {sseStrategy === 'custom' ? (
                <Input
                  className='h-8 w-[min(100%,260px)] font-mono text-xs'
                  value={customJsonPath}
                  onChange={(event) => setCustomJsonPath(event.currentTarget.value)}
                  placeholder={t('Supports JSONPath, e.g. $.choices[0].delta.content')}
                />
              ) : null}
            </>
          ) : null}
          {textBody || props.part?.body ? (
            <Button
              type='button'
              size='sm'
              variant='outline'
              onClick={() => copyToClipboard(currentCopyText)}
            >
              {t('Copy All')}
            </Button>
          ) : null}
          {props.part?.body ? (
            <Button
              type='button'
              size='sm'
              variant='outline'
              onClick={() =>
                downloadTraceBody(props.part, props.filenameBase, currentCopyText)
              }
            >
              <Download />
              {t('Download')}
            </Button>
          ) : null}
        </div>
      </div>

      {isMultipart ? (
        <p className='text-muted-foreground text-sm'>
          {t('Binary multipart ({{size}}). Download to inspect.', {
            size: formatBytes(props.part?.body_bytes),
          })}
        </p>
      ) : null}

      {isBinary && !isMultipart ? (
        <p className='text-muted-foreground text-sm'>
          {t('Binary content ({{size}}, {{type}}). Download to inspect.', {
            size: formatBytes(props.part?.body_bytes),
            type: contentType || 'binary',
          })}
        </p>
      ) : null}

      {!isBinary && !textBody ? (
        <p className='text-muted-foreground text-sm'>-</p>
      ) : null}

      {!isBinary && isSse && sseViewMode === 'events' && sseEventsToRender.length > 0 ? (
        <div className='space-y-2'>
          {props.liveUpdate ? (
            <p className='text-muted-foreground text-xs'>
              {t('Showing the latest {{count}} events while response is in progress', {
                count: sseEventsToRender.length,
              })}
            </p>
          ) : null}
          {sseEventsToRender.map((event) => (
            <Collapsible key={event.index} defaultOpen={event.index <= 3}>
              <CollapsibleTrigger className='border-border hover:bg-muted/50 flex w-full items-center gap-2 rounded-md border px-3 py-2 text-left text-xs'>
                <span className='font-mono'>[{event.index}]</span>
                <span className='truncate'>
                  {event.extracted
                    ? `${event.raw.split('\n')[0]} → ${event.extracted}`
                    : event.raw.split('\n')[0]}
                </span>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <pre className={cn('mt-2 overflow-x-auto', tracePreClassName)}>
                  {event.raw}
                </pre>
              </CollapsibleContent>
            </Collapsible>
          ))}
        </div>
      ) : null}

      {textBodyContent}
    </div>
  )
}

const MOCK_INFLIGHT_TRACE: InflightTaskTrace = {
  request_id: 'req_inflight_mock_002',
  user_id: 1,
  status: 'failed',
  kind: 'chat',
  model_name: 'gpt-4.1',
  is_stream: true,
  created_at: 1751905000,
  updated_at: 1751905016,
  recorded_at: 1751905016,
  flags: {
    request_truncated: false,
    response_truncated: false,
    response_incomplete: false,
  },
  client_request: {
    method: 'POST',
    path: '/v1/chat/completions',
    protocol: 'HTTP/1.1',
    headers: {
      'Content-Type': 'application/json',
      Authorization: '***',
    },
    body_encoding: 'text',
    body: JSON.stringify(
      {
        model: 'gpt-4.1',
        messages: [{ role: 'user', content: 'Hello' }],
        stream: true,
      },
      null,
      2
    ),
    body_bytes: 96,
    content_type: 'application/json',
  },
  client_response: {
    status_code: 200,
    headers: {
      'Content-Type': 'text/event-stream',
    },
    body_encoding: 'text',
    body: [
      'data: {"id":"chatcmpl-mock","choices":[{"delta":{"content":"Hi"}}]}',
      '',
      'data: {"id":"chatcmpl-mock","choices":[{"delta":{"content":" there"}}]}',
      '',
      'data: [DONE]',
      '',
    ].join('\n'),
    body_bytes: 180,
    content_type: 'text/event-stream',
  },
}

async function fetchInflightTaskTrace(requestId: string, useMock: boolean) {
  if (useMock) {
    if (requestId === MOCK_INFLIGHT_TRACE.request_id) {
      return MOCK_INFLIGHT_TRACE
    }
    return null
  }
  try {
    const res = await api.get<{ success: boolean; data?: InflightTaskTrace }>(
      `/api/log/inflight/self/${requestId}/trace`,
      { disableDuplicate: true, skipBusinessError: true, skipErrorHandler: true }
    )
    if (!res.data.success || !res.data.data) {
      return null
    }
    return res.data.data
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.status === 404) {
      return null
    }
    throw error
  }
}

export function InflightTaskTraceDialog(props: InflightTaskTraceDialogProps) {
  const { t } = useTranslation()
  const { copyToClipboard } = useCopyToClipboard()
  const queryClient = useQueryClient()
  const [isCachingArchive, setIsCachingArchive] = useState(false)
  const [showArchiveDownloadConfirm, setShowArchiveDownloadConfirm] = useState(false)
  const [autoDownloadArchive, setAutoDownloadArchive] = useState(
    () => localStorage.getItem(archiveAutoDownloadStorageKey) === 'true'
  )
  const traceQueryKey = useMemo(
    () => ['inflight-trace', props.requestId, props.useMock],
    [props.requestId, props.useMock]
  )
  const { data: trace, isLoading, isError, isFetched } = useQuery({
    queryKey: traceQueryKey,
    queryFn: () => {
      const requestId = props.requestId
      if (!requestId) {
        return Promise.resolve(null)
      }
      return fetchInflightTaskTrace(requestId, props.useMock)
    },
    enabled: props.open && !!props.requestId,
    refetchInterval: (query) => {
      if (!props.open || props.useMock || !props.requestId) {
        return false
      }
      const data = query.state.data
      if (
        data &&
        isTerminalInflightStatus(data.status) &&
        !data.flags.in_progress
      ) {
        return false
      }
      const status = data?.status ?? props.summary?.status
      if (
        status &&
        isTerminalInflightStatus(status) &&
        props.summary?.has_trace === false
      ) {
        return false
      }
      return 2000
    },
    refetchIntervalInBackground: false,
  })

  const liveStatus = trace?.status ?? props.summary?.status

  const shouldPoll = useMemo(() => {
    if (!props.open || props.useMock || !props.requestId) {
      return false
    }
    if (
      trace &&
      isTerminalInflightStatus(trace.status) &&
      !trace.flags.in_progress
    ) {
      return false
    }
    if (
      liveStatus &&
      isTerminalInflightStatus(liveStatus) &&
      props.summary?.has_trace === false
    ) {
      return false
    }
    return true
  }, [
    liveStatus,
    props.open,
    props.requestId,
    props.summary?.has_trace,
    props.useMock,
    trace,
  ])

  const awaitingSnapshot = !trace && shouldPoll && !isError
  const isDialogOpen = props.open
  const isMockTrace = props.useMock
  const closeDialog = props.onOpenChange

  useEffect(() => {
    if (
      !props.requestId ||
      !trace?.archive ||
      trace.client_request?.body_encoding !== 'disk'
    ) {
      return
    }
    let cancelled = false
    void readInflightTraceArchiveRecord(trace.archive.file_name, props.requestId)
      .then((cachedTrace) => {
        if (cancelled || !cachedTrace) return
        queryClient.setQueryData<InflightTaskTrace>(traceQueryKey, {
          ...cachedTrace,
          archive: trace.archive,
        })
      })
    return () => {
      cancelled = true
    }
  }, [props.requestId, queryClient, trace, traceQueryKey])

  useEffect(() => {
    if (
      !isDialogOpen ||
      isMockTrace ||
      !isFetched ||
      isLoading ||
      isError ||
      shouldPoll
    ) {
      return
    }
    if (!trace) {
      toast.error(
        t('This inflight record does not exist or has been cleaned up')
      )
      closeDialog(false)
    }
  }, [
    isDialogOpen,
    isMockTrace,
    closeDialog,
    isFetched,
    isLoading,
    isError,
    shouldPoll,
    trace,
    t,
  ])

  const description = [
    props.requestId,
    props.summary?.model_name || trace?.model_name,
    props.summary?.status || trace?.status,
    (props.summary?.is_stream ?? trace?.is_stream) ? t('Yes') : t('No'),
  ]
    .filter(Boolean)
    .join(' · ')

  const cacheArchive = async () => {
    if (!props.requestId || !trace?.archive) return
    if (!supportsInflightTraceLocalCache()) {
      toast.error(t('Local cache is only supported in Chromium browsers.'))
      return
    }
    setShowArchiveDownloadConfirm(false)
    setIsCachingArchive(true)
    try {
      const response = await api.get(`/api/log/inflight/self/${props.requestId}/trace/archive`, { responseType: 'blob' })
      await cacheInflightTraceArchive(trace.archive.file_name, response.data)
      const cachedTrace = await readInflightTraceArchiveRecord(trace.archive.file_name, props.requestId)
      if (cachedTrace) {
        queryClient.setQueryData<InflightTaskTrace>(traceQueryKey, {
          ...cachedTrace,
          archive: trace.archive,
        })
      }
      toast.success(t('Archive downloaded to local cache.'))
    } catch {
      toast.error(t('Failed to download archive'))
    } finally {
      setIsCachingArchive(false)
    }
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Debug Log')}
      description={description}
      contentClassName='max-h-[calc(100dvh-2rem)] overflow-hidden max-sm:w-screen max-sm:max-w-none max-sm:rounded-none max-sm:p-4 sm:max-w-4xl'
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <Button
          variant='outline'
          onClick={() => props.onOpenChange(false)}
          className='w-full sm:w-auto'
        >
          {t('Close')}
        </Button>
      }
    >
      {isLoading || awaitingSnapshot ? (
        <p className='text-muted-foreground py-6 text-sm'>{t('Loading...')}</p>
      ) : null}
      {isError ? (
        <p className='text-muted-foreground py-6 text-sm'>{t('Failed to load')}</p>
      ) : null}
      {trace ? (
        <Tabs defaultValue='overview' className='min-w-0'>
          {trace.archive && trace.client_request?.body_encoding === 'disk' ? (
            <Alert className='mb-3'>
              <AlertDescription className='flex flex-wrap items-center justify-between gap-2'>
                <span>{t('This debug log is archived remotely. Download it locally to load request and response bodies.')}</span>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={() => {
                    if (autoDownloadArchive) {
                      cacheArchive()
                      return
                    }
                    setShowArchiveDownloadConfirm(true)
                  }}
                  disabled={isCachingArchive}
                >
                  {isCachingArchive ? <Loader2 className='animate-spin' /> : <Download />}
                  {t('Download archive')}
                </Button>
              </AlertDescription>
            </Alert>
          ) : null}
          <AlertDialog open={showArchiveDownloadConfirm} onOpenChange={setShowArchiveDownloadConfirm}>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{t('Download archive')}</AlertDialogTitle>
                <AlertDialogDescription>{t('Download this archive to your selected local cache directory?')}</AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <label className='mr-auto flex items-center gap-2 text-sm'>
                  <Checkbox
                    checked={autoDownloadArchive}
                    onCheckedChange={(checked) => {
                      const enabled = checked === true
                      setAutoDownloadArchive(enabled)
                      localStorage.setItem(archiveAutoDownloadStorageKey, String(enabled))
                    }}
                  />
                  {t('Download archives automatically in this browser')}
                </label>
                <AlertDialogCancel>{t('Cancel')}</AlertDialogCancel>
                <AlertDialogAction onClick={cacheArchive}>{t('Download archive')}</AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
          <TabsList>
            <TabsTrigger value='overview'>{t('Overview')}</TabsTrigger>
            <TabsTrigger value='request'>{t('Request')}</TabsTrigger>
            <TabsTrigger value='response'>{t('Response')}</TabsTrigger>
          </TabsList>
          <TabsContent value='overview' className='space-y-3'>
            <div className='flex flex-wrap items-center gap-2'>
              <StatusBadge
                label={trace.status}
                variant={trace.status === 'completed' ? 'success' : 'danger'}
                size='sm'
                copyable={false}
              />
            </div>
            {(trace.flags.request_truncated ||
              trace.flags.response_truncated) && (
              <Alert className={traceInfoAlertClassName}>
                <AlertDescription>
                  {t('Content truncated to first {{size}}', {
                    size: formatBytes(
                      trace.flags.request_truncated
                        ? trace.client_request?.body_bytes
                        : trace.client_response?.body_bytes
                    ),
                  })}
                </AlertDescription>
              </Alert>
            )}
            {trace.flags.in_progress ? (
              <Alert className={traceInfoAlertClassName}>
                <Loader2 className='animate-spin' />
                <AlertDescription>
                  {t(
                    'Response in progress. Content will update automatically.'
                  )}
                </AlertDescription>
              </Alert>
            ) : null}
            {trace.flags.response_incomplete && !trace.flags.in_progress ? (
              <Alert className={traceInfoAlertClassName}>
                <AlertDescription>
                  {t('Response may be incomplete')}
                </AlertDescription>
              </Alert>
            ) : null}
            <InflightDetailSection label={t('Overview')}>
              <InflightDetailRow
                label={t('Request ID')}
                value={trace.request_id}
                mono
              />
              <InflightDetailRow label={t('Status')} value={trace.status} />
              <InflightDetailRow label={t('Type')} value={trace.kind} />
              <InflightDetailRow
                label={t('Stream')}
                value={trace.is_stream ? t('Yes') : t('No')}
              />
              <InflightDetailRow
                label={t('Recorded At')}
                value={dayjs.unix(trace.recorded_at).format('YYYY-MM-DD HH:mm:ss')}
              />
              <InflightDetailRow
                label={t('Request Body')}
                value={formatBytes(trace.client_request?.body_bytes)}
              />
              <InflightDetailRow
                label={t('Response Body')}
                value={formatBytes(trace.client_response?.body_bytes)}
              />
            </InflightDetailSection>
          </TabsContent>
          <TabsContent value='request' className='space-y-4'>
            <p className='font-mono text-sm'>{buildHttpLine(trace.client_request)}</p>
            <HeadersTable
              headers={trace.client_request?.headers}
              copyLabel={t('Request Headers')}
              onCopy={() =>
                copyToClipboard(
                  JSON.stringify(trace.client_request?.headers ?? {}, null, 2)
                )
              }
            />
            <TraceBodyPanel
              part={trace.client_request}
              truncated={trace.flags.request_truncated}
              filenameBase={`${trace.request_id}-request`}
              bodyLabel={t('Request Body')}
              liveUpdate={trace.flags.in_progress}
              traceKind={trace.kind}
            />
          </TabsContent>
          <TabsContent value='response' className='space-y-4'>
            <p className='font-mono text-sm'>
              {buildHttpLine(trace.client_response, true)}
            </p>
            <HeadersTable
              headers={trace.client_response?.headers}
              copyLabel={t('Response Headers')}
              onCopy={() =>
                copyToClipboard(
                  JSON.stringify(trace.client_response?.headers ?? {}, null, 2)
                )
              }
            />
            <TraceBodyPanel
              part={trace.client_response}
              truncated={trace.flags.response_truncated}
              filenameBase={`${trace.request_id}-response`}
              bodyLabel={t('Response Body')}
              liveUpdate={trace.flags.in_progress}
              traceKind={trace.kind}
            />
          </TabsContent>
        </Tabs>
      ) : null}
    </Dialog>
  )
}

export { MOCK_INFLIGHT_TRACE }
