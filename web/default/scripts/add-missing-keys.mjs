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

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
  return JSON.stringify(obj, null, 2) + '\n'
}

const newKeys = {
  en: {
    'Allowed range: {{min}}-{{max}} minutes (up to 7 days)':
      'Allowed range: {{min}}-{{max}} minutes (up to 7 days)',
    'Choose a fixed interval or a cron schedule for cleanup scans.':
      'Choose a fixed interval or a cron schedule for cleanup scans.',
    'Cron schedule': 'Cron schedule',
    'Every N hours': 'Every N hours',
    'Execution times use the server local timezone. Preview below follows the server clock.':
      'Execution times use the server local timezone. Preview below follows the server clock.',
    'Fixed interval': 'Fixed interval',
    'Friday': 'Friday',
    'Generated cron expression': 'Generated cron expression',
    'Hour interval': 'Hour interval',
    'Inflight log cleanup interval must be between 1 and 10080 minutes':
      'Inflight log cleanup interval must be between 1 and 10080 minutes',
    'Inflight log cleanup schedule mode': 'Inflight log cleanup schedule mode',
    'Invalid cron expression': 'Invalid cron expression',
    'Monday': 'Monday',
    'Next 6 scheduled runs': 'Next 6 scheduled runs',
    'Saturday': 'Saturday',
    'Schedule preset': 'Schedule preset',
    'Server timezone: {{timezone}}': 'Server timezone: {{timezone}}',
    'Start live refresh': 'Start live refresh',
    'Stop live refresh': 'Stop live refresh',
    'Sunday': 'Sunday',
    'Thursday': 'Thursday',
    'Tuesday': 'Tuesday',
    'Wednesday': 'Wednesday',
    'Weekday': 'Weekday',
    'Automatic inflight log cleanup is disabled':
      'Automatic inflight log cleanup is disabled',
    'Clean terminal inflight logs every {{minutes}} minutes':
      'Clean terminal inflight logs every {{minutes}} minutes',
    'Clean terminal inflight logs on schedule: {{expr}}':
      'Clean terminal inflight logs on schedule: {{expr}}',
    'Cleanup in progress': 'Cleanup in progress',
    'Loading cleanup schedule...': 'Loading cleanup schedule...',
    'Next cleanup: {{time}}': 'Next cleanup: {{time}}',
    'Async Logs': 'Async Logs',
  },
  zh: {
    'Allowed range: {{min}}-{{max}} minutes (up to 7 days)':
      '可设置范围：{{min}}–{{max}} 分钟（最长 7 天）',
    'Choose a fixed interval or a cron schedule for cleanup scans.':
      '选择固定间隔或 CRON 计划来扫描清理终态在途日志。',
    'Cron schedule': 'CRON 计划',
    'Every N hours': '每 N 小时',
    'Execution times use the server local timezone. Preview below follows the server clock.':
      '执行时间以服务器本地时区为准，下方预览基于服务器时钟计算。',
    'Fixed interval': '固定间隔',
    'Friday': '周五',
    'Generated cron expression': '生成的 CRON 表达式',
    'Hour interval': '小时间隔',
    'Inflight log cleanup interval must be between 1 and 10080 minutes':
      '在途日志清理间隔必须在 1–10080 分钟之间',
    'Inflight log cleanup schedule mode': '在途日志清理调度方式',
    'Invalid cron expression': 'CRON 表达式无效',
    'Monday': '周一',
    'Next 6 scheduled runs': '未来 6 次计划执行时间',
    'Saturday': '周六',
    'Schedule preset': '计划预设',
    'Server timezone: {{timezone}}': '服务器时区：{{timezone}}',
    'Start live refresh': '开始实时刷新',
    'Stop live refresh': '停止实时刷新',
    'Sunday': '周日',
    'Thursday': '周四',
    'Tuesday': '周二',
    'Wednesday': '周三',
    'Weekday': '星期',
    'Automatic inflight log cleanup is disabled': '在途日志自动清理已关闭',
    'Clean terminal inflight logs every {{minutes}} minutes':
      '每 {{minutes}} 分钟清理终态在途日志',
    'Clean terminal inflight logs on schedule: {{expr}}':
      '按计划清理终态在途日志：{{expr}}',
    'Cleanup in progress': '清理进行中',
    'Loading cleanup schedule...': '正在加载清理计划...',
    'Next cleanup: {{time}}': '下次清理：{{time}}',
    'Async Logs': '异步日志',
  },
  fr: {
    'Allowed range: {{min}}-{{max}} minutes (up to 7 days)':
      'Plage autorisee : {{min}}-{{max}} minutes (jusqu a 7 jours)',
    'Choose a fixed interval or a cron schedule for cleanup scans.':
      'Choisissez un intervalle fixe ou une planification cron pour les scans de nettoyage.',
    'Cron schedule': 'Planification cron',
    'Every N hours': 'Toutes les N heures',
    'Execution times use the server local timezone. Preview below follows the server clock.':
      "Les heures d'execution utilisent le fuseau horaire local du serveur. L'apercu ci-dessous suit l'horloge du serveur.",
    'Fixed interval': 'Intervalle fixe',
    'Friday': 'Vendredi',
    'Generated cron expression': 'Expression cron generee',
    'Hour interval': 'Intervalle horaire',
    'Inflight log cleanup interval must be between 1 and 10080 minutes':
      "L'intervalle de nettoyage des journaux en cours doit etre compris entre 1 et 10080 minutes",
    'Inflight log cleanup schedule mode':
      'Mode de planification du nettoyage des journaux en cours',
    'Invalid cron expression': 'Expression cron invalide',
    'Monday': 'Lundi',
    'Next 6 scheduled runs': '6 prochaines executions planifiees',
    'Saturday': 'Samedi',
    'Schedule preset': 'Preset de planification',
    'Server timezone: {{timezone}}': 'Fuseau horaire du serveur : {{timezone}}',
    'Start live refresh': 'Demarrer le rafraichissement en direct',
    'Stop live refresh': 'Arreter le rafraichissement en direct',
    'Sunday': 'Dimanche',
    'Thursday': 'Jeudi',
    'Tuesday': 'Mardi',
    'Wednesday': 'Mercredi',
    'Weekday': 'Jour de la semaine',
    'Automatic inflight log cleanup is disabled':
      'Le nettoyage automatique des journaux en cours est desactive',
    'Clean terminal inflight logs every {{minutes}} minutes':
      'Nettoyer les journaux en cours termines toutes les {{minutes}} minutes',
    'Clean terminal inflight logs on schedule: {{expr}}':
      'Nettoyer les journaux en cours termines selon le planning : {{expr}}',
    'Cleanup in progress': 'Nettoyage en cours',
    'Loading cleanup schedule...': 'Chargement du planning de nettoyage...',
    'Next cleanup: {{time}}': 'Prochain nettoyage : {{time}}',
    'Async Logs': 'Journaux asynchrones',
  },
  ja: {
    'Allowed range: {{min}}-{{max}} minutes (up to 7 days)':
      '設定可能範囲：{{min}}〜{{max}} 分（最長 7 日）',
    'Choose a fixed interval or a cron schedule for cleanup scans.':
      'クリーンアップ走査には固定間隔または cron スケジュールを選択します。',
    'Cron schedule': 'cron スケジュール',
    'Every N hours': 'N 時間ごと',
    'Execution times use the server local timezone. Preview below follows the server clock.':
      '実行時刻はサーバーのローカルタイムゾーンを使用します。下のプレビューはサーバー時計に基づきます。',
    'Fixed interval': '固定間隔',
    'Friday': '金曜日',
    'Generated cron expression': '生成された cron 式',
    'Hour interval': '時間間隔',
    'Inflight log cleanup interval must be between 1 and 10080 minutes':
      '進行中ログのクリーンアップ間隔は 1〜10080 分の範囲で設定してください',
    'Inflight log cleanup schedule mode': '進行中ログのクリーンアップ方式',
    'Invalid cron expression': 'cron 式が無効です',
    'Monday': '月曜日',
    'Next 6 scheduled runs': '今後 6 回の実行予定',
    'Saturday': '土曜日',
    'Schedule preset': 'スケジュールプリセット',
    'Server timezone: {{timezone}}': 'サーバータイムゾーン：{{timezone}}',
    'Start live refresh': 'リアルタイム更新を開始',
    'Stop live refresh': 'リアルタイム更新を停止',
    'Sunday': '日曜日',
    'Thursday': '木曜日',
    'Tuesday': '火曜日',
    'Wednesday': '水曜日',
    'Weekday': '曜日',
    'Automatic inflight log cleanup is disabled':
      '進行中ログの自動クリーンアップは無効です',
    'Clean terminal inflight logs every {{minutes}} minutes':
      '終了した進行中ログを {{minutes}} 分ごとにクリーンアップ',
    'Clean terminal inflight logs on schedule: {{expr}}':
      'スケジュールに従って終了した進行中ログをクリーンアップ：{{expr}}',
    'Cleanup in progress': 'クリーンアップ実行中',
    'Loading cleanup schedule...': 'クリーンアップ予定を読み込み中...',
    'Next cleanup: {{time}}': '次回クリーンアップ：{{time}}',
    'Async Logs': '非同期ログ',
  },
  ru: {
    'Allowed range: {{min}}-{{max}} minutes (up to 7 days)':
      'Допустимый диапазон: {{min}}–{{max}} минут (до 7 дней)',
    'Choose a fixed interval or a cron schedule for cleanup scans.':
      'Выберите фиксированный интервал или cron-расписание для сканирования очистки.',
    'Cron schedule': 'Cron-расписание',
    'Every N hours': 'Каждые N часов',
    'Execution times use the server local timezone. Preview below follows the server clock.':
      'Время выполнения использует локальный часовой пояс сервера. Предпросмотр ниже основан на часах сервера.',
    'Fixed interval': 'Фиксированный интервал',
    'Friday': 'Пятница',
    'Generated cron expression': 'Сгенерированное cron-выражение',
    'Hour interval': 'Интервал в часах',
    'Inflight log cleanup interval must be between 1 and 10080 minutes':
      'Интервал очистки журналов в процессе должен быть от 1 до 10080 минут',
    'Inflight log cleanup schedule mode':
      'Режим расписания очистки журналов в процессе',
    'Invalid cron expression': 'Недопустимое cron-выражение',
    'Monday': 'Понедельник',
    'Next 6 scheduled runs': 'Следующие 6 запланированных запусков',
    'Saturday': 'Суббота',
    'Schedule preset': 'Предустановка расписания',
    'Server timezone: {{timezone}}': 'Часовой пояс сервера: {{timezone}}',
    'Start live refresh': 'Включить обновление в реальном времени',
    'Stop live refresh': 'Остановить обновление в реальном времени',
    'Sunday': 'Воскресенье',
    'Thursday': 'Четверг',
    'Tuesday': 'Вторник',
    'Wednesday': 'Среда',
    'Weekday': 'День недели',
    'Automatic inflight log cleanup is disabled':
      'Автоматическая очистка журналов в процессе отключена',
    'Clean terminal inflight logs every {{minutes}} minutes':
      'Очищать завершенные журналы в процессе каждые {{minutes}} минут',
    'Clean terminal inflight logs on schedule: {{expr}}':
      'Очищать завершенные журналы в процессе по расписанию: {{expr}}',
    'Cleanup in progress': 'Очистка выполняется',
    'Loading cleanup schedule...': 'Загрузка расписания очистки...',
    'Next cleanup: {{time}}': 'Следующая очистка: {{time}}',
    'Async Logs': 'Асинхронные журналы',
  },
  vi: {
    'Allowed range: {{min}}-{{max}} minutes (up to 7 days)':
      'Pham vi cho phep: {{min}}-{{max}} phut (toi da 7 ngay)',
    'Choose a fixed interval or a cron schedule for cleanup scans.':
      'Chon khoang thoi gian co dinh hoac lich cron de quet don dep.',
    'Cron schedule': 'Lich cron',
    'Every N hours': 'Moi N gio',
    'Execution times use the server local timezone. Preview below follows the server clock.':
      'Thoi gian thuc thi dung mui gio cuc bo cua may chu. Xem truoc ben duoi theo dong ho may chu.',
    'Fixed interval': 'Khoang thoi gian co dinh',
    'Friday': 'Thu Sau',
    'Generated cron expression': 'Bieu thuc cron duoc tao',
    'Hour interval': 'Khoang gio',
    'Inflight log cleanup interval must be between 1 and 10080 minutes':
      'Khoang thoi gian don dep nhat ky dang xu ly phai tu 1 den 10080 phut',
    'Inflight log cleanup schedule mode':
      'Che do lich don dep nhat ky dang xu ly',
    'Invalid cron expression': 'Bieu thuc cron khong hop le',
    'Monday': 'Thu Hai',
    'Next 6 scheduled runs': '6 lan chay ke hoach tiep theo',
    'Saturday': 'Thu Bay',
    'Schedule preset': 'Mau lich',
    'Server timezone: {{timezone}}': 'Mui gio may chu: {{timezone}}',
    'Start live refresh': 'Bat dau lam moi thoi gian thuc',
    'Stop live refresh': 'Dung lam moi thoi gian thuc',
    'Sunday': 'Chu Nhat',
    'Thursday': 'Thu Tu',
    'Tuesday': 'Thu Ba',
    'Wednesday': 'Thu Tu',
    'Weekday': 'Ngay trong tuan',
    'Automatic inflight log cleanup is disabled':
      'Tat don dep tu dong nhat ky dang xu ly',
    'Clean terminal inflight logs every {{minutes}} minutes':
      'Don dep nhat ky dang xu ly da ket thuc moi {{minutes}} phut',
    'Clean terminal inflight logs on schedule: {{expr}}':
      'Don dep nhat ky dang xu ly da ket thuc theo lich: {{expr}}',
    'Cleanup in progress': 'Dang don dep',
    'Loading cleanup schedule...': 'Dang tai lich don dep...',
    'Next cleanup: {{time}}': 'Lan don dep tiep theo: {{time}}',
    'Async Logs': 'Nhat ky bat dong bo',
  },
}

async function main() {
  let totalAdded = 0

  for (const [locale, trans] of Object.entries(newKeys)) {
    const filePath = path.join(LOCALES_DIR, `${locale}.json`)
    const json = JSON.parse(await fs.readFile(filePath, 'utf8'))

    let count = 0
    for (const [key, value] of Object.entries(trans)) {
      if (!Object.prototype.hasOwnProperty.call(json.translation, key)) {
        json.translation[key] = value
        count++
      } else if (json.translation[key] !== value) {
        json.translation[key] = value
        count++
      }
    }

    if (count > 0) {
      json.translation = Object.fromEntries(
        Object.entries(json.translation).sort(([a], [b]) => a.localeCompare(b))
      )
      await fs.writeFile(filePath, stableStringify(json), 'utf8')
    }

    console.log(`${locale}: ${count} translations applied`)
    totalAdded += count
  }

  console.log(`\nTotal: ${totalAdded} translations applied`)
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
