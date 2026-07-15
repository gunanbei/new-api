import { zodResolver } from "@hookform/resolvers/zod";
import {
  Copy,
  Loader2,
  Pencil,
  Power,
  Radar,
  Star,
  Trash2,
  UserRoundPen,
} from "lucide-react";
import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";

import { CopyButton } from "@/components/copy-button";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { BadgeCell } from "@/components/data-table/core/badge-cell";
import { StaticDataTable } from "@/components/data-table/static/static-data-table";
import { Dialog } from "@/components/dialog";
import { StatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";

import { SettingsSection } from "../../components/settings-section";
import {
  SettingsSwitchContent,
  SettingsSwitchItem,
} from "../../components/settings-form-layout";
import {
  copyFileUploadChannel,
  createFileUploadChannel,
  deleteFileUploadChannel,
  getFileUploadChannelProbeFileInfo,
  getFileUploadChannels,
  probeDraftFileUploadChannel,
  probeSavedFileUploadChannel,
  setDefaultFileUploadChannel,
  updateFileUploadChannel,
  updateFileUploadChannelStatus,
} from "./api";
import type {
  FileUploadChannel,
  FileUploadChannelConfig,
  FileUploadChannelProbeResult,
  FileUploadChannelType,
} from "./types";

const channelFormSchema = z.object({
  name: z.string().trim().min(1),
  type: z.enum(["1", "2", "3"]),
  status: z.enum(["0", "1"]),
  is_default: z.enum(["0", "1"]),
  chunk_threshold: z.string(),
  chunk_size: z.string(),
  max_size: z.string(),
  base_url: z.string().optional(),
  username: z.string().optional(),
  password: z.string().optional(),
  auth_type: z.string().optional(),
  path_prefix: z.string().optional(),
  public_base_url: z.string().optional(),
  timeout_ms: z.string().optional(),
  api_token: z.string().optional(),
  upload_channel: z.string().optional(),
  upload_folder: z.string().optional(),
  return_format: z.string().optional(),
  endpoint: z.string().optional(),
  region: z.string().optional(),
  bucket: z.string().optional(),
  access_key_id: z.string().optional(),
  secret_access_key: z.string().optional(),
  force_path_style: z.boolean(),
  key_prefix: z.string().optional(),
});

type FileUploadChannelFormValues = z.infer<typeof channelFormSchema>;

const emptyConfig = {
  base_url: "",
  username: "",
  password: "",
  auth_type: "basic",
  path_prefix: "",
  public_base_url: "",
  timeout_ms: "600000",
  api_token: "",
  upload_channel: "cfr2",
  upload_folder: "",
  return_format: "default",
  endpoint: "",
  region: "",
  bucket: "",
  access_key_id: "",
  secret_access_key: "",
  force_path_style: false,
  key_prefix: "",
};

function buildEmptyFormValues(): FileUploadChannelFormValues {
  return {
    name: "",
    type: "1",
    status: "1",
    is_default: "0",
    chunk_threshold: "16777216",
    chunk_size: "8388608",
    max_size: "0",
    ...emptyConfig,
  };
}

function formatBytes(bytes: number) {
  if (!Number.isFinite(bytes)) return "-";
  if (bytes === 0) return "0 B";
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  let value = Math.abs(bytes);
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  return `${bytes < 0 ? "-" : ""}${value.toFixed(value >= 10 || unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function getTypeLabel(t: (key: string) => string, type: FileUploadChannelType) {
  if (type === "1") return t("WebDAV");
  if (type === "2") return t("Cloudflare ImageBed");
  return t("S3 Compatible Storage");
}

function getConfigValue(
  config: FileUploadChannelConfig | undefined,
  key: string,
): string {
  const value = config?.[key];
  if (typeof value === "string" && !value.includes("*")) return value;
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return "";
}

function getSensitiveConfigHint(
  config: FileUploadChannelConfig | undefined,
  key: string,
): string {
  const value = config?.[key];
  return typeof value === "string" && value.includes("*") ? value : "";
}

function getConfigBoolean(
  config: FileUploadChannelConfig | undefined,
  key: string,
) {
  return Boolean(config?.[key]);
}

function buildFormValues(
  channel?: FileUploadChannel | null,
): FileUploadChannelFormValues {
  if (!channel) return buildEmptyFormValues();

  const config = channel.config_proflle;
  const type = channel.type;
  return {
    name: channel.name,
    type,
    status: channel.status,
    is_default: channel.is_default,
    chunk_threshold: String(channel.chunk_threshold ?? 16777216),
    chunk_size: String(channel.chunk_size ?? 8388608),
    max_size: String(channel.max_size ?? 0),
    base_url:
      type === "1" || type === "2" ? getConfigValue(config, "base_url") : "",
    username: type === "1" ? getConfigValue(config, "username") : "",
    password: type === "1" ? "" : "",
    auth_type:
      type === "1" ? getConfigValue(config, "auth_type") || "basic" : "basic",
    path_prefix: type === "1" ? getConfigValue(config, "path_prefix") : "",
    public_base_url:
      type === "1" || type === "2" || type === "3"
        ? getConfigValue(config, "public_base_url")
        : "",
    timeout_ms:
      type === "1"
        ? getConfigValue(config, "timeout_ms") || "600000"
        : "600000",
    api_token: type === "2" ? "" : "",
    upload_channel:
      type === "2"
        ? getConfigValue(config, "upload_channel") || "cfr2"
        : "cfr2",
    upload_folder: type === "2" ? getConfigValue(config, "upload_folder") : "",
    return_format:
      type === "2"
        ? getConfigValue(config, "return_format") || "default"
        : "default",
    endpoint: type === "3" ? getConfigValue(config, "endpoint") : "",
    region: type === "3" ? getConfigValue(config, "region") : "",
    bucket: type === "3" ? getConfigValue(config, "bucket") : "",
    access_key_id: type === "3" ? getConfigValue(config, "access_key_id") : "",
    secret_access_key: type === "3" ? "" : "",
    force_path_style:
      type === "3" ? getConfigBoolean(config, "force_path_style") : false,
    key_prefix: type === "3" ? getConfigValue(config, "key_prefix") : "",
  };
}

function buildConfigPayload(
  values: FileUploadChannelFormValues,
): Record<string, unknown> {
  if (values.type === "1") {
    return {
      base_url: values.base_url?.trim() || "",
      username: values.username?.trim() || "",
      password: values.password?.trim() || "",
      auth_type: values.auth_type?.trim() || "basic",
      path_prefix: values.path_prefix?.trim() || "",
      public_base_url: values.public_base_url?.trim() || "",
      timeout_ms: Number(values.timeout_ms || 600000),
    };
  }

  if (values.type === "2") {
    return {
      base_url: values.base_url?.trim() || "",
      api_token: values.api_token?.trim() || "",
      upload_channel: values.upload_channel?.trim() || "cfr2",
      upload_folder: values.upload_folder?.trim() || "",
      return_format: values.return_format?.trim() || "default",
    };
  }

  return {
    endpoint: values.endpoint?.trim() || "",
    region: values.region?.trim() || "",
    bucket: values.bucket?.trim() || "",
    access_key_id: values.access_key_id?.trim() || "",
    secret_access_key: values.secret_access_key?.trim() || "",
    force_path_style: values.force_path_style,
    public_base_url: values.public_base_url?.trim() || "",
    key_prefix: values.key_prefix?.trim() || "",
  };
}

function normalizeSubmitPayload(values: FileUploadChannelFormValues) {
  return {
    name: values.name.trim(),
    type: values.type,
    status: values.status,
    is_default: values.is_default,
    chunk_threshold: Number(values.chunk_threshold || 0),
    chunk_size: Number(values.chunk_size || 0),
    max_size: Number(values.max_size || 0),
    config_proflle: buildConfigPayload(values),
  };
}

function getChannelTypeOptions(t: (key: string) => string) {
  return [
    { value: "1", label: t("WebDAV") },
    { value: "2", label: t("Cloudflare ImageBed") },
    { value: "3", label: t("S3 Compatible Storage") },
  ];
}

function getAuthTypeOptions(t: (key: string) => string) {
  return [
    { value: "basic", label: t("Basic") },
    { value: "bearer", label: t("Bearer") },
  ];
}

function getReturnFormatOptions(t: (key: string) => string) {
  return [
    { value: "default", label: t("Default") },
    { value: "full", label: t("Full") },
  ];
}

function getUploadChannelOptions() {
  return [
    { value: "telegram", label: "TG" },
    { value: "cfr2", label: "Cloudflare R2" },
    { value: "s3", label: "s3" },
    { value: "webdav", label: "webdav" },
  ];
}

function getSelectedLabel(
  options: Array<{ value: string; label: string }>,
  value: string | null | undefined,
) {
  return options.find((option) => option.value === value)?.label ?? value ?? "";
}

function normalizeProbeResult(result: FileUploadChannelProbeResult) {
  return {
    ...result,
    file_name: result.file_name ?? "",
    file_size:
      typeof result.file_size === "number" && Number.isFinite(result.file_size)
        ? result.file_size
        : 0,
    url: result.url ?? "",
  };
}

function normalizeProbeFileInfo(
  info?: {
    file_name?: string;
    file_size?: number;
  } | null,
) {
  return {
    file_name: info?.file_name ?? "test.txt",
    file_size:
      typeof info?.file_size === "number" && Number.isFinite(info.file_size)
        ? info.file_size
        : 0,
  };
}

function mergeProbeResultWithFileInfo(
  result: FileUploadChannelProbeResult | null,
  info: { file_name: string; file_size: number },
) {
  if (!result) return null;
  return {
    ...result,
    file_name: result.file_name || info.file_name,
    file_size:
      typeof result.file_size === "number" && Number.isFinite(result.file_size)
        ? result.file_size
        : info.file_size,
  };
}

const EMPTY_CHANNELS: never[] = [];

type ProbeIntent =
  | { kind: "saved"; channel: FileUploadChannel }
  | { kind: "draft"; payload: Record<string, unknown> };

export function DataManagementSection() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const channelsQuery = useQuery({
    queryKey: ["file-upload-channels"],
    queryFn: () => getFileUploadChannels(),
    staleTime: 30 * 1000,
  });
  const [editingChannel, setEditingChannel] =
    useState<FileUploadChannel | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<FileUploadChannel | null>(
    null,
  );
  const [copyTarget, setCopyTarget] = useState<FileUploadChannel | null>(null);
  const [copySuffix, setCopySuffix] = useState("_copy");
  const [formOpen, setFormOpen] = useState(false);
  const [probeIntent, setProbeIntent] = useState<ProbeIntent | null>(null);
  const [probeResult, setProbeResult] =
    useState<FileUploadChannelProbeResult | null>(null);
  const [probeResultOpen, setProbeResultOpen] = useState(false);

  const probeFileInfoQuery = useQuery({
    queryKey: ["file-upload-channel-probe-file-info"],
    queryFn: getFileUploadChannelProbeFileInfo,
    staleTime: 5 * 60 * 1000,
  });

  const channels = channelsQuery.data?.data ?? EMPTY_CHANNELS;
  const probeFileInfo = normalizeProbeFileInfo(probeFileInfoQuery.data?.data);
  const displayProbeResult = mergeProbeResultWithFileInfo(
    probeResult,
    probeFileInfo,
  );
  const defaultChannel = useMemo(
    () => channels.find((item) => item.is_default === "1") ?? null,
    [channels],
  );

  const openCreate = () => {
    setEditingChannel(null);
    setFormOpen(true);
  };

  const openEdit = (channel: FileUploadChannel) => {
    const latestChannel =
      channels.find((item) => item.id === channel.id) ?? channel;
    setEditingChannel(latestChannel);
    setFormOpen(true);
  };

  const refresh = async () => {
    await queryClient.invalidateQueries({ queryKey: ["file-upload-channels"] });
  };

  const upsertChannelInCache = (channel: FileUploadChannel) => {
    queryClient.setQueryData(
      ["file-upload-channels"],
      (
        current:
          | { success: boolean; message: string; data?: FileUploadChannel[] }
          | undefined,
      ) => {
        const currentItems = current?.data ?? [];
        const nextItems = currentItems.some((item) => item.id === channel.id)
          ? currentItems.map((item) =>
              item.id === channel.id ? channel : item,
            )
          : [channel, ...currentItems];
        return {
          success: true,
          message: "",
          data: nextItems,
        };
      },
    );
  };

  const setDefaultMutation = useMutation({
    mutationFn: async (channel: FileUploadChannel) =>
      setDefaultFileUploadChannel(channel.id),
    onSuccess: async (res) => {
      if (res.success) {
        toast.success(t("Default channel updated"));
        await refresh();
      }
    },
  });

  const toggleStatusMutation = useMutation({
    mutationFn: async (channel: FileUploadChannel) =>
      updateFileUploadChannelStatus(
        channel.id,
        channel.status === "1" ? "0" : "1",
      ),
    onSuccess: async (res) => {
      if (res.success) {
        toast.success(t("Status updated"));
        await refresh();
      }
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (channel: FileUploadChannel) =>
      deleteFileUploadChannel(channel.id),
    onSuccess: async (res) => {
      if (res.success) {
        toast.success(t("Deleted"));
        await refresh();
      }
    },
  });

  const copyMutation = useMutation({
    mutationFn: async ({
      channel,
      suffix,
    }: {
      channel: FileUploadChannel;
      suffix: string;
    }) => copyFileUploadChannel(channel.id, suffix),
    onSuccess: async (res) => {
      if (res.success && res.data) {
        toast.success(t("Copied"));
        upsertChannelInCache(res.data);
        setCopyTarget(null);
        setCopySuffix("_copy");
        await refresh();
      }
    },
  });

  const probeMutation = useMutation({
    mutationFn: async (channel: FileUploadChannel) =>
      probeSavedFileUploadChannel(channel.id),
    onSuccess: (res) => {
      if (res.success && res.data) {
        toast.success(t("Probe succeeded"));
        setProbeResult(normalizeProbeResult(res.data));
        setProbeResultOpen(true);
      }
    },
  });

  const probeDraftMutation = useMutation({
    mutationFn: async (payload: Record<string, unknown>) =>
      probeDraftFileUploadChannel(payload),
    onSuccess: (res) => {
      if (res.success && res.data) {
        toast.success(t("Probe succeeded"));
        setProbeResult(normalizeProbeResult(res.data));
        setProbeResultOpen(true);
      }
    },
  });

  const confirmProbe = async () => {
    if (!probeIntent) return;
    if (probeIntent.kind === "saved") {
      await probeMutation.mutateAsync(probeIntent.channel);
    } else {
      await probeDraftMutation.mutateAsync(probeIntent.payload);
    }
    setProbeIntent(null);
  };

  const saveChannel = async (
    values: FileUploadChannelFormValues,
    editing?: FileUploadChannel | null,
  ) => {
    const payload = normalizeSubmitPayload(values);
    if (editing) {
      const res = await updateFileUploadChannel(editing.id, payload);
      if (res.success && res.data) {
        toast.success(t("Updated"));
        upsertChannelInCache(res.data);
        setFormOpen(false);
        setEditingChannel(null);
        await refresh();
      }
      return;
    }

    const res = await createFileUploadChannel(payload);
    if (res.success && res.data) {
      toast.success(t("Created"));
      upsertChannelInCache(res.data);
      setFormOpen(false);
      await refresh();
    }
  };

  return (
    <SettingsSection title={t("Data Management")}>
      <div className="flex items-center justify-between gap-3">
        <p className="text-muted-foreground text-sm">
          {t(
            "Configure file upload channels and keep secrets server-side only.",
          )}
        </p>
        <Button onClick={openCreate} size="sm">
          <UserRoundPen className="mr-1.5 size-4" />
          {t("Add Channel")}
        </Button>
      </div>

      {defaultChannel ? (
        <div className="border-border bg-muted/30 flex items-center justify-between rounded-lg border px-3 py-2 text-sm">
          <span className="text-muted-foreground">{t("Current default")}</span>
          <span className="font-medium">{defaultChannel.name}</span>
        </div>
      ) : null}

      <StaticDataTable
        data={channels}
        getRowKey={(channel) => channel.id}
        emptyContent={t("No file upload channels configured yet.")}
        columns={[
          {
            id: "name",
            header: t("Name"),
            cellClassName: "font-medium",
            cell: (channel) => channel.name,
          },
          {
            id: "type",
            header: t("Type"),
            cell: (channel) => (
              <BadgeCell>
                <StatusBadge
                  label={getTypeLabel(t, channel.type)}
                  variant="neutral"
                  copyable={false}
                />
              </BadgeCell>
            ),
          },
          {
            id: "status",
            header: t("Status"),
            cell: (channel) => (
              <BadgeCell>
                <StatusBadge
                  label={channel.status === "1" ? t("Enabled") : t("Disabled")}
                  variant={channel.status === "1" ? "success" : "neutral"}
                  copyable={false}
                />
              </BadgeCell>
            ),
          },
          {
            id: "default",
            header: t("Default"),
            cell: (channel) => (
              <BadgeCell>
                <StatusBadge
                  label={channel.is_default === "1" ? t("Yes") : t("No")}
                  variant={channel.is_default === "1" ? "success" : "neutral"}
                  copyable={false}
                />
              </BadgeCell>
            ),
          },
          {
            id: "chunk-threshold",
            header: t("Chunk threshold"),
            cell: (channel) => formatBytes(channel.chunk_threshold),
          },
          {
            id: "chunk-size",
            header: t("Chunk size"),
            cell: (channel) => formatBytes(channel.chunk_size),
          },
          {
            id: "updated",
            header: t("Updated"),
            cell: (channel) =>
              new Date(channel.update_time * 1000).toLocaleString(),
          },
          {
            id: "actions",
            header: t("Actions"),
            className: "text-right",
            cellClassName: "text-right",
            cell: (channel) => (
              <div className="flex justify-end gap-1">
                <Button
                  size="icon-sm"
                  variant="ghost"
                  onClick={() => openEdit(channel)}
                  title={t("Edit")}
                  aria-label={t("Edit")}
                >
                  <Pencil className="size-4" />
                </Button>
                <Button
                  size="icon-sm"
                  variant="ghost"
                  onClick={() => setCopyTarget(channel)}
                  title={t("Copy Channel")}
                  aria-label={t("Copy Channel")}
                >
                  <Copy className="size-4" />
                </Button>
                <Button
                  size="icon-sm"
                  variant="ghost"
                  onClick={() => setDefaultMutation.mutate(channel)}
                  disabled={
                    channel.status !== "1" || channel.is_default === "1"
                  }
                  title={t("Set default")}
                  aria-label={t("Set default")}
                >
                  <Star className="size-4" />
                </Button>
                <Button
                  size="icon-sm"
                  variant="ghost"
                  onClick={() => setProbeIntent({ kind: "saved", channel })}
                  title={t("Probe")}
                  aria-label={t("Probe")}
                >
                  <Radar className="size-4" />
                </Button>
                <Button
                  size="icon-sm"
                  variant="ghost"
                  onClick={() => toggleStatusMutation.mutate(channel)}
                  title={channel.status === "1" ? t("Disable") : t("Enable")}
                  aria-label={
                    channel.status === "1" ? t("Disable") : t("Enable")
                  }
                >
                  <Power className="size-4" />
                </Button>
                <Button
                  size="icon-sm"
                  variant="ghost"
                  onClick={() => setDeleteTarget(channel)}
                  title={t("Delete")}
                  aria-label={t("Delete")}
                >
                  <Trash2 className="size-4 text-destructive" />
                </Button>
              </div>
            ),
          },
        ]}
      />

      <ChannelFormDialog
        open={formOpen}
        channel={editingChannel}
        onOpenChange={(open) => {
          setFormOpen(open);
          if (!open) setEditingChannel(null);
        }}
        onSave={saveChannel}
        onProbe={async (payload) => {
          setProbeIntent({ kind: "draft", payload });
        }}
        isProbing={probeDraftMutation.isPending}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t("Delete Channel")}
        desc={t('Delete "{{name}}"? This cannot be undone.', {
          name: deleteTarget?.name ?? "",
        })}
        confirmText={t("Delete")}
        destructive
        handleConfirm={async () => {
          if (!deleteTarget) return;
          await deleteMutation.mutateAsync(deleteTarget);
          setDeleteTarget(null);
        }}
        isLoading={deleteMutation.isPending}
      />

      <Dialog
        open={!!copyTarget}
        onOpenChange={(open) => {
          if (!open && !copyMutation.isPending) {
            setCopyTarget(null);
            setCopySuffix("_copy");
          }
        }}
        title={t("Copy Channel")}
        description={
          copyTarget ? (
            <>
              {t("Create a copy of:")}
              <strong>{copyTarget.name}</strong>
            </>
          ) : undefined
        }
        contentHeight="auto"
        footer={
          <>
            <Button
              variant="outline"
              onClick={() => setCopyTarget(null)}
              disabled={copyMutation.isPending}
            >
              {t("Cancel")}
            </Button>
            <Button
              onClick={() => {
                if (!copyTarget) return;
                copyMutation.mutate({
                  channel: copyTarget,
                  suffix: copySuffix,
                });
              }}
              disabled={copyMutation.isPending}
            >
              {copyMutation.isPending && (
                <Loader2 className="mr-2 size-4 animate-spin" />
              )}
              {copyMutation.isPending ? t("Copying...") : t("Copy Channel")}
            </Button>
          </>
        }
      >
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="file-upload-channel-copy-suffix">
              {t("Name Suffix")}
            </Label>
            <Input
              id="file-upload-channel-copy-suffix"
              placeholder={t("_copy")}
              value={copySuffix}
              onChange={(event) => setCopySuffix(event.target.value)}
              disabled={copyMutation.isPending}
            />
            <p className="text-muted-foreground text-xs">
              {t("New name will be:")} {copyTarget?.name}
              {copySuffix}
            </p>
          </div>
        </div>
      </Dialog>

      <ConfirmDialog
        open={!!probeIntent}
        onOpenChange={(open) => !open && setProbeIntent(null)}
        title={t("Confirm probe")}
        desc={t(
          "Probe will upload temporary file {{name}}, file size: {{size}}. Please confirm or cancel.",
          {
            name: probeFileInfo.file_name,
            size: formatBytes(probeFileInfo.file_size),
          },
        )}
        confirmText={t("Confirm")}
        cancelBtnText={t("Cancel")}
        handleConfirm={confirmProbe}
        isLoading={probeMutation.isPending || probeDraftMutation.isPending}
      />

      <Dialog
        open={probeResultOpen}
        onOpenChange={setProbeResultOpen}
        title={t("Probe result")}
        description={t(
          "Probe uploaded the built-in temporary file and returned the final address.",
        )}
        contentClassName="sm:max-w-xl"
        footer={
          <Button variant="outline" onClick={() => setProbeResultOpen(false)}>
            {t("Close")}
          </Button>
        }
      >
        {displayProbeResult ? (
          <Card size="sm" className="shadow-none">
            <CardHeader>
              <CardTitle>{t("Probe succeeded")}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="grid gap-2 text-sm">
                <div className="flex items-center justify-between gap-3">
                  <span className="text-muted-foreground">{t("File")}</span>
                  <span className="font-medium">
                    {displayProbeResult.file_name}
                  </span>
                </div>
                <div className="flex items-center justify-between gap-3">
                  <span className="text-muted-foreground">
                    {t("File size")}
                  </span>
                  <span className="font-medium">
                    {formatBytes(displayProbeResult.file_size)}
                  </span>
                </div>
                <div className="flex items-center justify-between gap-3">
                  <span className="text-muted-foreground">{t("Latency")}</span>
                  <span className="font-medium">
                    {displayProbeResult.latency_ms} ms
                  </span>
                </div>
              </div>
              <div className="space-y-2">
                <div className="text-muted-foreground text-sm">
                  {t("Response URL")}
                </div>
                <div className="bg-muted flex items-start gap-2 rounded-lg border p-3">
                  <a
                    className="min-w-0 flex-1 break-all text-sm underline underline-offset-2"
                    href={displayProbeResult.url}
                    target="_blank"
                    rel="noreferrer"
                  >
                    {displayProbeResult.url}
                  </a>
                  <CopyButton
                    value={displayProbeResult.url}
                    tooltip={t("Copy URL")}
                    successTooltip={t("Copied!")}
                    aria-label={t("Copy URL")}
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        ) : null}
      </Dialog>
    </SettingsSection>
  );
}

type ChannelFormDialogProps = {
  open: boolean;
  channel: FileUploadChannel | null;
  onOpenChange: (open: boolean) => void;
  onSave: (
    values: FileUploadChannelFormValues,
    editing: FileUploadChannel | null,
  ) => Promise<void>;
  onProbe: (payload: Record<string, unknown>) => Promise<void>;
  isProbing: boolean;
};

function ChannelFormDialog(props: ChannelFormDialogProps) {
  const { t } = useTranslation();
  const isEditing = !!props.channel;
  const maskedPasswordHint =
    props.channel?.type === "1"
      ? getSensitiveConfigHint(props.channel.config_proflle, "password")
      : "";
  const maskedApiTokenHint =
    props.channel?.type === "2"
      ? getSensitiveConfigHint(props.channel.config_proflle, "api_token")
      : "";
  const maskedSecretAccessKeyHint =
    props.channel?.type === "3"
      ? getSensitiveConfigHint(
          props.channel.config_proflle,
          "secret_access_key",
        )
      : "";
  const form = useForm<FileUploadChannelFormValues>({
    resolver: zodResolver(channelFormSchema),
    defaultValues: buildEmptyFormValues(),
  });

  const watchedType = useWatch({ control: form.control, name: "type" });
  const watchedAuthType = useWatch({
    control: form.control,
    name: "auth_type",
  });
  const watchedReturnFormat = useWatch({
    control: form.control,
    name: "return_format",
  });
  const lastTypeRef = useRef<FileUploadChannelType | null>(null);
  const pendingHydrateTypeRef = useRef<FileUploadChannelType | null>(null);

  useEffect(() => {
    if (!props.open) return;
    const nextValues = buildFormValues(props.channel);
    pendingHydrateTypeRef.current = nextValues.type;
    form.reset(nextValues);
    lastTypeRef.current = nextValues.type;
  }, [form, props.channel, props.open]);

  useEffect(() => {
    if (!props.open) return;
    if (pendingHydrateTypeRef.current !== null) {
      if (watchedType !== pendingHydrateTypeRef.current) return;
      pendingHydrateTypeRef.current = null;
      return;
    }
    if (!lastTypeRef.current) {
      lastTypeRef.current = watchedType;
      return;
    }
    if (lastTypeRef.current === watchedType) return;
    lastTypeRef.current = watchedType;
    form.setValue("base_url", "");
    form.setValue("username", "");
    form.setValue("password", "");
    form.setValue("auth_type", "basic");
    form.setValue("path_prefix", "");
    form.setValue("public_base_url", "");
    form.setValue("timeout_ms", "600000");
    form.setValue("api_token", "");
    form.setValue("upload_channel", "cfr2");
    form.setValue("upload_folder", "");
    form.setValue("return_format", "default");
    form.setValue("endpoint", "");
    form.setValue("region", "");
    form.setValue("bucket", "");
    form.setValue("access_key_id", "");
    form.setValue("secret_access_key", "");
    form.setValue("force_path_style", false);
    form.setValue("key_prefix", "");
  }, [form, watchedType, props.open]);

  const onSubmit = async (values: FileUploadChannelFormValues) => {
    await props.onSave(values, props.channel);
  };

  const renderTypeSpecificFields = () => {
    if (watchedType === "1") {
      return (
        <div className="grid gap-4 md:grid-cols-2">
          <Field label={t("Base URL")}>
            <Input {...form.register("base_url")} />
          </Field>
          <Field label={t("Username")}>
            <Input {...form.register("username")} />
          </Field>
          <Field label={t("Password")}>
            <Input
              type="password"
              placeholder={maskedPasswordHint || undefined}
              {...form.register("password")}
            />
          </Field>
          <Field label={t("Auth type")}>
            <Select
              value={watchedAuthType || "basic"}
              onValueChange={(value) =>
                form.setValue(
                  "auth_type",
                  typeof value === "string" ? value : "basic",
                )
              }
            >
              <SelectTrigger className="w-full min-w-0">
                <SelectValue>
                  {getSelectedLabel(getAuthTypeOptions(t), watchedAuthType)}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {getAuthTypeOptions(t).map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
          <Field label={t("Timeout ms")}>
            <Input
              type="number"
              min={1}
              step={1}
              {...form.register("timeout_ms")}
            />
          </Field>
          <Field label={t("Path prefix")}>
            <Input {...form.register("path_prefix")} />
          </Field>
          <Field label={t("Public base URL")}>
            <Input {...form.register("public_base_url")} />
          </Field>
        </div>
      );
    }

    if (watchedType === "2") {
      return (
        <div className="grid gap-4 md:grid-cols-2">
          <Field label={t("Base URL")}>
            <Input {...form.register("base_url")} />
          </Field>
          <Field label={t("API token")}>
            <Input
              type="password"
              placeholder={maskedApiTokenHint || undefined}
              {...form.register("api_token")}
            />
          </Field>
          <Field label={t("Upstream upload channel")}>
            <Select
              value={form.watch("upload_channel") || "cfr2"}
              onValueChange={(value) =>
                form.setValue(
                  "upload_channel",
                  typeof value === "string" ? value : "cfr2",
                )
              }
            >
              <SelectTrigger className="w-full min-w-0">
                <SelectValue>
                  {getSelectedLabel(
                    getUploadChannelOptions(),
                    form.watch("upload_channel"),
                  )}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {getUploadChannelOptions().map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
          <Field label={t("Upload folder")}>
            <Input {...form.register("upload_folder")} />
          </Field>
          <Field label={t("Public base URL")}>
            <Input {...form.register("public_base_url")} />
          </Field>
          <Field label={t("Return format")}>
            <Select
              value={watchedReturnFormat || "default"}
              onValueChange={(value) =>
                form.setValue(
                  "return_format",
                  typeof value === "string" ? value : "default",
                )
              }
            >
              <SelectTrigger className="w-full min-w-0">
                <SelectValue>
                  {getSelectedLabel(
                    getReturnFormatOptions(t),
                    watchedReturnFormat,
                  )}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {getReturnFormatOptions(t).map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
        </div>
      );
    }

    return (
      <div className="grid gap-4 md:grid-cols-2">
        <Field label={t("Endpoint")}>
          <Input {...form.register("endpoint")} />
        </Field>
        <Field label={t("Region")}>
          <Input {...form.register("region")} />
        </Field>
        <Field label={t("Bucket")}>
          <Input {...form.register("bucket")} />
        </Field>
        <Field label={t("Access key ID")}>
          <Input {...form.register("access_key_id")} />
        </Field>
        <Field label={t("Secret access key")}>
          <Input
            type="password"
            placeholder={maskedSecretAccessKeyHint || undefined}
            {...form.register("secret_access_key")}
          />
        </Field>
        <Field label={t("Key prefix")}>
          <Input {...form.register("key_prefix")} />
        </Field>
        <Field label={t("Public base URL")}>
          <Input {...form.register("public_base_url")} />
        </Field>
        <SettingsSwitchItem className="mt-6">
          <SettingsSwitchContent>
            <label className="text-sm font-medium">
              {t("Force path style")}
            </label>
          </SettingsSwitchContent>
          <Switch
            checked={form.watch("force_path_style")}
            onCheckedChange={(checked) =>
              form.setValue("force_path_style", checked)
            }
          />
        </SettingsSwitchItem>
      </div>
    );
  };

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={isEditing ? t("Edit Channel") : t("Add Channel")}
      description={t("Secrets stay on the server and are never shown in full.")}
      contentClassName="sm:max-w-3xl"
      bodyClassName="space-y-5"
      footer={
        <>
          <Button variant="outline" onClick={() => props.onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            variant="outline"
            onClick={form.handleSubmit(async (values) => {
              await props.onProbe({
                id: props.channel?.id ?? 0,
                type: values.type,
                config_proflle: buildConfigPayload(values),
              });
            })}
            disabled={props.isProbing}
          >
            {t("Probe")}
          </Button>
          <Button onClick={form.handleSubmit(onSubmit)}>
            {isEditing ? t("Update") : t("Create")}
          </Button>
        </>
      }
    >
      <form className="space-y-5" onSubmit={form.handleSubmit(onSubmit)}>
        <div className="grid gap-4 md:grid-cols-2">
          <Field label={t("Name")}>
            <Input {...form.register("name")} />
          </Field>
          <Field label={t("Type")}>
            <Select
              value={watchedType}
              onValueChange={(value) =>
                form.setValue("type", value as FileUploadChannelType)
              }
              disabled={isEditing}
            >
              <SelectTrigger className="w-full min-w-0">
                <SelectValue>
                  {getSelectedLabel(getChannelTypeOptions(t), watchedType)}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {getChannelTypeOptions(t).map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
          <Field
            label={t("Chunk threshold")}
            description={t(
              "Unit: B. Files above this threshold use chunked upload.",
            )}
          >
            <Input
              type="number"
              min={0}
              step={1}
              {...form.register("chunk_threshold")}
            />
          </Field>
          <Field
            label={t("Chunk size")}
            description={t("Unit: B. Size of each uploaded chunk.")}
          >
            <Input
              type="number"
              min={0}
              step={1}
              {...form.register("chunk_size")}
            />
          </Field>
          <Field
            label={t("Max size")}
            description={t(
              "Unit: B. Maximum allowed file size. 0 means unlimited.",
            )}
          >
            <Input
              type="number"
              min={0}
              step={1}
              {...form.register("max_size")}
            />
          </Field>
          <SettingsSwitchItem className="mt-6">
            <SettingsSwitchContent>
              <label className="text-sm font-medium">{t("Enabled")}</label>
            </SettingsSwitchContent>
            <Switch
              checked={form.watch("status") === "1"}
              disabled={form.watch("is_default") === "1"}
              onCheckedChange={(checked) =>
                form.setValue("status", checked ? "1" : "0")
              }
            />
          </SettingsSwitchItem>
          <SettingsSwitchItem>
            <SettingsSwitchContent>
              <label className="text-sm font-medium">{t("Default")}</label>
            </SettingsSwitchContent>
            <Switch
              checked={form.watch("is_default") === "1"}
              disabled={form.watch("status") !== "1"}
              onCheckedChange={(checked) =>
                form.setValue("is_default", checked ? "1" : "0")
              }
            />
          </SettingsSwitchItem>
        </div>

        {renderTypeSpecificFields()}
      </form>
    </Dialog>
  );
}

function Field(props: {
  label: string;
  children: ReactNode;
  description?: ReactNode;
}) {
  return (
    <div className="grid gap-2">
      <label className="text-sm font-medium">{props.label}</label>
      {props.children}
      {props.description ? (
        <p className="text-muted-foreground text-xs leading-5">
          {props.description}
        </p>
      ) : null}
    </div>
  );
}
