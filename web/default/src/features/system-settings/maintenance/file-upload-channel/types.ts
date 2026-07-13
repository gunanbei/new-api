export type FileUploadChannelType = "1" | "2" | "3";

export type FileUploadChannelStatus = "0" | "1";

export type FileUploadChannelDefaultFlag = "0" | "1";

export type FileUploadChannelConfig = Record<string, unknown>;

export type FileUploadChannel = {
  id: number;
  name: string;
  type: FileUploadChannelType;
  status: FileUploadChannelStatus;
  is_default: FileUploadChannelDefaultFlag;
  chunk_threshold: number;
  chunk_size: number;
  max_size: number;
  config_proflle?: FileUploadChannelConfig;
  create_user_id: number;
  update_user_id: number;
  create_time: number;
  update_time: number;
};

export type FileUploadChannelSummary = Pick<
  FileUploadChannel,
  | "id"
  | "name"
  | "type"
  | "status"
  | "is_default"
  | "chunk_threshold"
  | "chunk_size"
  | "max_size"
>;

export type FileUploadChannelProbeResult = {
  ok: boolean;
  type: string;
  latency_ms: number;
  detail: string;
  file_name: string;
  file_size: number;
  url: string;
};

export type FileUploadChannelProbeFileInfo = {
  file_name: string;
  file_size: number;
};

export type FileUploadChannelListResponse = {
  success: boolean;
  message: string;
  data?: FileUploadChannel[];
};

export type FileUploadChannelResponse = {
  success: boolean;
  message: string;
  data?: FileUploadChannel | null;
};

export type FileUploadChannelMutationResponse = {
  success: boolean;
  message: string;
  data?: FileUploadChannel;
};

export type FileUploadChannelProbeResponse = {
  success: boolean;
  message: string;
  data?: FileUploadChannelProbeResult;
};

export type FileUploadChannelProbeFileInfoResponse = {
  success: boolean;
  message: string;
  data?: FileUploadChannelProbeFileInfo;
};
