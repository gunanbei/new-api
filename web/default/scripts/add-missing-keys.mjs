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
import fs from "node:fs/promises";
import path from "node:path";

const LOCALES_DIR = path.resolve("src/i18n/locales");

function stableStringify(obj) {
  return JSON.stringify(obj, null, 2) + "\n";
}

const newKeys = {
  en: {
    "Allowed range: {{min}}-{{max}} minutes (up to 7 days)":
      "Allowed range: {{min}}-{{max}} minutes (up to 7 days)",
    "Choose a fixed interval or a cron schedule for cleanup scans.":
      "Choose a fixed interval or a cron schedule for cleanup scans.",
    "Cron schedule": "Cron schedule",
    "Every N hours": "Every N hours",
    "Execution times use the server local timezone. Preview below follows the server clock.":
      "Execution times use the server local timezone. Preview below follows the server clock.",
    "Fixed interval": "Fixed interval",
    Friday: "Friday",
    "Generated cron expression": "Generated cron expression",
    "Hour interval": "Hour interval",
    "Inflight log cleanup interval must be between 1 and 10080 minutes":
      "Inflight log cleanup interval must be between 1 and 10080 minutes",
    "Inflight log cleanup schedule mode": "Inflight log cleanup schedule mode",
    "Invalid cron expression": "Invalid cron expression",
    Monday: "Monday",
    "Next 6 scheduled runs": "Next 6 scheduled runs",
    Saturday: "Saturday",
    "Schedule preset": "Schedule preset",
    "Server timezone: {{timezone}}": "Server timezone: {{timezone}}",
    "Start live refresh": "Start live refresh",
    "Stop live refresh": "Stop live refresh",
    Sunday: "Sunday",
    Thursday: "Thursday",
    Tuesday: "Tuesday",
    Wednesday: "Wednesday",
    Weekday: "Weekday",
    "Automatic inflight log cleanup is disabled":
      "Automatic inflight log cleanup is disabled",
    "Clean terminal inflight logs every {{minutes}} minutes":
      "Clean terminal inflight logs every {{minutes}} minutes",
    "Clean terminal inflight logs on schedule: {{expr}}":
      "Clean terminal inflight logs on schedule: {{expr}}",
    "Cleanup in progress": "Cleanup in progress",
    "Loading cleanup schedule...": "Loading cleanup schedule...",
    "Next cleanup: {{time}}": "Next cleanup: {{time}}",
    "Async Logs": "Async Logs",
    "Showing the latest {{count}} events while response is in progress":
      "Showing the latest {{count}} events while response is in progress",
    "API token": "API token",
    "Access key ID": "Access key ID",
    "Add Channel": "Add Channel",
    "Auth type": "Auth type",
    Basic: "Basic",
    Bearer: "Bearer",
    Bucket: "Bucket",
    "Chunk size": "Chunk size",
    "Chunk threshold": "Chunk threshold",
    "Cloudflare ImageBed": "Cloudflare ImageBed",
    "Configure file upload channels and keep secrets server-side only.":
      "Configure file upload channels and keep secrets server-side only.",
    "Current default": "Current default",
    "Data Management": "Data Management",
    "Default channel updated": "Default channel updated",
    'Delete "{{name}}"? This cannot be undone.':
      'Delete "{{name}}"? This cannot be undone.',
    "Force path style": "Force path style",
    Full: "Full",
    "Key prefix": "Key prefix",
    "Max size": "Max size",
    "No file upload channels configured yet.":
      "No file upload channels configured yet.",
    "Path prefix": "Path prefix",
    Probe: "Probe",
    "Probe uploaded the built-in temporary file and returned the final address.":
      "Probe uploaded the built-in temporary file and returned the final address.",
    "Probe result": "Probe result",
    "Probe will upload temporary file {{name}}, file size: {{size}}. Please confirm or cancel.":
      "Probe will upload temporary file {{name}}, file size: {{size}}. Please confirm or cancel.",
    "Probe succeeded": "Probe succeeded",
    "Public base URL": "Public base URL",
    "Response URL": "Response URL",
    Region: "Region",
    "Return format": "Return format",
    "S3 Compatible Storage": "S3 Compatible Storage",
    "Secret access key": "Secret access key",
    "Secrets stay on the server and are never shown in full.":
      "Secrets stay on the server and are never shown in full.",
    "Set default": "Set default",
    "Status updated": "Status updated",
    "Timeout ms": "Timeout ms",
    "Confirm probe": "Confirm probe",
    File: "File",
    "File size": "File size",
    Latency: "Latency",
    "Unit: B. Files above this threshold use chunked upload.":
      "Unit: B. Files above this threshold use chunked upload.",
    "Unit: B. Maximum allowed file size. 0 means unlimited.":
      "Unit: B. Maximum allowed file size. 0 means unlimited.",
    "Unit: B. Size of each uploaded chunk.":
      "Unit: B. Size of each uploaded chunk.",
    "Upstream upload channel": "Upstream upload channel",
    "Upload channel": "Upload channel",
    "Upload folder": "Upload folder",
    WebDAV: "WebDAV",
  },
  zh: {
    "Allowed range: {{min}}-{{max}} minutes (up to 7 days)":
      "可设置范围：{{min}}–{{max}} 分钟（最长 7 天）",
    "Choose a fixed interval or a cron schedule for cleanup scans.":
      "选择固定间隔或 CRON 计划来扫描清理终态在途日志。",
    "Cron schedule": "CRON 计划",
    "Every N hours": "每 N 小时",
    "Execution times use the server local timezone. Preview below follows the server clock.":
      "执行时间以服务器本地时区为准，下方预览基于服务器时钟计算。",
    "Fixed interval": "固定间隔",
    Friday: "周五",
    "Generated cron expression": "生成的 CRON 表达式",
    "Hour interval": "小时间隔",
    "Inflight log cleanup interval must be between 1 and 10080 minutes":
      "在途日志清理间隔必须在 1–10080 分钟之间",
    "Inflight log cleanup schedule mode": "在途日志清理调度方式",
    "Invalid cron expression": "CRON 表达式无效",
    Monday: "周一",
    "Next 6 scheduled runs": "未来 6 次计划执行时间",
    Saturday: "周六",
    "Schedule preset": "计划预设",
    "Server timezone: {{timezone}}": "服务器时区：{{timezone}}",
    "Start live refresh": "开始实时刷新",
    "Stop live refresh": "停止实时刷新",
    Sunday: "周日",
    Thursday: "周四",
    Tuesday: "周二",
    Wednesday: "周三",
    Weekday: "星期",
    "Automatic inflight log cleanup is disabled": "在途日志自动清理已关闭",
    "Clean terminal inflight logs every {{minutes}} minutes":
      "每 {{minutes}} 分钟清理终态在途日志",
    "Clean terminal inflight logs on schedule: {{expr}}":
      "按计划清理终态在途日志：{{expr}}",
    "Cleanup in progress": "清理进行中",
    "Loading cleanup schedule...": "正在加载清理计划...",
    "Next cleanup: {{time}}": "下次清理：{{time}}",
    "Async Logs": "异步日志",
    "Showing the latest {{count}} events while response is in progress":
      "响应进行中，仅显示最近 {{count}} 条事件",
    "API token": "API 令牌",
    "Access key ID": "Access Key ID",
    "Add Channel": "添加渠道",
    "Auth type": "认证方式",
    Basic: "Basic",
    Bearer: "Bearer",
    Bucket: "存储桶",
    "Chunk size": "分片大小",
    "Chunk threshold": "分片阈值",
    "Cloudflare ImageBed": "Cloudflare ImageBed",
    "Configure file upload channels and keep secrets server-side only.":
      "配置文件上传渠道，敏感密钥仅保留在服务端。",
    "Current default": "当前默认渠道",
    "Data Management": "数据管理",
    "Default channel updated": "默认渠道已更新",
    'Delete "{{name}}"? This cannot be undone.':
      "删除“{{name}}”？此操作无法撤销。",
    "Force path style": "强制路径风格",
    Full: "完整",
    "Key prefix": "对象前缀",
    "Max size": "最大大小",
    "No file upload channels configured yet.": "暂无文件上传渠道配置。",
    "Path prefix": "路径前缀",
    Probe: "探测",
    "Probe uploaded the built-in temporary file and returned the final address.":
      "探测已上传内置临时文件，并返回最终访问地址。",
    "Probe result": "探测结果",
    "Probe will upload temporary file {{name}}, file size: {{size}}. Please confirm or cancel.":
      "探测将上传临时文件 {{name}}，文件大小为：{{size}}，请确认或取消。",
    "Probe succeeded": "探测成功",
    "Public base URL": "公开访问地址",
    "Response URL": "响应地址",
    Region: "区域",
    "Return format": "返回格式",
    "S3 Compatible Storage": "S3 兼容存储",
    "Secret access key": "Secret Access Key",
    "Secrets stay on the server and are never shown in full.":
      "敏感密钥仅保留在服务端，不会完整回显。",
    "Set default": "设为默认",
    "Status updated": "状态已更新",
    "Timeout ms": "超时时间（毫秒）",
    "Confirm probe": "确认探测",
    File: "文件",
    "File size": "文件大小",
    Latency: "延迟",
    "Unit: B. Files above this threshold use chunked upload.":
      "单位：B。超过该阈值后将使用分片上传。",
    "Unit: B. Maximum allowed file size. 0 means unlimited.":
      "单位：B。允许上传的最大文件大小，0 表示不限。",
    "Unit: B. Size of each uploaded chunk.": "单位：B。每个分片的大小。",
    "Upstream upload channel": "上游上传通道",
    "Upload channel": "上传通道",
    "Upload folder": "上传目录",
    WebDAV: "WebDAV",
  },
  fr: {
    "Allowed range: {{min}}-{{max}} minutes (up to 7 days)":
      "Plage autorisee : {{min}}-{{max}} minutes (jusqu a 7 jours)",
    "Choose a fixed interval or a cron schedule for cleanup scans.":
      "Choisissez un intervalle fixe ou une planification cron pour les scans de nettoyage.",
    "Cron schedule": "Planification cron",
    "Every N hours": "Toutes les N heures",
    "Execution times use the server local timezone. Preview below follows the server clock.":
      "Les heures d'execution utilisent le fuseau horaire local du serveur. L'apercu ci-dessous suit l'horloge du serveur.",
    "Fixed interval": "Intervalle fixe",
    Friday: "Vendredi",
    "Generated cron expression": "Expression cron generee",
    "Hour interval": "Intervalle horaire",
    "Inflight log cleanup interval must be between 1 and 10080 minutes":
      "L'intervalle de nettoyage des journaux en cours doit etre compris entre 1 et 10080 minutes",
    "Inflight log cleanup schedule mode":
      "Mode de planification du nettoyage des journaux en cours",
    "Invalid cron expression": "Expression cron invalide",
    Monday: "Lundi",
    "Next 6 scheduled runs": "6 prochaines executions planifiees",
    Saturday: "Samedi",
    "Schedule preset": "Preset de planification",
    "Server timezone: {{timezone}}": "Fuseau horaire du serveur : {{timezone}}",
    "Start live refresh": "Demarrer le rafraichissement en direct",
    "Stop live refresh": "Arreter le rafraichissement en direct",
    Sunday: "Dimanche",
    Thursday: "Jeudi",
    Tuesday: "Mardi",
    Wednesday: "Mercredi",
    Weekday: "Jour de la semaine",
    "Automatic inflight log cleanup is disabled":
      "Le nettoyage automatique des journaux en cours est desactive",
    "Clean terminal inflight logs every {{minutes}} minutes":
      "Nettoyer les journaux en cours termines toutes les {{minutes}} minutes",
    "Clean terminal inflight logs on schedule: {{expr}}":
      "Nettoyer les journaux en cours termines selon le planning : {{expr}}",
    "Cleanup in progress": "Nettoyage en cours",
    "Loading cleanup schedule...": "Chargement du planning de nettoyage...",
    "Next cleanup: {{time}}": "Prochain nettoyage : {{time}}",
    "Async Logs": "Journaux asynchrones",
    "Showing the latest {{count}} events while response is in progress":
      "Affichage des {{count}} derniers evenements pendant la reponse en cours",
    "API token": "Jeton API",
    "Access key ID": "ID de cle d'acces",
    "Add Channel": "Ajouter un canal",
    "Auth type": "Type d'authentification",
    Basic: "Basic",
    Bearer: "Bearer",
    Bucket: "Bucket",
    "Chunk size": "Taille des segments",
    "Chunk threshold": "Seuil de segmentation",
    "Cloudflare ImageBed": "Cloudflare ImageBed",
    "Configure file upload channels and keep secrets server-side only.":
      "Configurez les canaux d'upload de fichiers et conservez les secrets uniquement cote serveur.",
    "Current default": "Canal par defaut actuel",
    "Data Management": "Gestion des donnees",
    "Default channel updated": "Canal par defaut mis a jour",
    'Delete "{{name}}"? This cannot be undone.':
      'Supprimer "{{name}}" ? Cette action est irreversible.',
    "Force path style": "Forcer le style de chemin",
    Full: "Complet",
    "Key prefix": "Prefixe de cle",
    "Max size": "Taille maximale",
    "No file upload channels configured yet.":
      "Aucun canal d'upload de fichiers n'est configure.",
    "Path prefix": "Prefixe de chemin",
    Probe: "Verifier",
    "Probe uploaded the built-in temporary file and returned the final address.":
      "Le test a televerse le fichier temporaire integre et a retourne l'adresse finale.",
    "Probe result": "Resultat du test",
    "Probe will upload temporary file {{name}}, file size: {{size}}. Please confirm or cancel.":
      "Le test televersera le fichier temporaire {{name}}, taille : {{size}}. Veuillez confirmer ou annuler.",
    "Probe succeeded": "Verification reussie",
    "Public base URL": "URL publique de base",
    "Response URL": "URL retournee",
    Region: "Region",
    "Return format": "Format de retour",
    "S3 Compatible Storage": "Stockage compatible S3",
    "Secret access key": "Cle secrete d'acces",
    "Secrets stay on the server and are never shown in full.":
      "Les secrets restent cote serveur et ne sont jamais affiches en entier.",
    "Set default": "Definir par defaut",
    "Status updated": "Statut mis a jour",
    "Timeout ms": "Delai en ms",
    "Confirm probe": "Confirmer le test",
    File: "Fichier",
    "File size": "Taille du fichier",
    Latency: "Latence",
    "Unit: B. Files above this threshold use chunked upload.":
      "Unite : B. Au-dela de ce seuil, l'envoi passe en mode segmente.",
    "Unit: B. Maximum allowed file size. 0 means unlimited.":
      "Unite : B. Taille maximale autorisee pour un fichier. 0 signifie illimite.",
    "Unit: B. Size of each uploaded chunk.":
      "Unite : B. Taille de chaque segment televerse.",
    "Upstream upload channel": "Canal d'upload amont",
    "Upload channel": "Canal d'upload",
    "Upload folder": "Dossier d'upload",
    WebDAV: "WebDAV",
  },
  ja: {
    "Allowed range: {{min}}-{{max}} minutes (up to 7 days)":
      "設定可能範囲：{{min}}〜{{max}} 分（最長 7 日）",
    "Choose a fixed interval or a cron schedule for cleanup scans.":
      "クリーンアップ走査には固定間隔または cron スケジュールを選択します。",
    "Cron schedule": "cron スケジュール",
    "Every N hours": "N 時間ごと",
    "Execution times use the server local timezone. Preview below follows the server clock.":
      "実行時刻はサーバーのローカルタイムゾーンを使用します。下のプレビューはサーバー時計に基づきます。",
    "Fixed interval": "固定間隔",
    Friday: "金曜日",
    "Generated cron expression": "生成された cron 式",
    "Hour interval": "時間間隔",
    "Inflight log cleanup interval must be between 1 and 10080 minutes":
      "進行中ログのクリーンアップ間隔は 1〜10080 分の範囲で設定してください",
    "Inflight log cleanup schedule mode": "進行中ログのクリーンアップ方式",
    "Invalid cron expression": "cron 式が無効です",
    Monday: "月曜日",
    "Next 6 scheduled runs": "今後 6 回の実行予定",
    Saturday: "土曜日",
    "Schedule preset": "スケジュールプリセット",
    "Server timezone: {{timezone}}": "サーバータイムゾーン：{{timezone}}",
    "Start live refresh": "リアルタイム更新を開始",
    "Stop live refresh": "リアルタイム更新を停止",
    Sunday: "日曜日",
    Thursday: "木曜日",
    Tuesday: "火曜日",
    Wednesday: "水曜日",
    Weekday: "曜日",
    "Automatic inflight log cleanup is disabled":
      "進行中ログの自動クリーンアップは無効です",
    "Clean terminal inflight logs every {{minutes}} minutes":
      "終了した進行中ログを {{minutes}} 分ごとにクリーンアップ",
    "Clean terminal inflight logs on schedule: {{expr}}":
      "スケジュールに従って終了した進行中ログをクリーンアップ：{{expr}}",
    "Cleanup in progress": "クリーンアップ実行中",
    "Loading cleanup schedule...": "クリーンアップ予定を読み込み中...",
    "Next cleanup: {{time}}": "次回クリーンアップ：{{time}}",
    "Async Logs": "非同期ログ",
    "Showing the latest {{count}} events while response is in progress":
      "応答進行中は直近 {{count}} 件のイベントのみ表示します",
    "API token": "API トークン",
    "Access key ID": "アクセスキー ID",
    "Add Channel": "チャネルを追加",
    "Auth type": "認証方式",
    Basic: "Basic",
    Bearer: "Bearer",
    Bucket: "バケット",
    "Chunk size": "チャンクサイズ",
    "Chunk threshold": "チャンク閾値",
    "Cloudflare ImageBed": "Cloudflare ImageBed",
    "Configure file upload channels and keep secrets server-side only.":
      "ファイルアップロードチャネルを設定し、シークレットはサーバー側のみに保持します。",
    "Current default": "現在のデフォルト",
    "Data Management": "データ管理",
    "Default channel updated": "デフォルトチャネルを更新しました",
    'Delete "{{name}}"? This cannot be undone.':
      "「{{name}}」を削除しますか？この操作は元に戻せません。",
    "Force path style": "パススタイルを強制",
    Full: "完全",
    "Key prefix": "キープレフィックス",
    "Max size": "最大サイズ",
    "No file upload channels configured yet.":
      "ファイルアップロードチャネルはまだ設定されていません。",
    "Path prefix": "パスプレフィックス",
    Probe: "疎通確認",
    "Probe uploaded the built-in temporary file and returned the final address.":
      "疎通確認は内蔵の一時ファイルをアップロードし、最終URLを返しました。",
    "Probe result": "疎通確認結果",
    "Probe will upload temporary file {{name}}, file size: {{size}}. Please confirm or cancel.":
      "疎通確認は一時ファイル {{name}} をアップロードします。ファイルサイズ: {{size}}。確認またはキャンセルしてください。",
    "Probe succeeded": "疎通確認に成功しました",
    "Public base URL": "公開ベース URL",
    "Response URL": "応答URL",
    Region: "リージョン",
    "Return format": "返却形式",
    "S3 Compatible Storage": "S3 互換ストレージ",
    "Secret access key": "シークレットアクセスキー",
    "Secrets stay on the server and are never shown in full.":
      "シークレットはサーバー側にのみ保持され、完全な内容は表示されません。",
    "Set default": "デフォルトに設定",
    "Status updated": "ステータスを更新しました",
    "Timeout ms": "タイムアウト（ms）",
    "Confirm probe": "疎通確認",
    File: "ファイル",
    "File size": "ファイルサイズ",
    Latency: "レイテンシ",
    "Unit: B. Files above this threshold use chunked upload.":
      "単位: B。このしきい値を超えると分割アップロードを使用します。",
    "Unit: B. Maximum allowed file size. 0 means unlimited.":
      "単位: B。アップロード可能な最大ファイルサイズです。0 は無制限を意味します。",
    "Unit: B. Size of each uploaded chunk.":
      "単位: B。各チャンクのサイズです。",
    "Upstream upload channel": "上游アップロードチャネル",
    "Upload channel": "アップロードチャネル",
    "Upload folder": "アップロードフォルダ",
    WebDAV: "WebDAV",
  },
  ru: {
    "Allowed range: {{min}}-{{max}} minutes (up to 7 days)":
      "Допустимый диапазон: {{min}}–{{max}} минут (до 7 дней)",
    "Choose a fixed interval or a cron schedule for cleanup scans.":
      "Выберите фиксированный интервал или cron-расписание для сканирования очистки.",
    "Cron schedule": "Cron-расписание",
    "Every N hours": "Каждые N часов",
    "Execution times use the server local timezone. Preview below follows the server clock.":
      "Время выполнения использует локальный часовой пояс сервера. Предпросмотр ниже основан на часах сервера.",
    "Fixed interval": "Фиксированный интервал",
    Friday: "Пятница",
    "Generated cron expression": "Сгенерированное cron-выражение",
    "Hour interval": "Интервал в часах",
    "Inflight log cleanup interval must be between 1 and 10080 minutes":
      "Интервал очистки журналов в процессе должен быть от 1 до 10080 минут",
    "Inflight log cleanup schedule mode":
      "Режим расписания очистки журналов в процессе",
    "Invalid cron expression": "Недопустимое cron-выражение",
    Monday: "Понедельник",
    "Next 6 scheduled runs": "Следующие 6 запланированных запусков",
    Saturday: "Суббота",
    "Schedule preset": "Предустановка расписания",
    "Server timezone: {{timezone}}": "Часовой пояс сервера: {{timezone}}",
    "Start live refresh": "Включить обновление в реальном времени",
    "Stop live refresh": "Остановить обновление в реальном времени",
    Sunday: "Воскресенье",
    Thursday: "Четверг",
    Tuesday: "Вторник",
    Wednesday: "Среда",
    Weekday: "День недели",
    "Automatic inflight log cleanup is disabled":
      "Автоматическая очистка журналов в процессе отключена",
    "Clean terminal inflight logs every {{minutes}} minutes":
      "Очищать завершенные журналы в процессе каждые {{minutes}} минут",
    "Clean terminal inflight logs on schedule: {{expr}}":
      "Очищать завершенные журналы в процессе по расписанию: {{expr}}",
    "Cleanup in progress": "Очистка выполняется",
    "Loading cleanup schedule...": "Загрузка расписания очистки...",
    "Next cleanup: {{time}}": "Следующая очистка: {{time}}",
    "Async Logs": "Асинхронные журналы",
    "Showing the latest {{count}} events while response is in progress":
      "Показаны последние {{count}} событий, пока ответ еще выполняется",
    "API token": "API-токен",
    "Access key ID": "ID ключа доступа",
    "Add Channel": "Добавить канал",
    "Auth type": "Тип аутентификации",
    Basic: "Basic",
    Bearer: "Bearer",
    Bucket: "Бакет",
    "Chunk size": "Размер чанка",
    "Chunk threshold": "Порог чанков",
    "Cloudflare ImageBed": "Cloudflare ImageBed",
    "Configure file upload channels and keep secrets server-side only.":
      "Настройте каналы загрузки файлов, а секреты храните только на стороне сервера.",
    "Current default": "Текущий канал по умолчанию",
    "Data Management": "Управление данными",
    "Default channel updated": "Канал по умолчанию обновлен",
    'Delete "{{name}}"? This cannot be undone.':
      'Удалить "{{name}}"? Это действие нельзя отменить.',
    "Force path style": "Принудительный path-style",
    Full: "Полный",
    "Key prefix": "Префикс ключа",
    "Max size": "Максимальный размер",
    "No file upload channels configured yet.":
      "Каналы загрузки файлов еще не настроены.",
    "Path prefix": "Префикс пути",
    Probe: "Проверить",
    "Probe uploaded the built-in temporary file and returned the final address.":
      "Проверка загрузила встроенный временный файл и вернула итоговый адрес.",
    "Probe result": "Результат проверки",
    "Probe will upload temporary file {{name}}, file size: {{size}}. Please confirm or cancel.":
      "Проверка загрузит временный файл {{name}}, размер файла: {{size}}. Подтвердите или отмените.",
    "Probe succeeded": "Проверка прошла успешно",
    "Public base URL": "Публичный базовый URL",
    "Response URL": "URL ответа",
    Region: "Регион",
    "Return format": "Формат возврата",
    "S3 Compatible Storage": "S3-совместимое хранилище",
    "Secret access key": "Секретный ключ доступа",
    "Secrets stay on the server and are never shown in full.":
      "Секреты остаются на сервере и никогда не показываются полностью.",
    "Set default": "Сделать по умолчанию",
    "Status updated": "Статус обновлен",
    "Timeout ms": "Таймаут (мс)",
    "Confirm probe": "Подтвердить проверку",
    File: "Файл",
    "File size": "Размер файла",
    Latency: "Задержка",
    "Unit: B. Files above this threshold use chunked upload.":
      "Единица: B. Файлы больше этого порога загружаются по частям.",
    "Unit: B. Maximum allowed file size. 0 means unlimited.":
      "Единица: B. Максимально допустимый размер файла. 0 означает без ограничений.",
    "Unit: B. Size of each uploaded chunk.":
      "Единица: B. Размер каждой части загрузки.",
    "Upstream upload channel": "Верхний канал загрузки",
    "Upload channel": "Канал загрузки",
    "Upload folder": "Папка загрузки",
    WebDAV: "WebDAV",
  },
  vi: {
    "Allowed range: {{min}}-{{max}} minutes (up to 7 days)":
      "Pham vi cho phep: {{min}}-{{max}} phut (toi da 7 ngay)",
    "Choose a fixed interval or a cron schedule for cleanup scans.":
      "Chon khoang thoi gian co dinh hoac lich cron de quet don dep.",
    "Cron schedule": "Lich cron",
    "Every N hours": "Moi N gio",
    "Execution times use the server local timezone. Preview below follows the server clock.":
      "Thoi gian thuc thi dung mui gio cuc bo cua may chu. Xem truoc ben duoi theo dong ho may chu.",
    "Fixed interval": "Khoang thoi gian co dinh",
    Friday: "Thu Sau",
    "Generated cron expression": "Bieu thuc cron duoc tao",
    "Hour interval": "Khoang gio",
    "Inflight log cleanup interval must be between 1 and 10080 minutes":
      "Khoang thoi gian don dep nhat ky dang xu ly phai tu 1 den 10080 phut",
    "Inflight log cleanup schedule mode":
      "Che do lich don dep nhat ky dang xu ly",
    "Invalid cron expression": "Bieu thuc cron khong hop le",
    Monday: "Thu Hai",
    "Next 6 scheduled runs": "6 lan chay ke hoach tiep theo",
    Saturday: "Thu Bay",
    "Schedule preset": "Mau lich",
    "Server timezone: {{timezone}}": "Mui gio may chu: {{timezone}}",
    "Start live refresh": "Bat dau lam moi thoi gian thuc",
    "Stop live refresh": "Dung lam moi thoi gian thuc",
    Sunday: "Chu Nhat",
    Thursday: "Thu Tu",
    Tuesday: "Thu Ba",
    Wednesday: "Thu Tu",
    Weekday: "Ngay trong tuan",
    "Automatic inflight log cleanup is disabled":
      "Tat don dep tu dong nhat ky dang xu ly",
    "Clean terminal inflight logs every {{minutes}} minutes":
      "Don dep nhat ky dang xu ly da ket thuc moi {{minutes}} phut",
    "Clean terminal inflight logs on schedule: {{expr}}":
      "Don dep nhat ky dang xu ly da ket thuc theo lich: {{expr}}",
    "Cleanup in progress": "Dang don dep",
    "Loading cleanup schedule...": "Dang tai lich don dep...",
    "Next cleanup: {{time}}": "Lan don dep tiep theo: {{time}}",
    "Async Logs": "Nhat ky bat dong bo",
    "Showing the latest {{count}} events while response is in progress":
      "Dang xu ly phan hoi, chi hien thi {{count}} su kien moi nhat",
    "API token": "API token",
    "Access key ID": "ID khoa truy cap",
    "Add Channel": "Them kenh",
    "Auth type": "Kieu xac thuc",
    Basic: "Basic",
    Bearer: "Bearer",
    Bucket: "Bucket",
    "Chunk size": "Kich thuoc chunk",
    "Chunk threshold": "Nguong chunk",
    "Cloudflare ImageBed": "Cloudflare ImageBed",
    "Configure file upload channels and keep secrets server-side only.":
      "Cau hinh kenh tai tep len va chi giu bi mat o phia may chu.",
    "Current default": "Kenh mac dinh hien tai",
    "Data Management": "Quan ly du lieu",
    "Default channel updated": "Da cap nhat kenh mac dinh",
    'Delete "{{name}}"? This cannot be undone.':
      'Xoa "{{name}}"? Hanh dong nay khong the hoan tac.',
    "Force path style": "Bat buoc kieu duong dan",
    Full: "Day du",
    "Key prefix": "Tien to khoa",
    "Max size": "Kich thuoc toi da",
    "No file upload channels configured yet.":
      "Chua co kenh tai tep len nao duoc cau hinh.",
    "Path prefix": "Tien to duong dan",
    Probe: "Kiem tra",
    "Probe uploaded the built-in temporary file and returned the final address.":
      "Kiem tra da tai len tep tam thoi tich hop san va tra ve dia chi cuoi cung.",
    "Probe result": "Ket qua kiem tra",
    "Probe will upload temporary file {{name}}, file size: {{size}}. Please confirm or cancel.":
      "Kiem tra se tai len tep tam thoi {{name}}, kich thuoc tep: {{size}}. Vui long xac nhan hoac huy.",
    "Probe succeeded": "Kiem tra thanh cong",
    "Public base URL": "URL cong khai co so",
    "Response URL": "URL phan hoi",
    Region: "Vung",
    "Return format": "Dinh dang tra ve",
    "S3 Compatible Storage": "Luu tru tuong thich S3",
    "Secret access key": "Khoa truy cap bi mat",
    "Secrets stay on the server and are never shown in full.":
      "Bi mat chi duoc giu tren may chu va khong bao gio hien day du.",
    "Set default": "Dat mac dinh",
    "Status updated": "Da cap nhat trang thai",
    "Timeout ms": "Thoi gian cho (ms)",
    "Confirm probe": "Xac nhan kiem tra",
    File: "Tep",
    "File size": "Kich thuoc tep",
    Latency: "Do tre",
    "Unit: B. Files above this threshold use chunked upload.":
      "Don vi: B. Tep vuot nguong nay se duoc tai len theo tung phan.",
    "Unit: B. Maximum allowed file size. 0 means unlimited.":
      "Don vi: B. Kich thuoc tep toi da duoc phep. 0 co nghia la khong gioi han.",
    "Unit: B. Size of each uploaded chunk.":
      "Don vi: B. Kich thuoc moi phan tai len.",
    "Upstream upload channel": "Kenh tai len phia tren",
    "Upload channel": "Kenh tai len",
    "Upload folder": "Thu muc tai len",
    WebDAV: "WebDAV",
  },
};

async function main() {
  let totalAdded = 0;

  for (const [locale, trans] of Object.entries(newKeys)) {
    const filePath = path.join(LOCALES_DIR, `${locale}.json`);
    const json = JSON.parse(await fs.readFile(filePath, "utf8"));

    let count = 0;
    for (const [key, value] of Object.entries(trans)) {
      if (!Object.prototype.hasOwnProperty.call(json.translation, key)) {
        json.translation[key] = value;
        count++;
      } else if (json.translation[key] !== value) {
        json.translation[key] = value;
        count++;
      }
    }

    if (count > 0) {
      json.translation = Object.fromEntries(
        Object.entries(json.translation).sort(([a], [b]) => a.localeCompare(b)),
      );
      await fs.writeFile(filePath, stableStringify(json), "utf8");
    }

    console.log(`${locale}: ${count} translations applied`);
    totalAdded += count;
  }

  console.log(`\nTotal: ${totalAdded} translations applied`);
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
