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
import fs from 'node:fs/promises'
import path from 'node:path'

const localesDir = path.resolve('src/i18n/locales')

const newKeys = {
  en: {
    'Archive upload started.': 'Archive upload started.', 'Failed to start archive upload': 'Failed to start archive upload', 'Inflight CSV disk usage': 'Inflight CSV disk usage', 'Inflight CSV files': 'Inflight CSV files', 'Pending archive uploads': 'Pending archive uploads', 'Upload pending archives': 'Upload pending archives',
    'Remove terminal inflight logs and completed CSV archives updated before the selected timestamp.': 'Remove terminal inflight logs and completed CSV archives updated before the selected timestamp.',
    'Inflight trace directory': 'Inflight trace directory', 'Leave empty to use the disk cache directory': 'Leave empty to use the disk cache directory', 'Stores disk-mode inflight trace CSV archives separately from request-body cache files.': 'Stores disk-mode inflight trace CSV archives separately from request-body cache files.',
    'Choose cache directory': 'Choose cache directory', 'Failed to load local cache files': 'Failed to load local cache files', 'Local cache files': 'Local cache files', 'Manage CSV archives downloaded by this browser.': 'Manage CSV archives downloaded by this browser.', 'No local cache files.': 'No local cache files.',
    'Archive downloaded to local cache.': 'Archive downloaded to local cache.',
    'Download archives automatically in this browser': 'Download archives automatically in this browser',
    'Download archive': 'Download archive',
    'Download this archive to your selected local cache directory?': 'Download this archive to your selected local cache directory?',
    'Failed to download archive': 'Failed to download archive',
    'Local cache is only supported in Chromium browsers.': 'Local cache is only supported in Chromium browsers.',
    'This debug log is archived remotely. Download it locally to load request and response bodies.': 'This debug log is archived remotely. Download it locally to load request and response bodies.',
    Years: 'Years', Months: 'Months', Days: 'Days', Hours: 'Hours',
    'Archive retention': 'Archive retention',
    'Archive storage channel': 'Archive storage channel',
    'Archive upload threshold (bytes)': 'Archive upload threshold (bytes)',
    'Archive upload threshold (KB)': 'Archive upload threshold (KB)',
    'Debug log body storage': 'Debug log body storage',
    'Disk CSV': 'Disk CSV',
    'Memory storage increases Redis usage.': 'Memory storage increases Redis usage.',
    'Set all values to 0 to keep archives permanently.': 'Set all values to 0 to keep archives permanently.',
    'Store request and response bodies in memory or per-user CSV files.': 'Store request and response bodies in memory or per-user CSV files.',
    PNG: 'PNG',
    JPEG: 'JPEG',
    WebP: 'WebP',
    'Available after saving': 'Available after saving',
    'Base64 copied to clipboard': 'Base64 copied to clipboard',
    'Convert to Base64': 'Convert to Base64',
    'History works': 'History works',
    'Image to image': 'Image to image',
    Reuse: 'Reuse',
    'Bindings remain disabled until protocol verification passes.':
      'Bindings remain disabled until protocol verification passes.',
    'Last validated': 'Last validated',
    'Not validated': 'Not validated',
    'Protocol verification failed': 'Protocol verification failed',
    'Protocol verification passed': 'Protocol verification passed',
    Processing: 'Processing',
    Revalidate: 'Revalidate',
    'The completion time of the latest protocol compatibility check.':
      'The completion time of the latest protocol compatibility check.',
    'Text to image': 'Text to image',
    'Unable to copy Base64': 'Unable to copy Base64',
    Succeeded: 'Succeeded',
    Uploading: 'Uploading',
    'Validate now': 'Validate now',
    Validated: 'Validated',
    Validating: 'Validating',
    'Validating...': 'Validating...',
    'Validation failed': 'Validation failed',
    'Validation runs immediately in this request. It is not queued and does not contact the upstream provider.':
      'Validation runs immediately in this request. It is not queued and does not contact the upstream provider.',
    Verified: 'Verified',
    'model name changed; validation required':
      'model name changed; validation required',
    'not validated': 'not validated',
    'validation in progress': 'validation in progress',
    validated: 'validated',
  },
  zh: {
    'Archive upload started.': '已开始上传归档。', 'Failed to start archive upload': '无法开始上传归档', 'Inflight CSV disk usage': '在途 CSV 磁盘占用', 'Inflight CSV files': '在途 CSV 文件', 'Pending archive uploads': '待上传归档', 'Upload pending archives': '上传待处理归档',
    'Remove terminal inflight logs and completed CSV archives updated before the selected timestamp.': '删除所选时间之前更新的终态在途日志和已完成的 CSV 归档。',
    'Inflight trace directory': '在途日志目录', 'Leave empty to use the disk cache directory': '留空则使用磁盘缓存目录', 'Stores disk-mode inflight trace CSV archives separately from request-body cache files.': '将磁盘模式的在途日志 CSV 归档与请求体缓存文件分开保存。',
    'Choose cache directory': '选择缓存目录', 'Failed to load local cache files': '无法加载本地缓存文件', 'Local cache files': '本地缓存文件', 'Manage CSV archives downloaded by this browser.': '管理此浏览器下载的 CSV 归档。', 'No local cache files.': '没有本地缓存文件。',
    'Archive downloaded to local cache.': '归档已下载到本地缓存。',
    'Download archives automatically in this browser': '在此浏览器中自动下载归档',
    'Download archive': '下载归档',
    'Download this archive to your selected local cache directory?': '是否将此归档下载到已选择的本地缓存目录？',
    'Failed to download archive': '下载归档失败',
    'Local cache is only supported in Chromium browsers.': '本地缓存仅支持 Chromium 浏览器。',
    'This debug log is archived remotely. Download it locally to load request and response bodies.': '此调试日志已远端归档。下载到本地后可加载请求和响应体。',
    Years: '年', Months: '月', Days: '天', Hours: '小时',
    'Archive retention': '归档保留期',
    'Archive storage channel': '归档存储渠道',
    'Archive upload threshold (bytes)': '归档上传阈值（字节）',
    'Archive upload threshold (KB)': '归档上传阈值（KB）',
    'Debug log body storage': '调试日志请求体存储',
    'Disk CSV': '磁盘 CSV',
    'Memory storage increases Redis usage.': '内存存储会增加 Redis 占用。',
    'Set all values to 0 to keep archives permanently.': '全部设为 0 表示永久保留归档。',
    'Store request and response bodies in memory or per-user CSV files.': '将请求和响应体存储在内存或按用户划分的 CSV 文件中。',
    PNG: 'PNG',
    JPEG: 'JPEG',
    WebP: 'WebP',
    'Available after saving': '保存完成后可用',
    'Base64 copied to clipboard': 'Base64 已复制到剪贴板',
    'Convert to Base64': '转换为 Base64',
    'History works': '历史作品',
    'Image to image': '图生图',
    Reuse: '复用',
    'Bindings remain disabled until protocol verification passes.':
      '绑定会保持禁用，直到协议校验通过。',
    'Last validated': '最近校验时间',
    'Not validated': '未校验',
    'Protocol verification failed': '协议校验未通过',
    'Protocol verification passed': '协议校验已通过',
    Processing: '处理中',
    Revalidate: '重新校验',
    'The completion time of the latest protocol compatibility check.':
      '最近一次协议兼容性校验完成的时间。',
    'Text to image': '文生图',
    'Unable to copy Base64': '无法复制 Base64',
    Succeeded: '已完成',
    Uploading: '保存中',
    'Validate now': '立即校验',
    Validated: '已校验',
    Validating: '校验中',
    'Validating...': '正在校验…',
    'Validation failed': '校验未通过',
    'Validation runs immediately in this request. It is not queued and does not contact the upstream provider.':
      '校验会在当前请求中立即完成，不会进入队列，也不会请求上游服务商。',
    Verified: '已验证',
    'model name changed; validation required': '模型名称已变更，需要重新校验',
    'not validated': '尚未校验',
    'validation in progress': '正在校验',
    validated: '已验证',
  },
  'zh-TW': {
    'Archive upload started.': '已開始上傳封存。', 'Failed to start archive upload': '無法開始上傳封存', 'Inflight CSV disk usage': '在途 CSV 磁碟使用量', 'Inflight CSV files': '在途 CSV 檔案', 'Pending archive uploads': '待上傳封存', 'Upload pending archives': '上傳待處理封存',
    'Remove terminal inflight logs and completed CSV archives updated before the selected timestamp.': '刪除所選時間之前更新的終態在途日誌和已完成的 CSV 封存。',
    'Inflight trace directory': '在途日誌目錄', 'Leave empty to use the disk cache directory': '留空則使用磁碟快取目錄', 'Stores disk-mode inflight trace CSV archives separately from request-body cache files.': '將磁碟模式的在途日誌 CSV 封存與請求內容快取檔案分開儲存。',
    'Choose cache directory': '選擇快取目錄', 'Failed to load local cache files': '無法載入本機快取檔案', 'Local cache files': '本機快取檔案', 'Manage CSV archives downloaded by this browser.': '管理此瀏覽器下載的 CSV 封存。', 'No local cache files.': '沒有本機快取檔案。',
    'Archive downloaded to local cache.': '封存已下載至本機快取。', 'Download archives automatically in this browser': '在此瀏覽器中自動下載封存', 'Download archive': '下載封存', 'Download this archive to your selected local cache directory?': '要將此封存下載到已選取的本機快取目錄嗎？', 'Failed to download archive': '下載封存失敗', 'Local cache is only supported in Chromium browsers.': '本機快取僅支援 Chromium 瀏覽器。', 'This debug log is archived remotely. Download it locally to load request and response bodies.': '此偵錯日誌已遠端封存。下載至本機後可載入請求和回應內容。',
    Years: '年', Months: '月', Days: '天', Hours: '小時',
    'Archive retention': '封存保留期',
    'Archive storage channel': '封存儲存渠道',
    'Archive upload threshold (bytes)': '封存上傳閾值（位元組）',
    'Archive upload threshold (KB)': '封存上傳閾值（KB）',
    'Debug log body storage': '偵錯日誌請求體儲存',
    'Disk CSV': '磁碟 CSV',
    'Memory storage increases Redis usage.': '記憶體儲存會增加 Redis 使用量。',
    'Set all values to 0 to keep archives permanently.': '全部設為 0 代表永久保留封存。',
    'Store request and response bodies in memory or per-user CSV files.': '將請求和回應內容儲存在記憶體或依使用者分類的 CSV 檔案中。',
    PNG: 'PNG',
    JPEG: 'JPEG',
    WebP: 'WebP',
    'Available after saving': '儲存完成後可用',
    'Base64 copied to clipboard': 'Base64 已複製到剪貼簿',
    'Convert to Base64': '轉換為 Base64',
    'History works': '歷史作品',
    'Image to image': '圖生圖',
    Reuse: '重複使用',
    'Bindings remain disabled until protocol verification passes.':
      '繫結會保持停用，直到協定驗證通過。',
    'Last validated': '最近驗證時間',
    'Not validated': '尚未驗證',
    'Protocol verification failed': '協定驗證未通過',
    'Protocol verification passed': '協定驗證已通過',
    Processing: '處理中',
    Revalidate: '重新驗證',
    'The completion time of the latest protocol compatibility check.':
      '最近一次協定相容性驗證完成的時間。',
    'Text to image': '文生圖',
    'Unable to copy Base64': '無法複製 Base64',
    Succeeded: '已完成',
    Uploading: '儲存中',
    'Validate now': '立即驗證',
    Validated: '已驗證',
    Validating: '驗證中',
    'Validating...': '正在驗證…',
    'Validation failed': '驗證未通過',
    'Validation runs immediately in this request. It is not queued and does not contact the upstream provider.':
      '驗證會在目前請求中立即完成，不會進入佇列，也不會請求上游服務商。',
    Verified: '已驗證',
    'model name changed; validation required': '模型名稱已變更，需要重新驗證',
    'not validated': '尚未驗證',
    'validation in progress': '正在驗證',
    validated: '已驗證',
  },
  fr: {
    'Archive upload started.': 'Envoi de l’archive démarré.', 'Failed to start archive upload': 'Impossible de démarrer l’envoi de l’archive', 'Inflight CSV disk usage': 'Utilisation disque CSV en cours', 'Inflight CSV files': 'Fichiers CSV en cours', 'Pending archive uploads': 'Envois d’archives en attente', 'Upload pending archives': 'Envoyer les archives en attente',
    'Remove terminal inflight logs and completed CSV archives updated before the selected timestamp.': 'Supprime les journaux en cours terminés et les archives CSV finalisées avant l’heure sélectionnée.',
    'Inflight trace directory': 'Dossier des journaux en cours', 'Leave empty to use the disk cache directory': 'Laissez vide pour utiliser le dossier du cache disque', 'Stores disk-mode inflight trace CSV archives separately from request-body cache files.': 'Stocke séparément les archives CSV des journaux en cours et les fichiers cache des requêtes.',
    'Choose cache directory': 'Choisir le dossier de cache', 'Failed to load local cache files': 'Impossible de charger les fichiers du cache local', 'Local cache files': 'Fichiers du cache local', 'Manage CSV archives downloaded by this browser.': 'Gérez les archives CSV téléchargées par ce navigateur.', 'No local cache files.': 'Aucun fichier de cache local.',
    'Archive downloaded to local cache.': 'Archive téléchargée dans le cache local.', 'Download archives automatically in this browser': 'Télécharger automatiquement les archives dans ce navigateur', 'Download archive': 'Télécharger l’archive', 'Download this archive to your selected local cache directory?': 'Télécharger cette archive dans le dossier de cache local sélectionné ?', 'Failed to download archive': 'Échec du téléchargement de l’archive', 'Local cache is only supported in Chromium browsers.': 'Le cache local est pris en charge uniquement par les navigateurs Chromium.', 'This debug log is archived remotely. Download it locally to load request and response bodies.': 'Ce journal est archivé à distance. Téléchargez-le localement pour charger les contenus.',
    Years: 'Ans', Months: 'Mois', Days: 'Jours', Hours: 'Heures',
    'Archive retention': 'Conservation des archives',
    'Archive storage channel': 'Canal de stockage des archives',
    'Archive upload threshold (bytes)': 'Seuil d’envoi des archives (octets)',
    'Archive upload threshold (KB)': 'Seuil d’envoi des archives (Ko)',
    'Debug log body storage': 'Stockage du contenu des journaux de débogage',
    'Disk CSV': 'CSV sur disque',
    'Memory storage increases Redis usage.': 'Le stockage en mémoire augmente l’utilisation de Redis.',
    'Set all values to 0 to keep archives permanently.': 'Définissez toutes les valeurs à 0 pour conserver les archives définitivement.',
    'Store request and response bodies in memory or per-user CSV files.': 'Stockez les contenus des requêtes et réponses en mémoire ou dans des fichiers CSV par utilisateur.',
    PNG: 'PNG',
    JPEG: 'JPEG',
    WebP: 'WebP',
    'Size plan': 'Format d’image',
    'Download resolution': 'Résolution de téléchargement',
    '1:1 Square': 'Carré 1:1',
    Landscape: 'Paysage',
    Portrait: 'Portrait',
    'Choose the image composition and orientation.':
      'Choisissez le cadrage et l’orientation de l’image.',
    'Higher download resolutions are upscaled locally and do not affect generation billing.':
      'Les résolutions supérieures sont agrandies localement sans modifier la facturation de la génération.',
    Low: 'Basse',
    Medium: 'Moyenne',
    High: 'Élevée',
    Opaque: 'Opaque',
    Transparent: 'Transparent',
    'Choose the composition ratio for the SVG illustration.':
      'Choisissez le ratio de composition de l’illustration SVG.',
    '16:9 Landscape': 'Paysage 16:9',
    '9:16 Portrait': 'Portrait 9:16',
    '3:2 Landscape': 'Paysage 3:2',
    '2:3 Portrait': 'Portrait 2:3',
    'Custom ratio': 'Ratio personnalisé',
    'SVG stays vector-based; 2K and 4K downloads are rendered locally without extra generation charges.':
      'Le SVG reste vectoriel ; les téléchargements 2K et 4K sont rendus localement sans frais de génération supplémentaires.',
    'Unable to download image': 'Impossible de télécharger l’image',
    'Aspect ratio': 'Format d’image',
    'Available after saving': 'Disponible après l’enregistrement',
    'Base64 copied to clipboard': 'Base64 copié dans le presse-papiers',
    'Convert to Base64': 'Convertir en Base64',
    'History works': 'Historique des créations',
    'Image to image': 'Image vers image',
    Reuse: 'Réutiliser',
    'Bindings remain disabled until protocol verification passes.':
      'Les liaisons restent désactivées jusqu’à la réussite de la vérification du protocole.',
    'Last validated': 'Dernière vérification',
    'Not validated': 'Non vérifié',
    'Protocol verification failed': 'Échec de la vérification du protocole',
    'Protocol verification passed': 'Vérification du protocole réussie',
    Processing: 'Traitement en cours',
    Revalidate: 'Vérifier à nouveau',
    'The completion time of the latest protocol compatibility check.':
      'Heure de fin de la dernière vérification de compatibilité du protocole.',
    'Text to image': 'Texte vers image',
    'Unable to copy Base64': 'Impossible de copier Base64',
    Succeeded: 'Terminé',
    Uploading: 'Enregistrement en cours',
    'Validate now': 'Vérifier maintenant',
    Validated: 'Vérifié',
    Validating: 'Vérification en cours',
    'Validating...': 'Vérification en cours…',
    'Validation failed': 'Échec de la vérification',
    'Validation runs immediately in this request. It is not queued and does not contact the upstream provider.':
      'La vérification s’exécute immédiatement dans cette requête. Elle n’est pas mise en file d’attente et ne contacte pas le fournisseur en amont.',
    Verified: 'Vérifié',
    'model name changed; validation required':
      'le nom du modèle a changé ; une vérification est requise',
    'not validated': 'non vérifié',
    'validation in progress': 'vérification en cours',
    validated: 'validé',
  },
  ja: {
    'Archive upload started.': 'アーカイブのアップロードを開始しました。', 'Failed to start archive upload': 'アーカイブのアップロードを開始できませんでした', 'Inflight CSV disk usage': '進行中 CSV のディスク使用量', 'Inflight CSV files': '進行中 CSV ファイル', 'Pending archive uploads': '保留中のアーカイブアップロード', 'Upload pending archives': '保留中のアーカイブをアップロード',
    'Remove terminal inflight logs and completed CSV archives updated before the selected timestamp.': '選択した時刻より前に更新された完了済みの進行中ログと CSV アーカイブを削除します。',
    'Inflight trace directory': '進行中ログディレクトリ', 'Leave empty to use the disk cache directory': '空欄の場合はディスクキャッシュディレクトリを使用します', 'Stores disk-mode inflight trace CSV archives separately from request-body cache files.': 'ディスクモードの進行中ログ CSV アーカイブをリクエスト本文キャッシュとは別に保存します。',
    'Choose cache directory': 'キャッシュディレクトリを選択', 'Failed to load local cache files': 'ローカルキャッシュファイルを読み込めませんでした', 'Local cache files': 'ローカルキャッシュファイル', 'Manage CSV archives downloaded by this browser.': 'このブラウザーでダウンロードした CSV アーカイブを管理します。', 'No local cache files.': 'ローカルキャッシュファイルはありません。',
    'Archive downloaded to local cache.': 'アーカイブをローカルキャッシュにダウンロードしました。', 'Download archives automatically in this browser': 'このブラウザーでアーカイブを自動的にダウンロード', 'Download archive': 'アーカイブをダウンロード', 'Download this archive to your selected local cache directory?': 'このアーカイブを選択したローカルキャッシュディレクトリにダウンロードしますか？', 'Failed to download archive': 'アーカイブのダウンロードに失敗しました', 'Local cache is only supported in Chromium browsers.': 'ローカルキャッシュは Chromium ブラウザーでのみ利用できます。', 'This debug log is archived remotely. Download it locally to load request and response bodies.': 'このデバッグログはリモートに保存されています。本文を読み込むにはローカルにダウンロードしてください。',
    Years: '年', Months: 'か月', Days: '日', Hours: '時間',
    'Archive retention': 'アーカイブ保持期間',
    'Archive storage channel': 'アーカイブ保存チャネル',
    'Archive upload threshold (bytes)': 'アーカイブアップロードしきい値（バイト）',
    'Archive upload threshold (KB)': 'アーカイブアップロードしきい値（KB）',
    'Debug log body storage': 'デバッグログ本文の保存先',
    'Disk CSV': 'ディスク CSV',
    'Memory storage increases Redis usage.': 'メモリ保存は Redis の使用量を増やします。',
    'Set all values to 0 to keep archives permanently.': 'すべて 0 にするとアーカイブを無期限に保持します。',
    'Store request and response bodies in memory or per-user CSV files.': 'リクエストとレスポンスの本文をメモリまたはユーザー別 CSV ファイルに保存します。',
    PNG: 'PNG',
    JPEG: 'JPEG',
    WebP: 'WebP',
    'Size plan': 'サイズ設定',
    'Download resolution': 'ダウンロード解像度',
    '1:1 Square': '1:1 正方形',
    Landscape: '横長',
    Portrait: '縦長',
    'Choose the image composition and orientation.':
      '画像の構図比率と向きを選択します。',
    'Higher download resolutions are upscaled locally and do not affect generation billing.':
      '高いダウンロード解像度はローカルで拡大され、生成料金には影響しません。',
    Low: '低',
    Medium: '中',
    High: '高',
    Opaque: '不透明',
    Transparent: '透明',
    'Choose the composition ratio for the SVG illustration.':
      'SVG イラストの構図比率を選択します。',
    '16:9 Landscape': '16:9 横長',
    '9:16 Portrait': '9:16 縦長',
    '3:2 Landscape': '3:2 横長',
    '2:3 Portrait': '2:3 縦長',
    'Custom ratio': 'カスタム比率',
    'SVG stays vector-based; 2K and 4K downloads are rendered locally without extra generation charges.':
      'SVG はベクター形式のままです。2K と 4K はローカルで描画され、追加の生成料金はかかりません。',
    'Unable to download image': '画像をダウンロードできません',
    'Aspect ratio': 'アスペクト比',
    'Available after saving': '保存後に利用できます',
    'Base64 copied to clipboard': 'Base64 をクリップボードにコピーしました',
    'Convert to Base64': 'Base64 に変換',
    'History works': '作品履歴',
    'Image to image': '画像から画像',
    Reuse: '再利用',
    'Bindings remain disabled until protocol verification passes.':
      'プロトコル検証に成功するまで、バインディングは無効のままです。',
    'Last validated': '最終検証日時',
    'Not validated': '未検証',
    'Protocol verification failed': 'プロトコル検証に失敗しました',
    'Protocol verification passed': 'プロトコル検証に成功しました',
    Processing: '処理中',
    Revalidate: '再検証',
    'The completion time of the latest protocol compatibility check.':
      '直近のプロトコル互換性検証が完了した日時です。',
    'Text to image': 'テキストから画像',
    'Unable to copy Base64': 'Base64 をコピーできません',
    Succeeded: '完了',
    Uploading: '保存中',
    'Validate now': '今すぐ検証',
    Validated: '検証済み',
    Validating: '検証中',
    'Validating...': '検証中…',
    'Validation failed': '検証に失敗',
    'Validation runs immediately in this request. It is not queued and does not contact the upstream provider.':
      '検証はこのリクエスト内で直ちに実行されます。キューには入らず、上流プロバイダーにも接続しません。',
    Verified: '検証済み',
    'model name changed; validation required':
      'モデル名が変更されたため、検証が必要です',
    'not validated': '未検証',
    'validation in progress': '検証中',
    validated: '検証済み',
  },
  ru: {
    'Archive upload started.': 'Загрузка архива начата.', 'Failed to start archive upload': 'Не удалось начать загрузку архива', 'Inflight CSV disk usage': 'Использование диска CSV в процессе', 'Inflight CSV files': 'CSV-файлы в процессе', 'Pending archive uploads': 'Ожидающие загрузки архивов', 'Upload pending archives': 'Загрузить ожидающие архивы',
    'Remove terminal inflight logs and completed CSV archives updated before the selected timestamp.': 'Удаляет завершённые журналы в процессе и готовые CSV-архивы, обновлённые до выбранного времени.',
    'Inflight trace directory': 'Каталог журналов в процессе', 'Leave empty to use the disk cache directory': 'Оставьте пустым, чтобы использовать каталог дискового кэша', 'Stores disk-mode inflight trace CSV archives separately from request-body cache files.': 'Хранит CSV-архивы журналов в процессе отдельно от файлов кэша тел запросов.',
    'Choose cache directory': 'Выбрать каталог кэша', 'Failed to load local cache files': 'Не удалось загрузить файлы локального кэша', 'Local cache files': 'Файлы локального кэша', 'Manage CSV archives downloaded by this browser.': 'Управляйте CSV-архивами, загруженными этим браузером.', 'No local cache files.': 'Нет файлов локального кэша.',
    'Archive downloaded to local cache.': 'Архив загружен в локальный кэш.', 'Download archives automatically in this browser': 'Автоматически скачивать архивы в этом браузере', 'Download archive': 'Скачать архив', 'Download this archive to your selected local cache directory?': 'Скачать этот архив в выбранный каталог локального кэша?', 'Failed to download archive': 'Не удалось скачать архив', 'Local cache is only supported in Chromium browsers.': 'Локальный кэш поддерживается только браузерами Chromium.', 'This debug log is archived remotely. Download it locally to load request and response bodies.': 'Этот журнал архивирован удалённо. Загрузите его локально, чтобы просмотреть содержимое запроса и ответа.',
    Years: 'Годы', Months: 'Месяцы', Days: 'Дни', Hours: 'Часы',
    'Archive retention': 'Срок хранения архива',
    'Archive storage channel': 'Канал хранения архива',
    'Archive upload threshold (bytes)': 'Порог загрузки архива (байты)',
    'Archive upload threshold (KB)': 'Порог загрузки архива (КБ)',
    'Debug log body storage': 'Хранение содержимого отладочного журнала',
    'Disk CSV': 'CSV на диске',
    'Memory storage increases Redis usage.': 'Хранение в памяти увеличивает использование Redis.',
    'Set all values to 0 to keep archives permanently.': 'Установите все значения в 0 для бессрочного хранения архивов.',
    'Store request and response bodies in memory or per-user CSV files.': 'Сохраняйте содержимое запросов и ответов в памяти или в CSV-файлах пользователей.',
    PNG: 'PNG',
    JPEG: 'JPEG',
    WebP: 'WebP',
    'Size plan': 'Формат изображения',
    'Download resolution': 'Разрешение загрузки',
    '1:1 Square': 'Квадрат 1:1',
    Landscape: 'Альбомная',
    Portrait: 'Портретная',
    'Choose the image composition and orientation.':
      'Выберите пропорции и ориентацию изображения.',
    'Higher download resolutions are upscaled locally and do not affect generation billing.':
      'Более высокое разрешение создаётся локальным масштабированием и не влияет на стоимость генерации.',
    Low: 'Низкое',
    Medium: 'Среднее',
    High: 'Высокое',
    Opaque: 'Непрозрачный',
    Transparent: 'Прозрачный',
    'Choose the composition ratio for the SVG illustration.':
      'Выберите соотношение сторон для SVG-иллюстрации.',
    '16:9 Landscape': 'Альбомная 16:9',
    '9:16 Portrait': 'Портретная 9:16',
    '3:2 Landscape': 'Альбомная 3:2',
    '2:3 Portrait': 'Портретная 2:3',
    'Custom ratio': 'Свои пропорции',
    'SVG stays vector-based; 2K and 4K downloads are rendered locally without extra generation charges.':
      'SVG остаётся векторным; версии 2K и 4K создаются локально без дополнительной платы за генерацию.',
    'Unable to download image': 'Не удалось скачать изображение',
    'Aspect ratio': 'Соотношение сторон',
    'Available after saving': 'Доступно после сохранения',
    'Base64 copied to clipboard': 'Base64 скопирован в буфер обмена',
    'Convert to Base64': 'Преобразовать в Base64',
    'History works': 'История работ',
    'Image to image': 'Изображение в изображение',
    Reuse: 'Повторить',
    'Bindings remain disabled until protocol verification passes.':
      'Привязки остаются отключёнными, пока проверка протокола не будет пройдена.',
    'Last validated': 'Последняя проверка',
    'Not validated': 'Не проверено',
    'Protocol verification failed': 'Проверка протокола не пройдена',
    'Protocol verification passed': 'Проверка протокола пройдена',
    Processing: 'Обработка',
    Revalidate: 'Проверить снова',
    'The completion time of the latest protocol compatibility check.':
      'Время завершения последней проверки совместимости протокола.',
    'Text to image': 'Текст в изображение',
    'Unable to copy Base64': 'Не удалось скопировать Base64',
    Succeeded: 'Завершено',
    Uploading: 'Сохранение',
    'Validate now': 'Проверить сейчас',
    Validated: 'Проверено',
    Validating: 'Проверка',
    'Validating...': 'Проверка…',
    'Validation failed': 'Проверка не пройдена',
    'Validation runs immediately in this request. It is not queued and does not contact the upstream provider.':
      'Проверка выполняется сразу в этом запросе. Она не ставится в очередь и не обращается к вышестоящему провайдеру.',
    Verified: 'Проверено',
    'model name changed; validation required':
      'имя модели изменено; требуется проверка',
    'not validated': 'не проверено',
    'validation in progress': 'выполняется проверка',
    validated: 'проверено',
  },
  vi: {
    'Archive upload started.': 'Đã bắt đầu tải lên kho lưu trữ.', 'Failed to start archive upload': 'Không thể bắt đầu tải lên kho lưu trữ', 'Inflight CSV disk usage': 'Dung lượng đĩa CSV đang xử lý', 'Inflight CSV files': 'Tệp CSV đang xử lý', 'Pending archive uploads': 'Tải lên kho lưu trữ đang chờ', 'Upload pending archives': 'Tải lên kho lưu trữ đang chờ',
    'Remove terminal inflight logs and completed CSV archives updated before the selected timestamp.': 'Xóa nhật ký đang xử lý đã hoàn tất và kho lưu trữ CSV hoàn tất được cập nhật trước thời điểm đã chọn.',
    'Inflight trace directory': 'Thư mục nhật ký đang xử lý', 'Leave empty to use the disk cache directory': 'Để trống để dùng thư mục bộ nhớ đệm đĩa', 'Stores disk-mode inflight trace CSV archives separately from request-body cache files.': 'Lưu kho lưu trữ CSV nhật ký đang xử lý riêng với tệp bộ nhớ đệm nội dung yêu cầu.',
    'Choose cache directory': 'Chọn thư mục bộ nhớ đệm', 'Failed to load local cache files': 'Không thể tải tệp bộ nhớ đệm cục bộ', 'Local cache files': 'Tệp bộ nhớ đệm cục bộ', 'Manage CSV archives downloaded by this browser.': 'Quản lý các kho lưu trữ CSV được tải xuống bởi trình duyệt này.', 'No local cache files.': 'Không có tệp bộ nhớ đệm cục bộ.',
    'Archive downloaded to local cache.': 'Đã tải kho lưu trữ vào bộ nhớ đệm cục bộ.', 'Download archives automatically in this browser': 'Tự động tải kho lưu trữ trong trình duyệt này', 'Download archive': 'Tải kho lưu trữ', 'Download this archive to your selected local cache directory?': 'Tải kho lưu trữ này vào thư mục bộ nhớ đệm cục bộ đã chọn?', 'Failed to download archive': 'Không thể tải kho lưu trữ', 'Local cache is only supported in Chromium browsers.': 'Bộ nhớ đệm cục bộ chỉ được hỗ trợ trên trình duyệt Chromium.', 'This debug log is archived remotely. Download it locally to load request and response bodies.': 'Nhật ký này được lưu trữ từ xa. Hãy tải cục bộ để xem nội dung yêu cầu và phản hồi.',
    Years: 'Năm', Months: 'Tháng', Days: 'Ngày', Hours: 'Giờ',
    'Archive retention': 'Thời gian lưu trữ',
    'Archive storage channel': 'Kênh lưu trữ kho lưu trữ',
    'Archive upload threshold (bytes)': 'Ngưỡng tải lên kho lưu trữ (byte)',
    'Archive upload threshold (KB)': 'Ngưỡng tải lên kho lưu trữ (KB)',
    'Debug log body storage': 'Lưu nội dung nhật ký gỡ lỗi',
    'Disk CSV': 'CSV trên đĩa',
    'Memory storage increases Redis usage.': 'Lưu trong bộ nhớ làm tăng mức sử dụng Redis.',
    'Set all values to 0 to keep archives permanently.': 'Đặt tất cả giá trị thành 0 để lưu kho lưu trữ vĩnh viễn.',
    'Store request and response bodies in memory or per-user CSV files.': 'Lưu nội dung yêu cầu và phản hồi trong bộ nhớ hoặc tệp CSV theo người dùng.',
    PNG: 'PNG',
    JPEG: 'JPEG',
    WebP: 'WebP',
    'Size plan': 'Tỷ lệ ảnh',
    'Download resolution': 'Độ phân giải tải xuống',
    '1:1 Square': 'Vuông 1:1',
    Landscape: 'Ngang',
    Portrait: 'Dọc',
    'Choose the image composition and orientation.':
      'Chọn tỷ lệ bố cục và hướng ảnh.',
    'Higher download resolutions are upscaled locally and do not affect generation billing.':
      'Độ phân giải tải xuống cao hơn được phóng to cục bộ và không ảnh hưởng đến phí tạo ảnh.',
    Low: 'Thấp',
    Medium: 'Trung bình',
    High: 'Cao',
    Opaque: 'Không trong suốt',
    Transparent: 'Trong suốt',
    'Choose the composition ratio for the SVG illustration.':
      'Chọn tỷ lệ bố cục cho hình minh họa SVG.',
    '16:9 Landscape': 'Ngang 16:9',
    '9:16 Portrait': 'Dọc 9:16',
    '3:2 Landscape': 'Ngang 3:2',
    '2:3 Portrait': 'Dọc 2:3',
    'Custom ratio': 'Tỷ lệ tùy chỉnh',
    'SVG stays vector-based; 2K and 4K downloads are rendered locally without extra generation charges.':
      'SVG vẫn ở dạng vector; bản tải xuống 2K và 4K được kết xuất cục bộ, không phát sinh thêm phí tạo ảnh.',
    'Unable to download image': 'Không thể tải ảnh xuống',
    'Aspect ratio': 'Tỷ lệ khung hình',
    'Available after saving': 'Có sẵn sau khi lưu',
    'Base64 copied to clipboard': 'Đã sao chép Base64 vào bộ nhớ tạm',
    'Convert to Base64': 'Chuyển thành Base64',
    'History works': 'Lịch sử tác phẩm',
    'Image to image': 'Ảnh sang ảnh',
    Reuse: 'Dùng lại',
    'Bindings remain disabled until protocol verification passes.':
      'Liên kết sẽ vẫn bị tắt cho đến khi xác minh giao thức thành công.',
    'Last validated': 'Lần xác minh gần nhất',
    'Not validated': 'Chưa xác minh',
    'Protocol verification failed': 'Xác minh giao thức không thành công',
    'Protocol verification passed': 'Xác minh giao thức thành công',
    Processing: 'Đang xử lý',
    Revalidate: 'Xác minh lại',
    'The completion time of the latest protocol compatibility check.':
      'Thời điểm hoàn tất lần kiểm tra tương thích giao thức gần nhất.',
    'Text to image': 'Văn bản sang ảnh',
    'Unable to copy Base64': 'Không thể sao chép Base64',
    Succeeded: 'Hoàn tất',
    Uploading: 'Đang lưu',
    'Validate now': 'Xác minh ngay',
    Validated: 'Đã xác minh',
    Validating: 'Đang xác minh',
    'Validating...': 'Đang xác minh…',
    'Validation failed': 'Xác minh không thành công',
    'Validation runs immediately in this request. It is not queued and does not contact the upstream provider.':
      'Việc xác minh chạy ngay trong yêu cầu này. Nó không được xếp hàng và không liên hệ nhà cung cấp thượng nguồn.',
    Verified: 'Đã xác minh',
    'model name changed; validation required':
      'tên mô hình đã thay đổi; cần xác minh',
    'not validated': 'chưa xác minh',
    'validation in progress': 'đang xác minh',
    validated: 'đã xác minh',
  },
}

const validationMessages = {
  en: {
    'advanced custom channel has no matching image route':
      'advanced custom channel has no matching image route',
    'advanced custom image protocol requires an Advanced Custom channel':
      'advanced custom image protocol requires an Advanced Custom channel',
    'binding model snapshot is stale': 'binding model snapshot is stale',
    'capability is unavailable': 'capability is unavailable',
    'channel has no enabled matching ability':
      'channel has no enabled matching ability',
    'channel is unavailable': 'channel is unavailable',
    'creative storage is unavailable': 'creative storage is unavailable',
    'Midjourney protocol requires a Midjourney channel':
      'Midjourney protocol requires a Midjourney channel',
    'model is unavailable': 'model is unavailable',
    'protocol does not match the image capability contract':
      'protocol does not match the image capability contract',
    'publication group is unavailable': 'publication group is unavailable',
    'publication is unavailable': 'publication is unavailable',
  },
  zh: {
    'advanced custom channel has no matching image route':
      '高级自定义渠道没有匹配的图片路由',
    'advanced custom image protocol requires an Advanced Custom channel':
      '高级自定义图片协议需要高级自定义渠道',
    'binding model snapshot is stale': '绑定的模型快照已过期',
    'capability is unavailable': '能力不可用',
    'channel has no enabled matching ability': '渠道没有已启用的匹配能力',
    'channel is unavailable': '渠道不可用',
    'creative storage is unavailable': '创作台存储不可用',
    'Midjourney protocol requires a Midjourney channel':
      'Midjourney 协议需要 Midjourney 渠道',
    'model is unavailable': '模型不可用',
    'protocol does not match the image capability contract':
      '协议不符合图片能力约定',
    'publication group is unavailable': '发布分组不可用',
    'publication is unavailable': '分组发布不可用',
  },
  'zh-TW': {
    'advanced custom channel has no matching image route':
      '進階自訂渠道沒有符合的圖片路由',
    'advanced custom image protocol requires an Advanced Custom channel':
      '進階自訂圖片協定需要進階自訂渠道',
    'binding model snapshot is stale': '繫結的模型快照已過期',
    'capability is unavailable': '能力不可用',
    'channel has no enabled matching ability': '渠道沒有已啟用的相符能力',
    'channel is unavailable': '渠道不可用',
    'creative storage is unavailable': 'Creative Studio 儲存空間不可用',
    'Midjourney protocol requires a Midjourney channel':
      'Midjourney 協定需要 Midjourney 渠道',
    'model is unavailable': '模型不可用',
    'protocol does not match the image capability contract':
      '協定不符合圖片能力合約',
    'publication group is unavailable': '發布群組不可用',
    'publication is unavailable': '群組發布不可用',
  },
  fr: {
    'advanced custom channel has no matching image route':
      'le canal Advanced Custom n’a pas de route d’image correspondante',
    'advanced custom image protocol requires an Advanced Custom channel':
      'le protocole d’image Advanced Custom nécessite un canal Advanced Custom',
    'binding model snapshot is stale':
      'l’instantané du modèle de la liaison est obsolète',
    'capability is unavailable': 'la capacité est indisponible',
    'channel has no enabled matching ability':
      'le canal n’a aucune capacité correspondante activée',
    'channel is unavailable': 'le canal est indisponible',
    'creative storage is unavailable':
      'le stockage du Studio créatif est indisponible',
    'Midjourney protocol requires a Midjourney channel':
      'le protocole Midjourney requiert un canal Midjourney',
    'model is unavailable': 'le modèle est indisponible',
    'protocol does not match the image capability contract':
      'le protocole ne correspond pas au contrat de capacité d’image',
    'publication group is unavailable':
      'le groupe de publication est indisponible',
    'publication is unavailable': 'la publication est indisponible',
  },
  ja: {
    'advanced custom channel has no matching image route':
      'Advanced Custom チャネルに一致する画像ルートがありません',
    'advanced custom image protocol requires an Advanced Custom channel':
      'Advanced Custom 画像プロトコルには Advanced Custom チャネルが必要です',
    'binding model snapshot is stale':
      'バインディングのモデルスナップショットが古くなっています',
    'capability is unavailable': '機能を利用できません',
    'channel has no enabled matching ability':
      'チャネルに有効な一致機能がありません',
    'channel is unavailable': 'チャネルを利用できません',
    'creative storage is unavailable':
      'クリエイティブスタジオのストレージを利用できません',
    'Midjourney protocol requires a Midjourney channel':
      'Midjourney プロトコルには Midjourney チャネルが必要です',
    'model is unavailable': 'モデルを利用できません',
    'protocol does not match the image capability contract':
      'プロトコルが画像機能の契約と一致しません',
    'publication group is unavailable': '公開グループを利用できません',
    'publication is unavailable': '公開を利用できません',
  },
  ru: {
    'advanced custom channel has no matching image route':
      'в канале Advanced Custom нет подходящего маршрута изображений',
    'advanced custom image protocol requires an Advanced Custom channel':
      'протокол изображений Advanced Custom требует канал Advanced Custom',
    'binding model snapshot is stale': 'снимок модели привязки устарел',
    'capability is unavailable': 'возможность недоступна',
    'channel has no enabled matching ability':
      'в канале нет включённой подходящей возможности',
    'channel is unavailable': 'канал недоступен',
    'creative storage is unavailable': 'хранилище творческой студии недоступно',
    'Midjourney protocol requires a Midjourney channel':
      'протокол Midjourney требует канал Midjourney',
    'model is unavailable': 'модель недоступна',
    'protocol does not match the image capability contract':
      'протокол не соответствует контракту возможности изображения',
    'publication group is unavailable': 'группа публикации недоступна',
    'publication is unavailable': 'публикация недоступна',
  },
  vi: {
    'advanced custom channel has no matching image route':
      'kênh Advanced Custom không có tuyến ảnh phù hợp',
    'advanced custom image protocol requires an Advanced Custom channel':
      'giao thức ảnh Advanced Custom yêu cầu kênh Advanced Custom',
    'binding model snapshot is stale': 'bản chụp mô hình của liên kết đã cũ',
    'capability is unavailable': 'khả năng không khả dụng',
    'channel has no enabled matching ability':
      'kênh không có khả năng phù hợp đang bật',
    'channel is unavailable': 'kênh không khả dụng',
    'creative storage is unavailable': 'bộ nhớ Studio sáng tạo không khả dụng',
    'Midjourney protocol requires a Midjourney channel':
      'giao thức Midjourney yêu cầu kênh Midjourney',
    'model is unavailable': 'mô hình không khả dụng',
    'protocol does not match the image capability contract':
      'giao thức không khớp với hợp đồng khả năng ảnh',
    'publication group is unavailable': 'nhóm phát hành không khả dụng',
    'publication is unavailable': 'bản phát hành không khả dụng',
  },
}

const creativeStudioUi = {
  en: {
    'Size plan': 'Size plan',
    'Download resolution': 'Download resolution',
    '1:1 Square': '1:1 Square',
    Landscape: 'Landscape',
    Portrait: 'Portrait',
    'Choose the image composition and orientation.':
      'Choose the image composition and orientation.',
    'Higher download resolutions are upscaled locally and do not affect generation billing.':
      'Higher download resolutions are upscaled locally and do not affect generation billing.',
    Low: 'Low',
    Medium: 'Medium',
    High: 'High',
    Opaque: 'Opaque',
    Transparent: 'Transparent',
    'Choose the composition ratio for the SVG illustration.':
      'Choose the composition ratio for the SVG illustration.',
    '16:9 Landscape': '16:9 Landscape',
    '9:16 Portrait': '9:16 Portrait',
    '3:2 Landscape': '3:2 Landscape',
    '2:3 Portrait': '2:3 Portrait',
    'Custom ratio': 'Custom ratio',
    'SVG stays vector-based; 2K and 4K downloads are rendered locally without extra generation charges.':
      'SVG stays vector-based; 2K and 4K downloads are rendered locally without extra generation charges.',
    'Unable to download image': 'Unable to download image',
    'Aspect ratio': 'Aspect ratio',
    'A Shiba Inu wearing sunglasses, flat illustration style, vibrant colors':
      'A Shiba Inu wearing sunglasses, flat illustration style, vibrant colors',
    'Cyberpunk city nightscape, neon reflections, cinematic lighting, 8k':
      'Cyberpunk city nightscape, neon reflections, cinematic lighting, 8k',
    'Example: a cyberpunk hacker cat coding beneath neon lights, cinematic lighting, highly detailed':
      'Example: a cyberpunk hacker cat coding beneath neon lights, cinematic lighting, highly detailed',
    'Failed to create task': 'Failed to create task',
    'Generation results': 'Generation results',
    'Minimalist product photography: a floating perfume bottle, soft light, beige background':
      'Minimalist product photography: a floating perfume bottle, soft light, beige background',
    'Open in new window': 'Open in new window',
    'Unable to upload reference image': 'Unable to upload reference image',
    'Unable to add reference image': 'Unable to add reference image',
    'Reference image added': 'Reference image added',
    'Adding reference image': 'Adding reference image',
    'Added. Closing in {{count}} seconds':
      'Added. Closing in {{count}} seconds',
    'Previous image': 'Previous image',
    'Next image': 'Next image',
    'No image-to-image model available': 'No image-to-image model available',
    'Current image-to-image reference image limit has been reached':
      'Current image-to-image reference image limit has been reached',
    'Upload reference image': 'Upload reference image',
    'Watercolor Jiangnan town at dawn, thin mist, leave blank space':
      'Watercolor Jiangnan town at dawn, thin mist, leave blank space',
    'Are you sure you want to delete this task? This action cannot be undone.':
      'Are you sure you want to delete this task? This action cannot be undone.',
    'Delete task?': 'Delete task?',
    'Delete selected': 'Delete selected',
    'Download selected': 'Download selected',
    'Failed to delete task': 'Failed to delete task',
    'Failed to retry task': 'Failed to retry task',
    'Invert selection': 'Invert selection',
    Background: 'Background',
    Moderation: 'Moderation',
    Quality: 'Quality',
    Redo: 'Redo',
    'Retry started': 'Retry started',
    'Rotate left': 'Rotate left',
    'Rotate right': 'Rotate right',
    'Zoom in': 'Zoom in',
    'Zoom out': 'Zoom out',
    Size: 'Size',
    'Input images': 'Input images',
    'Transparent background': 'Transparent background',
  },
  zh: {
    'Size plan': '尺寸方案',
    'Download resolution': '下载分辨率',
    '1:1 Square': '1:1 方图',
    Landscape: '横图',
    Portrait: '竖图',
    'Choose the image composition and orientation.': '选择图片构图比例和方向。',
    'Higher download resolutions are upscaled locally and do not affect generation billing.':
      '更高下载分辨率会在本地高清放大，不影响生成计费。',
    Low: '低',
    Medium: '中',
    High: '高',
    Opaque: '不透明',
    Transparent: '透明',
    'Choose the composition ratio for the SVG illustration.':
      '选择 SVG 插画的构图比例。',
    '16:9 Landscape': '16:9 横图',
    '9:16 Portrait': '9:16 竖图',
    '3:2 Landscape': '3:2 横图',
    '2:3 Portrait': '2:3 竖图',
    'Custom ratio': '自定义比例',
    'SVG stays vector-based; 2K and 4K downloads are rendered locally without extra generation charges.':
      'SVG 保持矢量格式；2K 和 4K 下载在本地渲染，不额外产生生成费用。',
    'Unable to download image': '无法下载图片',
    'Aspect ratio': '宽高比',
    'A Shiba Inu wearing sunglasses, flat illustration style, vibrant colors':
      '一只戴墨镜的柴犬，扁平插画风，明快配色',
    'Cyberpunk city nightscape, neon reflections, cinematic lighting, 8k':
      '赛博朋克城市夜景，霓虹倒影，电影级光影，8K',
    'Example: a cyberpunk hacker cat coding beneath neon lights, cinematic lighting, highly detailed':
      '例如：霓虹灯下敲代码的赛博朋克黑客猫，电影感光影，超精细',
    'Failed to create task': '创建任务失败',
    'Generation results': '生成结果',
    'Minimalist product photography: a floating perfume bottle, soft light, beige background':
      '极简产品摄影：悬浮的香水瓶，柔光，米色背景',
    'Open in new window': '新窗口打开',
    'Unable to upload reference image': '无法上传参考图片',
    'Unable to add reference image': '无法添加参考图片',
    'Reference image added': '已添加参考图片',
    'Adding reference image': '正在添加参考图片',
    'Added. Closing in {{count}} seconds': '已添加，将在 {{count}} 秒后关闭',
    'Previous image': '上一张图片',
    'Next image': '下一张图片',
    'No image-to-image model available': '暂无可用的图生图模型',
    'Current image-to-image reference image limit has been reached':
      '当前图生图参考图数量已满，无法加入图生图',
    'Upload reference image': '上传参考图片',
    'Watercolor Jiangnan town at dawn, thin mist, leave blank space':
      '水彩风格的江南古镇清晨，薄雾，留白',
    'Are you sure you want to delete this task? This action cannot be undone.':
      '确定要删除此任务吗？此操作无法撤销。',
    'Delete task?': '删除任务？',
    'Delete selected': '删除所选',
    'Download selected': '下载所选',
    'Failed to delete task': '删除任务失败',
    'Failed to retry task': '重做任务失败',
    'Invert selection': '反选',
    Background: '背景',
    Moderation: '审核',
    Quality: '质量',
    Redo: '重做',
    'Retry started': '已开始重做',
    'Rotate left': '向左旋转',
    'Rotate right': '向右旋转',
    'Zoom in': '放大',
    'Zoom out': '缩小',
    Size: '尺寸',
    'Input images': '输入图片',
    'Transparent background': '透明背景',
  },
  'zh-TW': {
    'Size plan': '尺寸方案',
    'Download resolution': '下載解析度',
    '1:1 Square': '1:1 方圖',
    Landscape: '橫圖',
    Portrait: '直圖',
    'Choose the image composition and orientation.': '選擇圖片構圖比例和方向。',
    'Higher download resolutions are upscaled locally and do not affect generation billing.':
      '更高下載解析度會在本機高清放大，不影響生成計費。',
    Low: '低',
    Medium: '中',
    High: '高',
    Opaque: '不透明',
    Transparent: '透明',
    'Choose the composition ratio for the SVG illustration.':
      '選擇 SVG 插畫的構圖比例。',
    '16:9 Landscape': '16:9 橫圖',
    '9:16 Portrait': '9:16 直圖',
    '3:2 Landscape': '3:2 橫圖',
    '2:3 Portrait': '2:3 直圖',
    'Custom ratio': '自訂比例',
    'SVG stays vector-based; 2K and 4K downloads are rendered locally without extra generation charges.':
      'SVG 保持向量格式；2K 和 4K 下載在本機渲染，不額外產生生成費用。',
    'Unable to download image': '無法下載圖片',
    'Unable to add reference image': '無法加入參考圖片',
    'Reference image added': '已加入參考圖片',
    'Adding reference image': '正在加入參考圖片',
    'Added. Closing in {{count}} seconds': '已加入，將在 {{count}} 秒後關閉',
    'Previous image': '上一張圖片',
    'Next image': '下一張圖片',
    'No image-to-image model available': '暫無可用的圖生圖模型',
    'Current image-to-image reference image limit has been reached':
      '目前圖生圖參考圖數量已滿，無法加入圖生圖',
    'Aspect ratio': '寬高比',
    'Are you sure you want to delete this task? This action cannot be undone.':
      '確定要刪除此工作嗎？此操作無法復原。',
    'Delete task?': '刪除工作？',
    'Delete selected': '刪除所選',
    'Download selected': '下載所選',
    'Failed to delete task': '刪除工作失敗',
    'Failed to retry task': '重新執行工作失敗',
    'Invert selection': '反選',
    Background: '背景',
    Moderation: '審核',
    Quality: '品質',
    Redo: '重新執行',
    'Retry started': '已開始重新執行',
    'Rotate left': '向左旋轉',
    'Rotate right': '向右旋轉',
    'Zoom in': '放大',
    'Zoom out': '縮小',
    Size: '尺寸',
    'Input images': '輸入圖片',
    'Transparent background': '透明背景',
  },
  fr: {
    'A Shiba Inu wearing sunglasses, flat illustration style, vibrant colors':
      'Un Shiba Inu avec des lunettes de soleil, illustration plate, couleurs vives',
    'Cyberpunk city nightscape, neon reflections, cinematic lighting, 8k':
      'Paysage urbain cyberpunk nocturne, reflets néon, éclairage cinématographique, 8K',
    'Example: a cyberpunk hacker cat coding beneath neon lights, cinematic lighting, highly detailed':
      'Exemple : un chat hacker cyberpunk codant sous des néons, éclairage cinématographique, très détaillé',
    'Failed to create task': 'Échec de la création de la tâche',
    'Generation results': 'Résultats de génération',
    'Minimalist product photography: a floating perfume bottle, soft light, beige background':
      'Photo produit minimaliste : flacon de parfum flottant, lumière douce, fond beige',
    'Open in new window': 'Ouvrir dans une nouvelle fenêtre',
    'Unable to upload reference image':
      'Impossible d’importer l’image de référence',
    'Unable to add reference image':
      'Impossible d’ajouter l’image de référence',
    'Reference image added': 'Image de référence ajoutée',
    'Adding reference image': 'Ajout de l’image de référence',
    'Added. Closing in {{count}} seconds':
      'Ajoutée. Fermeture dans {{count}} secondes',
    'Previous image': 'Image précédente',
    'Next image': 'Image suivante',
    'No image-to-image model available':
      'Aucun modèle image vers image n’est disponible',
    'Current image-to-image reference image limit has been reached':
      'La limite d’images de référence est atteinte',
    'Upload reference image': 'Importer une image de référence',
    'Watercolor Jiangnan town at dawn, thin mist, leave blank space':
      'Village de Jiangnan à l’aube, aquarelle, brume légère, espace vide',
    'Are you sure you want to delete this task? This action cannot be undone.':
      'Voulez-vous vraiment supprimer cette tâche ? Cette action est irréversible.',
    'Delete task?': 'Supprimer la tâche ?',
    'Delete selected': 'Supprimer la sélection',
    'Download selected': 'Télécharger la sélection',
    'Failed to delete task': 'Échec de la suppression de la tâche',
    'Failed to retry task': 'Échec de la relance de la tâche',
    'Invert selection': 'Inverser la sélection',
    Background: 'Arrière-plan',
    Moderation: 'Modération',
    Quality: 'Qualité',
    Redo: 'Relancer',
    'Retry started': 'Relance commencée',
    'Rotate left': 'Pivoter à gauche',
    'Rotate right': 'Pivoter à droite',
    'Zoom in': 'Agrandir',
    'Zoom out': 'Réduire',
    Size: 'Taille',
    'Input images': 'Images d’entrée',
    'Transparent background': 'Arrière-plan transparent',
  },
  ja: {
    'A Shiba Inu wearing sunglasses, flat illustration style, vibrant colors':
      'サングラスをかけた柴犬、フラットイラスト、鮮やかな配色',
    'Cyberpunk city nightscape, neon reflections, cinematic lighting, 8k':
      'サイバーパンク都市の夜景、ネオンの反射、映画的な照明、8K',
    'Example: a cyberpunk hacker cat coding beneath neon lights, cinematic lighting, highly detailed':
      '例：ネオンの下でコーディングするサイバーパンクのハッカー猫、映画的な照明、超高精細',
    'Failed to create task': 'タスクを作成できませんでした',
    'Generation results': '生成結果',
    'Minimalist product photography: a floating perfume bottle, soft light, beige background':
      'ミニマルな商品写真：浮遊する香水瓶、柔らかな光、ベージュの背景',
    'Open in new window': '新しいウィンドウで開く',
    'Unable to upload reference image': '参照画像をアップロードできません',
    'Unable to add reference image': '参照画像を追加できません',
    'Reference image added': '参照画像を追加しました',
    'Adding reference image': '参照画像を追加中',
    'Added. Closing in {{count}} seconds': '{{count}} 秒後に閉じます',
    'Previous image': '前の画像',
    'Next image': '次の画像',
    'No image-to-image model available':
      '利用可能な画像から画像モデルがありません',
    'Current image-to-image reference image limit has been reached':
      '参照画像の上限に達しました',
    'Upload reference image': '参照画像をアップロード',
    'Watercolor Jiangnan town at dawn, thin mist, leave blank space':
      '水彩画風の江南古鎮の朝、薄霧、余白',
    'Are you sure you want to delete this task? This action cannot be undone.':
      'このタスクを削除しますか？この操作は元に戻せません。',
    'Delete task?': 'タスクを削除しますか？',
    'Delete selected': '選択項目を削除',
    'Download selected': '選択項目をダウンロード',
    'Failed to delete task': 'タスクを削除できませんでした',
    'Failed to retry task': 'タスクを再実行できませんでした',
    'Invert selection': '選択を反転',
    Background: '背景',
    Moderation: 'モデレーション',
    Quality: '品質',
    Redo: '再実行',
    'Retry started': '再実行を開始しました',
    'Rotate left': '左に回転',
    'Rotate right': '右に回転',
    'Zoom in': '拡大',
    'Zoom out': '縮小',
    Size: 'サイズ',
    'Input images': '入力画像',
    'Transparent background': '透明背景',
  },
  ru: {
    'A Shiba Inu wearing sunglasses, flat illustration style, vibrant colors':
      'Сиба-ину в солнечных очках, плоская иллюстрация, яркие цвета',
    'Cyberpunk city nightscape, neon reflections, cinematic lighting, 8k':
      'Ночной киберпанк-город, неоновые отражения, кинематографичный свет, 8K',
    'Example: a cyberpunk hacker cat coding beneath neon lights, cinematic lighting, highly detailed':
      'Пример: киберпанк-кот-хакер пишет код под неоном, кинематографичный свет, высокая детализация',
    'Failed to create task': 'Не удалось создать задачу',
    'Generation results': 'Результаты генерации',
    'Minimalist product photography: a floating perfume bottle, soft light, beige background':
      'Минималистичная предметная съёмка: парящий флакон духов, мягкий свет, бежевый фон',
    'Open in new window': 'Открыть в новом окне',
    'Unable to upload reference image':
      'Не удалось загрузить эталонное изображение',
    'Unable to add reference image':
      'Не удалось добавить эталонное изображение',
    'Reference image added': 'Эталонное изображение добавлено',
    'Adding reference image': 'Добавление эталонного изображения',
    'Added. Closing in {{count}} seconds':
      'Добавлено. Закрытие через {{count}} с',
    'Previous image': 'Предыдущее изображение',
    'Next image': 'Следующее изображение',
    'No image-to-image model available':
      'Нет доступной модели преобразования изображения',
    'Current image-to-image reference image limit has been reached':
      'Достигнут лимит эталонных изображений',
    'Upload reference image': 'Загрузить эталонное изображение',
    'Watercolor Jiangnan town at dawn, thin mist, leave blank space':
      'Акварельный городок Цзяннань на рассвете, лёгкий туман, свободное пространство',
    'Are you sure you want to delete this task? This action cannot be undone.':
      'Удалить эту задачу? Это действие нельзя отменить.',
    'Delete task?': 'Удалить задачу?',
    'Delete selected': 'Удалить выбранное',
    'Download selected': 'Скачать выбранное',
    'Failed to delete task': 'Не удалось удалить задачу',
    'Failed to retry task': 'Не удалось повторить задачу',
    'Invert selection': 'Инвертировать выбор',
    Background: 'Фон',
    Moderation: 'Модерация',
    Quality: 'Качество',
    Redo: 'Повторить',
    'Retry started': 'Повторный запуск начат',
    'Rotate left': 'Повернуть влево',
    'Rotate right': 'Повернуть вправо',
    'Zoom in': 'Увеличить',
    'Zoom out': 'Уменьшить',
    Size: 'Размер',
    'Input images': 'Входные изображения',
    'Transparent background': 'Прозрачный фон',
  },
  vi: {
    'A Shiba Inu wearing sunglasses, flat illustration style, vibrant colors':
      'Chó Shiba đeo kính râm, minh họa phẳng, màu sắc rực rỡ',
    'Cyberpunk city nightscape, neon reflections, cinematic lighting, 8k':
      'Cảnh đêm thành phố cyberpunk, phản chiếu neon, ánh sáng điện ảnh, 8K',
    'Example: a cyberpunk hacker cat coding beneath neon lights, cinematic lighting, highly detailed':
      'Ví dụ: mèo hacker cyberpunk lập trình dưới đèn neon, ánh sáng điện ảnh, cực kỳ chi tiết',
    'Failed to create task': 'Không thể tạo tác vụ',
    'Generation results': 'Kết quả tạo ảnh',
    'Minimalist product photography: a floating perfume bottle, soft light, beige background':
      'Ảnh sản phẩm tối giản: chai nước hoa lơ lửng, ánh sáng mềm, nền be',
    'Open in new window': 'Mở trong cửa sổ mới',
    'Unable to upload reference image': 'Không thể tải ảnh tham chiếu lên',
    'Unable to add reference image': 'Không thể thêm ảnh tham chiếu',
    'Reference image added': 'Đã thêm ảnh tham chiếu',
    'Adding reference image': 'Đang thêm ảnh tham chiếu',
    'Added. Closing in {{count}} seconds': 'Đã thêm. Đóng sau {{count}} giây',
    'Previous image': 'Ảnh trước',
    'Next image': 'Ảnh tiếp theo',
    'No image-to-image model available':
      'Không có mô hình ảnh sang ảnh khả dụng',
    'Current image-to-image reference image limit has been reached':
      'Đã đạt giới hạn ảnh tham chiếu',
    'Upload reference image': 'Tải ảnh tham chiếu lên',
    'Watercolor Jiangnan town at dawn, thin mist, leave blank space':
      'Thị trấn Giang Nam lúc bình minh phong cách màu nước, sương mỏng, khoảng trống',
    'Are you sure you want to delete this task? This action cannot be undone.':
      'Bạn có chắc muốn xóa tác vụ này không? Hành động này không thể hoàn tác.',
    'Delete task?': 'Xóa tác vụ?',
    'Delete selected': 'Xóa mục đã chọn',
    'Download selected': 'Tải mục đã chọn',
    'Failed to delete task': 'Không thể xóa tác vụ',
    'Failed to retry task': 'Không thể chạy lại tác vụ',
    'Invert selection': 'Đảo lựa chọn',
    Background: 'Nền',
    Moderation: 'Kiểm duyệt',
    Quality: 'Chất lượng',
    Redo: 'Làm lại',
    'Retry started': 'Đã bắt đầu chạy lại',
    'Rotate left': 'Xoay trái',
    'Rotate right': 'Xoay phải',
    'Zoom in': 'Phóng to',
    'Zoom out': 'Thu nhỏ',
    Size: 'Kích thước',
    'Input images': 'Ảnh đầu vào',
    'Transparent background': 'Nền trong suốt',
  },
}

for (const [locale, translations] of Object.entries(creativeStudioUi)) {
  Object.assign(newKeys[locale], translations)
}

for (const [locale, translations] of Object.entries(validationMessages)) {
  Object.assign(newKeys[locale], translations)
}

async function main() {
  for (const [locale, translations] of Object.entries(newKeys)) {
    const filePath = path.join(localesDir, `${locale}.json`)
    const content = JSON.parse(await fs.readFile(filePath, 'utf8'))
    Object.assign(content.translation, translations)
    content.translation = Object.fromEntries(
      Object.entries(content.translation).sort(([left], [right]) =>
        left.localeCompare(right)
      )
    )
    await fs.writeFile(filePath, `${JSON.stringify(content, null, 2)}\n`)
  }
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
