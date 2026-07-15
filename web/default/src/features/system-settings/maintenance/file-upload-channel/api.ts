import { api } from "@/lib/api";

import type {
  FileUploadChannelListResponse,
  FileUploadChannelMutationResponse,
  FileUploadChannelProbeFileInfoResponse,
  FileUploadChannelProbeResponse,
  FileUploadChannelResponse,
} from "./types";

export async function getFileUploadChannels(params?: {
  type?: string;
  status?: string;
}) {
  const res = await api.get<FileUploadChannelListResponse>(
    "/api/file-upload-channel/",
    { params },
  );
  return res.data;
}

export async function getFileUploadChannel(id: number) {
  const res = await api.get<FileUploadChannelResponse>(
    `/api/file-upload-channel/${id}`,
  );
  return res.data;
}

export async function createFileUploadChannel(data: Record<string, unknown>) {
  const res = await api.post<FileUploadChannelMutationResponse>(
    "/api/file-upload-channel/",
    data,
  );
  return res.data;
}

export async function copyFileUploadChannel(id: number, suffix: string) {
  const res = await api.post<FileUploadChannelMutationResponse>(
    `/api/file-upload-channel/${id}/copy`,
    { suffix },
  );
  return res.data;
}

export async function updateFileUploadChannel(
  id: number,
  data: Record<string, unknown>,
) {
  const res = await api.put<FileUploadChannelMutationResponse>(
    `/api/file-upload-channel/${id}`,
    data,
  );
  return res.data;
}

export async function setDefaultFileUploadChannel(id: number) {
  const res = await api.put<FileUploadChannelMutationResponse>(
    `/api/file-upload-channel/${id}/default`,
  );
  return res.data;
}

export async function updateFileUploadChannelStatus(
  id: number,
  status: "0" | "1",
) {
  const res = await api.put<FileUploadChannelMutationResponse>(
    `/api/file-upload-channel/${id}/status`,
    { status },
  );
  return res.data;
}

export async function deleteFileUploadChannel(id: number) {
  const res = await api.delete<{ success: boolean; message: string }>(
    `/api/file-upload-channel/${id}`,
  );
  return res.data;
}

export async function probeSavedFileUploadChannel(id: number) {
  const res = await api.post<FileUploadChannelProbeResponse>(
    `/api/file-upload-channel/${id}/probe`,
  );
  return res.data;
}

export async function getFileUploadChannelProbeFileInfo() {
  const res = await api.get<FileUploadChannelProbeFileInfoResponse>(
    "/api/file-upload-channel/probe-file-info",
  );
  return res.data;
}

export async function probeDraftFileUploadChannel(
  data: Record<string, unknown>,
) {
  const res = await api.post<FileUploadChannelProbeResponse>(
    "/api/file-upload-channel/probe",
    data,
  );
  return res.data;
}

export async function getDefaultFileUploadChannel() {
  const res = await api.get<FileUploadChannelResponse>(
    "/api/file-upload-channel/default",
  );
  return res.data;
}

export async function resolveFileUploadChannel(channelType: string) {
  const res = await api.get<FileUploadChannelResponse>(
    "/api/file-upload-channel/resolve",
    { params: { channel_type: channelType } },
  );
  return res.data;
}
