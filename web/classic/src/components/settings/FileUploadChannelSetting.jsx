/*
Copyright (C) 2025 QuantumNous

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

import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Empty,
  Input,
  Modal,
  Select,
  Space,
  Spin,
  Switch,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { Edit, Plus, Power, Radar, Star, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { API, showError, showSuccess } from '../../helpers';
import { copy } from '../../helpers/utils';

const { Text } = Typography;

const TYPE_OPTIONS = [
  { value: '1', label: 'WebDAV' },
  { value: '2', label: 'Cloudflare ImageBed' },
  { value: '3', label: 'S3 Compatible Storage' },
];

const AUTH_TYPE_OPTIONS = [
  { value: 'basic', label: 'Basic' },
  { value: 'bearer', label: 'Bearer' },
];

const RETURN_FORMAT_OPTIONS = [
  { value: 'default', label: 'Default' },
  { value: 'full', label: 'Full' },
];

const UPLOAD_CHANNEL_OPTIONS = [
  { value: 'telegram', label: 'TG' },
  { value: 'cfr2', label: 'Cloudflare R2' },
  { value: 's3', label: 's3' },
  { value: 'webdav', label: 'webdav' },
];

function createEmptyForm(type = '1') {
  return {
    name: '',
    type,
    status: '1',
    is_default: '0',
    chunk_threshold: '16777216',
    chunk_size: '8388608',
    max_size: '0',
    base_url: '',
    username: '',
    password: '',
    auth_type: 'basic',
    path_prefix: '',
    public_base_url: '',
    timeout_ms: '600000',
    api_token: '',
    upload_channel: 'cfr2',
    upload_folder: '',
    return_format: 'default',
    endpoint: '',
    region: '',
    bucket: '',
    access_key_id: '',
    secret_access_key: '',
    force_path_style: false,
    key_prefix: '',
  };
}

function formatBytes(bytes) {
  if (!Number.isFinite(bytes)) return '-';
  if (bytes === 0) return '0 B';
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
  let value = Math.abs(bytes);
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  const display =
    value >= 10 || unitIndex === 0 ? value.toFixed(0) : value.toFixed(1);
  return `${bytes < 0 ? '-' : ''}${display} ${units[unitIndex]}`;
}

function getConfigValue(config, key) {
  const value = config?.[key];
  if (typeof value === 'string' && !value.includes('*')) return value;
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }
  return '';
}

function getSensitiveConfigHint(config, key) {
  const value = config?.[key];
  return typeof value === 'string' && value.includes('*') ? value : '';
}

function getConfigBoolean(config, key) {
  const value = config?.[key];
  if (typeof value === 'boolean') return value;
  if (typeof value === 'string') {
    return value === 'true' || value === '1';
  }
  return false;
}

function buildFormFromChannel(channel) {
  if (!channel) return createEmptyForm();

  const config = channel.config_proflle || {};
  return {
    name: channel.name || '',
    type: channel.type || '1',
    status: channel.status || '1',
    is_default: channel.is_default || '0',
    chunk_threshold: String(channel.chunk_threshold ?? 16777216),
    chunk_size: String(channel.chunk_size ?? 8388608),
    max_size: String(channel.max_size ?? 0),
    base_url:
      channel.type === '1' || channel.type === '2'
        ? getConfigValue(config, 'base_url')
        : '',
    username: channel.type === '1' ? getConfigValue(config, 'username') : '',
    password: '',
    auth_type:
      channel.type === '1'
        ? getConfigValue(config, 'auth_type') || 'basic'
        : 'basic',
    path_prefix:
      channel.type === '1' ? getConfigValue(config, 'path_prefix') : '',
    public_base_url:
      channel.type === '1' || channel.type === '3'
        ? getConfigValue(config, 'public_base_url')
        : '',
    timeout_ms:
      channel.type === '1'
        ? getConfigValue(config, 'timeout_ms') || '600000'
        : '600000',
    api_token: '',
    upload_channel:
      channel.type === '2'
        ? getConfigValue(config, 'upload_channel') || 'cfr2'
        : 'cfr2',
    upload_folder:
      channel.type === '2' ? getConfigValue(config, 'upload_folder') : '',
    return_format:
      channel.type === '2'
        ? getConfigValue(config, 'return_format') || 'default'
        : 'default',
    endpoint: channel.type === '3' ? getConfigValue(config, 'endpoint') : '',
    region: channel.type === '3' ? getConfigValue(config, 'region') : '',
    bucket: channel.type === '3' ? getConfigValue(config, 'bucket') : '',
    access_key_id:
      channel.type === '3' ? getConfigValue(config, 'access_key_id') : '',
    secret_access_key: '',
    force_path_style:
      channel.type === '3'
        ? getConfigBoolean(config, 'force_path_style')
        : false,
    key_prefix:
      channel.type === '3' ? getConfigValue(config, 'key_prefix') : '',
  };
}

function resetTypeSpecificFields(nextType, previous) {
  return {
    ...createEmptyForm(nextType),
    name: previous.name,
    status: previous.status,
    is_default: previous.is_default,
    chunk_threshold: previous.chunk_threshold,
    chunk_size: previous.chunk_size,
    max_size: previous.max_size,
  };
}

function buildConfigPayload(formData) {
  if (formData.type === '1') {
    return {
      base_url: formData.base_url.trim(),
      username: formData.username.trim(),
      password: formData.password.trim(),
      auth_type: formData.auth_type || 'basic',
      path_prefix: formData.path_prefix.trim(),
      public_base_url: formData.public_base_url.trim(),
      timeout_ms: Number(formData.timeout_ms || 600000),
    };
  }

  if (formData.type === '2') {
    return {
      base_url: formData.base_url.trim(),
      api_token: formData.api_token.trim(),
      upload_channel: formData.upload_channel.trim() || 'cfr2',
      upload_folder: formData.upload_folder.trim(),
      return_format: formData.return_format || 'default',
    };
  }

  return {
    endpoint: formData.endpoint.trim(),
    region: formData.region.trim(),
    bucket: formData.bucket.trim(),
    access_key_id: formData.access_key_id.trim(),
    secret_access_key: formData.secret_access_key.trim(),
    force_path_style: formData.force_path_style,
    public_base_url: formData.public_base_url.trim(),
    key_prefix: formData.key_prefix.trim(),
  };
}

function buildSubmitPayload(formData) {
  return {
    name: formData.name.trim(),
    type: formData.type,
    status: formData.status,
    is_default: formData.is_default,
    chunk_threshold: Number(formData.chunk_threshold || 0),
    chunk_size: Number(formData.chunk_size || 0),
    max_size: Number(formData.max_size || 0),
    config_proflle: buildConfigPayload(formData),
  };
}

function normalizeProbeFileInfo(info) {
  return {
    file_name: info?.file_name || 'test.txt',
    file_size:
      typeof info?.file_size === 'number' && Number.isFinite(info.file_size)
        ? info.file_size
        : 0,
  };
}

function normalizeProbeResult(result) {
  if (!result) return null;
  return {
    ...result,
    file_name: result.file_name || '',
    file_size:
      typeof result.file_size === 'number' && Number.isFinite(result.file_size)
        ? result.file_size
        : 0,
    url: result.url || '',
  };
}

function mergeProbeResultWithFileInfo(result, info) {
  if (!result) return null;
  return {
    ...result,
    file_name: result.file_name || info.file_name,
    file_size:
      typeof result.file_size === 'number' && Number.isFinite(result.file_size)
        ? result.file_size
        : info.file_size,
  };
}

function Field(props) {
  return (
    <div style={{ display: 'grid', gap: 6 }}>
      <Text strong>{props.label}</Text>
      {props.children}
      {props.description ? (
        <Text
          type='secondary'
          size='small'
          style={{ display: 'block', lineHeight: 1.5 }}
        >
          {props.description}
        </Text>
      ) : null}
    </div>
  );
}

export default function FileUploadChannelSetting() {
  const { t } = useTranslation();
  const [channels, setChannels] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [modalLoading, setModalLoading] = useState(false);
  const [probeLoading, setProbeLoading] = useState(false);
  const [editingChannel, setEditingChannel] = useState(null);
  const [deleteTarget, setDeleteTarget] = useState(null);
  const [formData, setFormData] = useState(createEmptyForm());
  const [probeFileInfo, setProbeFileInfo] = useState({
    file_name: 'test.txt',
    file_size: 0,
  });
  const [probeConfirmVisible, setProbeConfirmVisible] = useState(false);
  const [pendingProbe, setPendingProbe] = useState(null);
  const [probeResult, setProbeResult] = useState(null);
  const [probeResultVisible, setProbeResultVisible] = useState(false);
  const maskedPasswordHint =
    editingChannel?.type === '1'
      ? getSensitiveConfigHint(editingChannel?.config_proflle, 'password')
      : '';
  const maskedApiTokenHint =
    editingChannel?.type === '2'
      ? getSensitiveConfigHint(editingChannel?.config_proflle, 'api_token')
      : '';
  const maskedSecretAccessKeyHint =
    editingChannel?.type === '3'
      ? getSensitiveConfigHint(
          editingChannel?.config_proflle,
          'secret_access_key',
        )
      : '';
  const displayProbeResult = mergeProbeResultWithFileInfo(
    probeResult,
    probeFileInfo,
  );

  const defaultChannel = useMemo(
    () => channels.find((item) => item.is_default === '1') || null,
    [channels],
  );

  const loadChannels = async () => {
    try {
      setLoading(true);
      const res = await API.get('/api/file-upload-channel/');
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setChannels(data || []);
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  const upsertChannel = (nextChannel) => {
    if (!nextChannel) return;
    setChannels((prev) => {
      const current = Array.isArray(prev) ? prev : [];
      const exists = current.some((item) => item.id === nextChannel.id);
      if (!exists) return [nextChannel, ...current];
      return current.map((item) =>
        item.id === nextChannel.id ? nextChannel : item,
      );
    });
  };

  useEffect(() => {
    loadChannels();
  }, []);

  useEffect(() => {
    const loadProbeFileInfo = async () => {
      try {
        const res = await API.get('/api/file-upload-channel/probe-file-info');
        const { success, data } = res.data;
        if (success && data) {
          setProbeFileInfo(normalizeProbeFileInfo(data));
        }
      } catch (error) {
        console.error(error);
      }
    };
    loadProbeFileInfo();
  }, []);

  const updateFormField = (key, value) => {
    setFormData((prev) => ({ ...prev, [key]: value }));
  };

  const handleOpenCreate = () => {
    setEditingChannel(null);
    setFormData(createEmptyForm());
    setModalVisible(true);
  };

  const handleOpenEdit = (channel) => {
    const latestChannel =
      channels.find((item) => item.id === channel.id) || channel;
    setEditingChannel(latestChannel);
    setFormData(buildFormFromChannel(latestChannel));
    setModalVisible(true);
  };

  const handleSave = async () => {
    try {
      setModalLoading(true);
      const payload = buildSubmitPayload(formData);
      const res = editingChannel
        ? await API.put(
            `/api/file-upload-channel/${editingChannel.id}`,
            payload,
          )
        : await API.post('/api/file-upload-channel/', payload);
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(t(editingChannel ? 'Updated' : 'Created'));
      if (data) {
        upsertChannel(data);
      }
      setModalVisible(false);
      setEditingChannel(null);
      await loadChannels();
    } catch (error) {
      console.error(error);
    } finally {
      setModalLoading(false);
    }
  };

  const handleProbeDraft = async () => {
    setPendingProbe({
      kind: 'draft',
      payload: {
        id: editingChannel?.id || 0,
        type: formData.type,
        config_proflle: buildConfigPayload(formData),
      },
    });
    setProbeConfirmVisible(true);
  };

  const runProbeDraft = async (payload) => {
    try {
      setProbeLoading(true);
      const res = await API.post('/api/file-upload-channel/probe', payload);
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(t('Probe succeeded'));
      setProbeResult(normalizeProbeResult(data));
      setProbeResultVisible(true);
    } catch (error) {
      console.error(error);
    } finally {
      setProbeLoading(false);
    }
  };

  const handleProbeSaved = async (channel) => {
    setPendingProbe({ kind: 'saved', channel });
    setProbeConfirmVisible(true);
  };

  const runProbeSaved = async (channel) => {
    try {
      setProbeLoading(true);
      const res = await API.post(
        `/api/file-upload-channel/${channel.id}/probe`,
      );
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(t('Probe succeeded'));
      setProbeResult(normalizeProbeResult(data));
      setProbeResultVisible(true);
    } catch (error) {
      console.error(error);
    } finally {
      setProbeLoading(false);
    }
  };

  const handleConfirmProbe = async () => {
    const currentProbe = pendingProbe;
    setProbeConfirmVisible(false);
    setPendingProbe(null);
    if (!currentProbe) return;
    if (currentProbe.kind === 'saved') {
      await runProbeSaved(currentProbe.channel);
      return;
    }
    await runProbeDraft(currentProbe.payload);
  };

  const handleCopyProbeUrl = async () => {
    if (!probeResult?.url) return;
    const ok = await copy(probeResult.url);
    if (ok) {
      showSuccess(t('复制成功'));
    } else {
      showError(t('复制失败'));
    }
  };

  const handleSetDefault = async (channel) => {
    try {
      const res = await API.put(
        `/api/file-upload-channel/${channel.id}/default`,
      );
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(t('Default channel updated'));
      await loadChannels();
    } catch (error) {
      console.error(error);
    }
  };

  const handleToggleStatus = async (channel) => {
    try {
      const nextStatus = channel.status === '1' ? '0' : '1';
      const res = await API.put(
        `/api/file-upload-channel/${channel.id}/status`,
        {
          status: nextStatus,
        },
      );
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(t('Status updated'));
      await loadChannels();
    } catch (error) {
      console.error(error);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      const res = await API.delete(
        `/api/file-upload-channel/${deleteTarget.id}`,
      );
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(t('Deleted'));
      setDeleteTarget(null);
      await loadChannels();
    } catch (error) {
      console.error(error);
    }
  };

  const renderTypeFields = () => {
    if (formData.type === '1') {
      return (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
            gap: 12,
          }}
        >
          <Field label={t('Base URL')}>
            <Input
              value={formData.base_url}
              onChange={(value) => updateFormField('base_url', value)}
            />
          </Field>
          <Field label={t('Username')}>
            <Input
              value={formData.username}
              onChange={(value) => updateFormField('username', value)}
            />
          </Field>
          <Field label={t('Password')}>
            <Input
              type='password'
              value={formData.password}
              placeholder={maskedPasswordHint || undefined}
              onChange={(value) => updateFormField('password', value)}
            />
          </Field>
          <Field label={t('Auth type')}>
            <Select
              value={formData.auth_type}
              optionList={AUTH_TYPE_OPTIONS.map((item) => ({
                ...item,
                label: t(item.label),
              }))}
              onChange={(value) => updateFormField('auth_type', value)}
            />
          </Field>
          <Field label={t('Timeout ms')}>
            <Input
              type='number'
              value={formData.timeout_ms}
              onChange={(value) => updateFormField('timeout_ms', value)}
            />
          </Field>
          <Field label={t('Path prefix')}>
            <Input
              value={formData.path_prefix}
              onChange={(value) => updateFormField('path_prefix', value)}
            />
          </Field>
          <Field label={t('Public base URL')}>
            <Input
              value={formData.public_base_url}
              onChange={(value) => updateFormField('public_base_url', value)}
            />
          </Field>
        </div>
      );
    }

    if (formData.type === '2') {
      return (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
            gap: 12,
          }}
        >
          <Field label={t('Base URL')}>
            <Input
              value={formData.base_url}
              onChange={(value) => updateFormField('base_url', value)}
            />
          </Field>
          <Field label={t('API token')}>
            <Input
              type='password'
              value={formData.api_token}
              placeholder={maskedApiTokenHint || undefined}
              onChange={(value) => updateFormField('api_token', value)}
            />
          </Field>
          <Field label={t('Upstream upload channel')}>
            <Select
              value={formData.upload_channel}
              optionList={UPLOAD_CHANNEL_OPTIONS}
              onChange={(value) => updateFormField('upload_channel', value)}
            />
          </Field>
          <Field label={t('Upload folder')}>
            <Input
              value={formData.upload_folder}
              onChange={(value) => updateFormField('upload_folder', value)}
            />
          </Field>
          <Field label={t('Public base URL')}>
            <Input
              value={formData.public_base_url}
              onChange={(value) => updateFormField('public_base_url', value)}
            />
          </Field>
          <Field label={t('Return format')}>
            <Select
              value={formData.return_format}
              optionList={RETURN_FORMAT_OPTIONS.map((item) => ({
                ...item,
                label: t(item.label),
              }))}
              onChange={(value) => updateFormField('return_format', value)}
            />
          </Field>
        </div>
      );
    }

    return (
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
          gap: 12,
        }}
      >
        <Field label={t('Endpoint')}>
          <Input
            value={formData.endpoint}
            onChange={(value) => updateFormField('endpoint', value)}
          />
        </Field>
        <Field label={t('Region')}>
          <Input
            value={formData.region}
            onChange={(value) => updateFormField('region', value)}
          />
        </Field>
        <Field label={t('Bucket')}>
          <Input
            value={formData.bucket}
            onChange={(value) => updateFormField('bucket', value)}
          />
        </Field>
        <Field label={t('Access key ID')}>
          <Input
            value={formData.access_key_id}
            onChange={(value) => updateFormField('access_key_id', value)}
          />
        </Field>
        <Field label={t('Secret access key')}>
          <Input
            type='password'
            value={formData.secret_access_key}
            placeholder={maskedSecretAccessKeyHint || undefined}
            onChange={(value) => updateFormField('secret_access_key', value)}
          />
        </Field>
        <Field label={t('Key prefix')}>
          <Input
            value={formData.key_prefix}
            onChange={(value) => updateFormField('key_prefix', value)}
          />
        </Field>
        <Field label={t('Public base URL')}>
          <Input
            value={formData.public_base_url}
            onChange={(value) => updateFormField('public_base_url', value)}
          />
        </Field>
        <Field label={t('Force path style')}>
          <Switch
            checked={formData.force_path_style}
            onChange={(checked) => updateFormField('force_path_style', checked)}
          />
        </Field>
      </div>
    );
  };

  const columns = [
    {
      title: t('Name'),
      dataIndex: 'name',
      render: (text) => <Text strong>{text}</Text>,
    },
    {
      title: t('Type'),
      dataIndex: 'type',
      render: (value) => (
        <Tag shape='circle'>
          {t(TYPE_OPTIONS.find((item) => item.value === value)?.label || value)}
        </Tag>
      ),
    },
    {
      title: t('Status'),
      dataIndex: 'status',
      render: (value) => (
        <Tag color={value === '1' ? 'green' : 'grey'} shape='circle'>
          {value === '1' ? t('Enabled') : t('Disabled')}
        </Tag>
      ),
    },
    {
      title: t('Default'),
      dataIndex: 'is_default',
      render: (value) => (
        <Tag color={value === '1' ? 'blue' : 'grey'} shape='circle'>
          {value === '1' ? t('Yes') : t('No')}
        </Tag>
      ),
    },
    {
      title: t('Chunk threshold'),
      dataIndex: 'chunk_threshold',
      render: (value) => formatBytes(Number(value)),
    },
    {
      title: t('Chunk size'),
      dataIndex: 'chunk_size',
      render: (value) => formatBytes(Number(value)),
    },
    {
      title: t('Updated'),
      dataIndex: 'update_time',
      render: (value) => new Date(value * 1000).toLocaleString(),
    },
    {
      title: t('Actions'),
      width: 220,
      render: (_, record) => (
        <Space>
          <Button
            theme='light'
            type='tertiary'
            icon={<Edit size={14} />}
            onClick={() => handleOpenEdit(record)}
            aria-label={t('Edit')}
          />
          <Button
            theme='light'
            type='tertiary'
            icon={<Star size={14} />}
            disabled={record.status !== '1' || record.is_default === '1'}
            onClick={() => handleSetDefault(record)}
            aria-label={t('Set default')}
          />
          <Button
            theme='light'
            type='tertiary'
            icon={<Radar size={14} />}
            onClick={() => handleProbeSaved(record)}
            aria-label={t('Probe')}
          />
          <Button
            theme='light'
            type='tertiary'
            icon={<Power size={14} />}
            onClick={() => handleToggleStatus(record)}
            aria-label={record.status === '1' ? t('Disable') : t('Enable')}
          />
          <Button
            theme='light'
            type='danger'
            icon={<Trash2 size={14} />}
            onClick={() => setDeleteTarget(record)}
            aria-label={t('Delete')}
          />
        </Space>
      ),
    },
  ];

  return (
    <>
      <Spin spinning={loading}>
        <Card
          title={t('Data Management')}
          style={{ marginTop: '10px' }}
          headerExtraContent={
            <Button icon={<Plus size={14} />} onClick={handleOpenCreate}>
              {t('Add Channel')}
            </Button>
          }
        >
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              gap: 12,
            }}
          >
            <Text type='secondary'>
              {t(
                'Configure file upload channels and keep secrets server-side only.',
              )}
            </Text>
            {defaultChannel ? (
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: 12,
                  padding: '10px 12px',
                  borderRadius: 8,
                  background: 'var(--semi-color-fill-0)',
                  border: '1px solid var(--semi-color-border)',
                }}
              >
                <Text type='secondary'>{t('Current default')}</Text>
                <Text strong>{defaultChannel.name}</Text>
              </div>
            ) : null}

            <Table
              rowKey='id'
              columns={columns}
              dataSource={channels}
              pagination={false}
              loading={loading}
              scroll={{ x: 'max-content' }}
              empty={
                <Empty
                  image={
                    <IllustrationNoResult style={{ width: 150, height: 150 }} />
                  }
                  darkModeImage={
                    <IllustrationNoResultDark
                      style={{ width: 150, height: 150 }}
                    />
                  }
                  description={t('No file upload channels configured yet.')}
                  style={{ padding: 30 }}
                />
              }
            />
          </div>
        </Card>
      </Spin>

      <Modal
        title={t(editingChannel ? 'Edit Channel' : 'Add Channel')}
        visible={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setEditingChannel(null);
        }}
        footer={
          <Space>
            <Button onClick={() => setModalVisible(false)}>
              {t('Cancel')}
            </Button>
            <Button
              type='tertiary'
              loading={probeLoading}
              onClick={handleProbeDraft}
            >
              {t('Probe')}
            </Button>
            <Button theme='solid' loading={modalLoading} onClick={handleSave}>
              {t(editingChannel ? 'Update' : 'Create')}
            </Button>
          </Space>
        }
        width={860}
        closeOnEsc
      >
        <div style={{ display: 'grid', gap: 16 }}>
          <Text type='secondary'>
            {t('Secrets stay on the server and are never shown in full.')}
          </Text>

          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
              gap: 12,
            }}
          >
            <Field label={t('Name')}>
              <Input
                value={formData.name}
                onChange={(value) => updateFormField('name', value)}
              />
            </Field>
            <Field label={t('Type')}>
              <Select
                value={formData.type}
                disabled={!!editingChannel}
                optionList={TYPE_OPTIONS.map((item) => ({
                  ...item,
                  label: t(item.label),
                }))}
                onChange={(value) =>
                  setFormData((prev) => resetTypeSpecificFields(value, prev))
                }
              />
            </Field>
            <Field
              label={t('Chunk threshold')}
              description={t(
                'Unit: B. Files above this threshold use chunked upload.',
              )}
            >
              <Input
                type='number'
                value={formData.chunk_threshold}
                onChange={(value) => updateFormField('chunk_threshold', value)}
              />
            </Field>
            <Field
              label={t('Chunk size')}
              description={t('Unit: B. Size of each uploaded chunk.')}
            >
              <Input
                type='number'
                value={formData.chunk_size}
                onChange={(value) => updateFormField('chunk_size', value)}
              />
            </Field>
            <Field
              label={t('Max size')}
              description={t(
                'Unit: B. Maximum allowed file size. 0 means unlimited.',
              )}
            >
              <Input
                type='number'
                value={formData.max_size}
                onChange={(value) => updateFormField('max_size', value)}
              />
            </Field>
            <Field label={t('Enabled')}>
              <Switch
                checked={formData.status === '1'}
                disabled={formData.is_default === '1'}
                onChange={(checked) =>
                  updateFormField('status', checked ? '1' : '0')
                }
              />
            </Field>
            <Field label={t('Default')}>
              <Switch
                checked={formData.is_default === '1'}
                disabled={formData.status !== '1'}
                onChange={(checked) =>
                  updateFormField('is_default', checked ? '1' : '0')
                }
              />
            </Field>
          </div>

          {renderTypeFields()}
        </div>
      </Modal>

      <Modal
        title={t('Confirm probe')}
        visible={probeConfirmVisible}
        onCancel={() => {
          setProbeConfirmVisible(false);
          setPendingProbe(null);
        }}
        footer={
          <Space>
            <Button
              onClick={() => {
                setProbeConfirmVisible(false);
                setPendingProbe(null);
              }}
            >
              {t('Cancel')}
            </Button>
            <Button
              theme='solid'
              loading={probeLoading}
              onClick={handleConfirmProbe}
            >
              {t('Confirm')}
            </Button>
          </Space>
        }
      >
        <Text>
          {t(
            'Probe will upload temporary file {{name}}, file size: {{size}}. Please confirm or cancel.',
            {
              name: probeFileInfo.file_name,
              size: formatBytes(Number(probeFileInfo.file_size || 0)),
            },
          )}
        </Text>
      </Modal>

      <Modal
        title={t('Probe result')}
        visible={probeResultVisible}
        onCancel={() => setProbeResultVisible(false)}
        closeOnEsc
        footer={
          <Space>
            <Button onClick={handleCopyProbeUrl} disabled={!probeResult?.url}>
              {t('Copy URL')}
            </Button>
            <Button theme='solid' onClick={() => setProbeResultVisible(false)}>
              {t('Close')}
            </Button>
          </Space>
        }
      >
        {displayProbeResult ? (
          <div style={{ display: 'grid', gap: 12 }}>
            <Card bodyStyle={{ padding: 16 }}>
              <div style={{ display: 'grid', gap: 8 }}>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    gap: 12,
                  }}
                >
                  <Text type='secondary'>{t('File')}</Text>
                  <Text strong>{displayProbeResult.file_name}</Text>
                </div>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    gap: 12,
                  }}
                >
                  <Text type='secondary'>{t('File size')}</Text>
                  <Text strong>
                    {formatBytes(Number(displayProbeResult.file_size || 0))}
                  </Text>
                </div>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    gap: 12,
                  }}
                >
                  <Text type='secondary'>{t('Latency')}</Text>
                  <Text strong>{displayProbeResult.latency_ms} ms</Text>
                </div>
              </div>
            </Card>
            <div style={{ display: 'grid', gap: 6 }}>
              <Text type='secondary'>{t('Response URL')}</Text>
              <a
                href={displayProbeResult.url}
                target='_blank'
                rel='noreferrer'
                style={{ wordBreak: 'break-all' }}
              >
                {displayProbeResult.url}
              </a>
            </div>
          </div>
        ) : null}
      </Modal>

      <Modal
        title={t('Delete Channel')}
        visible={!!deleteTarget}
        onOk={handleDelete}
        onCancel={() => setDeleteTarget(null)}
        okText={t('Delete')}
        cancelText={t('Cancel')}
        okButtonProps={{ theme: 'solid', type: 'danger' }}
      >
        <Text>
          {t('Delete "{{name}}"? This cannot be undone.', {
            name: deleteTarget?.name || '',
          })}
        </Text>
      </Modal>
    </>
  );
}
