import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
  return JSON.stringify(obj, null, 2) + '\n'
}

const newKeys = {
  en: {
    'Attempt {{current}} of {{total}}': 'Attempt {{current}} of {{total}}',
    'Attempt {{number}}': 'Attempt {{number}}',
    'Choose whether terminal inflight logs are cleaned automatically.': 'Choose whether terminal inflight logs are cleaned automatically.',
    'Clean inflight logs': 'Clean inflight logs',
    'Clean terminal logs': 'Clean terminal logs',
    'Current Node': 'Current Node',
    'Current Retry': 'Current Retry',
    'Current Stage': 'Current Stage',
    'Current inflight log usage': 'Current inflight log usage',
    'Do not clean': 'Do not clean',
    'Failure Reason': 'Failure Reason',
    'Final Failure': 'Final Failure',
    'How often the cron cleanup scans terminal inflight logs.': 'How often the cron cleanup scans terminal inflight logs.',
    'Inflight log cleanup interval (minutes)': 'Inflight log cleanup interval (minutes)',
    'Inflight log cleanup progress': 'Inflight log cleanup progress',
    'Inflight log cleanup rule': 'Inflight log cleanup rule',
    'Inflight log cleanup task started.': 'Inflight log cleanup task started.',
    'Inflight log entries': 'Inflight log entries',
    'Latest Error': 'Latest Error',
    'No Retry': 'No Retry',
    'Remove terminal inflight logs updated before the selected timestamp.': 'Remove terminal inflight logs updated before the selected timestamp.',
    'Retry {{count}}': 'Retry {{count}}',
    'Retry Path': 'Retry Path',
    'This will permanently remove terminal inflight logs before the selected timestamp.': 'This will permanently remove terminal inflight logs before the selected timestamp.',
    'This will permanently remove terminal inflight logs updated before {{date}}.': 'This will permanently remove terminal inflight logs updated before {{date}}.',
    'Total inflight log size': 'Total inflight log size',
    'Attempts': 'Attempts',
    'View the current Redis space used by inflight logs.': 'View the current Redis space used by inflight logs.',
    '{{processed}} of {{total}} inflight logs processed.': '{{processed}} of {{total}} inflight logs processed.'
  },
  zh: {
    'Attempt {{current}} of {{total}}': '第 {{current}} 次，共 {{total}} 次',
    'Attempt {{number}}': '第 {{number}} 次',
    'Choose whether terminal inflight logs are cleaned automatically.': '选择是否自动清理终态在途日志。',
    'Clean inflight logs': '清理在途日志',
    'Clean terminal logs': '清理终态日志',
    'Current Node': '当前节点',
    'Current Retry': '当前重试次数',
    'Current Stage': '当前阶段',
    'Current inflight log usage': '当前在途日志占用',
    'Do not clean': '不清理',
    'Failure Reason': '失败原因',
    'Final Failure': '最终失败',
    'How often the cron cleanup scans terminal inflight logs.': '设置定时任务扫描终态在途日志的频率。',
    'Inflight log cleanup interval (minutes)': '在途日志清理间隔（分钟）',
    'Inflight log cleanup progress': '在途日志清理进度',
    'Inflight log cleanup rule': '在途日志清理规则',
    'Inflight log cleanup task started.': '已启动在途日志清理任务。',
    'Inflight log entries': '在途日志条目',
    'Latest Error': '最近一次错误',
    'No Retry': '未重试',
    'Remove terminal inflight logs updated before the selected timestamp.': '删除所选时间之前更新的终态在途日志。',
    'Retry {{count}}': '重试 {{count}}',
    'Retry Path': '重试路径',
    'This will permanently remove terminal inflight logs before the selected timestamp.': '这将永久删除所选时间之前的终态在途日志。',
    'This will permanently remove terminal inflight logs updated before {{date}}.': '这将永久删除 {{date}} 之前更新的终态在途日志。',
    'Total inflight log size': '在途日志总大小',
    'Attempts': '尝试次数',
    'View the current Redis space used by inflight logs.': '查看当前在途日志占用的 Redis 空间。',
    '{{processed}} of {{total}} inflight logs processed.': '已处理 {{processed}} / {{total}} 条在途日志。'
  },
  fr: {
    'Attempt {{current}} of {{total}}': 'Tentative {{current}} sur {{total}}',
    'Attempt {{number}}': 'Tentative {{number}}',
    'Choose whether terminal inflight logs are cleaned automatically.': 'Choisissez si les journaux en cours terminaux sont nettoyes automatiquement.',
    'Clean inflight logs': 'Nettoyer les journaux en cours',
    'Clean terminal logs': 'Nettoyer les journaux terminaux',
    'Current Node': 'Noeud actuel',
    'Current Retry': 'Nouvelle tentative actuelle',
    'Current Stage': 'Etape actuelle',
    'Current inflight log usage': 'Utilisation actuelle des journaux en cours',
    'Do not clean': 'Ne pas nettoyer',
    'Failure Reason': 'Cause de l echec',
    'Final Failure': 'Echec final',
    'How often the cron cleanup scans terminal inflight logs.': 'Frequence a laquelle le cron analyse les journaux en cours terminaux.',
    'Inflight log cleanup interval (minutes)': 'Intervalle de nettoyage des journaux en cours (minutes)',
    'Inflight log cleanup progress': 'Progression du nettoyage des journaux en cours',
    'Inflight log cleanup rule': 'Regle de nettoyage des journaux en cours',
    'Inflight log cleanup task started.': 'La tache de nettoyage des journaux en cours a demarre.',
    'Inflight log entries': 'Entrees des journaux en cours',
    'Latest Error': 'Derniere erreur',
    'No Retry': 'Aucune nouvelle tentative',
    'Remove terminal inflight logs updated before the selected timestamp.': 'Supprimer les journaux en cours terminaux mis a jour avant l heure choisie.',
    'Retry {{count}}': 'Nouvelle tentative {{count}}',
    'Retry Path': 'Chemin de nouvelle tentative',
    'This will permanently remove terminal inflight logs before the selected timestamp.': 'Cela supprimera definitivement les journaux en cours terminaux avant l horodatage choisi.',
    'This will permanently remove terminal inflight logs updated before {{date}}.': 'Cela supprimera definitivement les journaux en cours terminaux mis a jour avant {{date}}.',
    'Total inflight log size': 'Taille totale des journaux en cours',
    'Attempts': 'Tentatives',
    'View the current Redis space used by inflight logs.': 'Afficher l espace Redis actuellement utilise par les journaux en cours.',
    '{{processed}} of {{total}} inflight logs processed.': '{{processed}} journaux en cours traites sur {{total}}.'
  },
  ja: {
    'Attempt {{current}} of {{total}}': '{{total}} 回中 {{current}} 回目',
    'Attempt {{number}}': '{{number}} 回目',
    'Choose whether terminal inflight logs are cleaned automatically.': '終了状態の進行中ログを自動的にクリーンアップするか選択します。',
    'Clean inflight logs': '進行中ログをクリーンアップ',
    'Clean terminal logs': '終了済みログをクリーンアップ',
    'Current Node': '現在のノード',
    'Current Retry': '現在の再試行回数',
    'Current Stage': '現在の段階',
    'Current inflight log usage': '現在の進行中ログ使用量',
    'Do not clean': 'クリーンアップしない',
    'Failure Reason': '失敗理由',
    'Final Failure': '最終失敗',
    'How often the cron cleanup scans terminal inflight logs.': 'cron クリーンアップが終了状態の進行中ログを走査する頻度です。',
    'Inflight log cleanup interval (minutes)': '進行中ログのクリーンアップ間隔（分）',
    'Inflight log cleanup progress': '進行中ログのクリーンアップ進捗',
    'Inflight log cleanup rule': '進行中ログのクリーンアップルール',
    'Inflight log cleanup task started.': '進行中ログのクリーンアップタスクを開始しました。',
    'Inflight log entries': '進行中ログ件数',
    'Latest Error': '最新エラー',
    'No Retry': '再試行なし',
    'Remove terminal inflight logs updated before the selected timestamp.': '選択した時刻より前に更新された終了状態の進行中ログを削除します。',
    'Retry {{count}}': '再試行 {{count}}',
    'Retry Path': '再試行経路',
    'This will permanently remove terminal inflight logs before the selected timestamp.': '選択した時刻より前の終了状態の進行中ログを完全に削除します。',
    'This will permanently remove terminal inflight logs updated before {{date}}.': '{{date}} より前に更新された終了状態の進行中ログを完全に削除します。',
    'Total inflight log size': '進行中ログの合計サイズ',
    'Attempts': '試行回数',
    'View the current Redis space used by inflight logs.': '進行中ログが現在使用している Redis 容量を表示します。',
    '{{processed}} of {{total}} inflight logs processed.': '{{total}} 件中 {{processed}} 件の進行中ログを処理しました。'
  },
  ru: {
    'Attempt {{current}} of {{total}}': 'Попытка {{current}} из {{total}}',
    'Attempt {{number}}': 'Попытка {{number}}',
    'Choose whether terminal inflight logs are cleaned automatically.': 'Выберите, нужно ли автоматически очищать завершенные журналы в процессе.',
    'Clean inflight logs': 'Очистить журналы в процессе',
    'Clean terminal logs': 'Очистить завершенные журналы',
    'Current Node': 'Текущий узел',
    'Current Retry': 'Текущая повторная попытка',
    'Current Stage': 'Текущий этап',
    'Current inflight log usage': 'Текущее использование журналов в процессе',
    'Do not clean': 'Не очищать',
    'Failure Reason': 'Причина сбоя',
    'Final Failure': 'Окончательный сбой',
    'How often the cron cleanup scans terminal inflight logs.': 'Как часто cron-очистка сканирует завершенные журналы в процессе.',
    'Inflight log cleanup interval (minutes)': 'Интервал очистки журналов в процессе (минуты)',
    'Inflight log cleanup progress': 'Ход очистки журналов в процессе',
    'Inflight log cleanup rule': 'Правило очистки журналов в процессе',
    'Inflight log cleanup task started.': 'Задача очистки журналов в процессе запущена.',
    'Inflight log entries': 'Записи журналов в процессе',
    'Latest Error': 'Последняя ошибка',
    'No Retry': 'Без повторных попыток',
    'Remove terminal inflight logs updated before the selected timestamp.': 'Удалить завершенные журналы в процессе, обновленные до выбранного времени.',
    'Retry {{count}}': 'Повторная попытка {{count}}',
    'Retry Path': 'Путь повторных попыток',
    'This will permanently remove terminal inflight logs before the selected timestamp.': 'Это навсегда удалит завершенные журналы в процессе до выбранной отметки времени.',
    'This will permanently remove terminal inflight logs updated before {{date}}.': 'Это навсегда удалит завершенные журналы в процессе, обновленные до {{date}}.',
    'Total inflight log size': 'Общий размер журналов в процессе',
    'Attempts': 'Попытки',
    'View the current Redis space used by inflight logs.': 'Просмотр текущего места в Redis, используемого журналами в процессе.',
    '{{processed}} of {{total}} inflight logs processed.': 'Обработано {{processed}} из {{total}} журналов в процессе.'
  },
  vi: {
    'Attempt {{current}} of {{total}}': 'Lan thu {{current}} trong {{total}} lan',
    'Attempt {{number}}': 'Lan thu {{number}}',
    'Choose whether terminal inflight logs are cleaned automatically.': 'Chon co tu dong don dep cac nhat ky dang xu ly da ket thuc hay khong.',
    'Clean inflight logs': 'Don dep nhat ky dang xu ly',
    'Clean terminal logs': 'Don dep nhat ky da ket thuc',
    'Current Node': 'Nut hien tai',
    'Current Retry': 'Lan thu lai hien tai',
    'Current Stage': 'Giai doan hien tai',
    'Current inflight log usage': 'Muc su dung nhat ky dang xu ly hien tai',
    'Do not clean': 'Khong don dep',
    'Failure Reason': 'Ly do that bai',
    'Final Failure': 'That bai cuoi cung',
    'How often the cron cleanup scans terminal inflight logs.': 'Tan suat tac vu cron quet cac nhat ky dang xu ly da ket thuc.',
    'Inflight log cleanup interval (minutes)': 'Khoang thoi gian don dep nhat ky dang xu ly (phut)',
    'Inflight log cleanup progress': 'Tien do don dep nhat ky dang xu ly',
    'Inflight log cleanup rule': 'Quy tac don dep nhat ky dang xu ly',
    'Inflight log cleanup task started.': 'Da bat dau tac vu don dep nhat ky dang xu ly.',
    'Inflight log entries': 'Ban ghi nhat ky dang xu ly',
    'Latest Error': 'Loi gan nhat',
    'No Retry': 'Khong thu lai',
    'Remove terminal inflight logs updated before the selected timestamp.': 'Xoa cac nhat ky dang xu ly da ket thuc duoc cap nhat truoc thoi diem da chon.',
    'Retry {{count}}': 'Thu lai {{count}}',
    'Retry Path': 'Duong di thu lai',
    'This will permanently remove terminal inflight logs before the selected timestamp.': 'Thao tac nay se xoa vinh vien cac nhat ky dang xu ly da ket thuc truoc thoi diem da chon.',
    'This will permanently remove terminal inflight logs updated before {{date}}.': 'Thao tac nay se xoa vinh vien cac nhat ky dang xu ly da ket thuc duoc cap nhat truoc {{date}}.',
    'Total inflight log size': 'Tong dung luong nhat ky dang xu ly',
    'Attempts': 'So lan thu',
    'View the current Redis space used by inflight logs.': 'Xem dung luong Redis hien dang duoc su dung boi nhat ky dang xu ly.',
    '{{processed}} of {{total}} inflight logs processed.': 'Da xu ly {{processed}} / {{total}} nhat ky dang xu ly.'
  }
}

async function main() {
  for (const [locale, trans] of Object.entries(newKeys)) {
    const filePath = path.join(LOCALES_DIR, `${locale}.json`)
    const json = JSON.parse(await fs.readFile(filePath, 'utf8'))
    for (const [key, value] of Object.entries(trans)) {
      json.translation[key] = value
    }
    json.translation = Object.fromEntries(
      Object.entries(json.translation).sort(([a], [b]) => a.localeCompare(b))
    )
    await fs.writeFile(filePath, stableStringify(json), 'utf8')
  }
}

main().catch((err) => {
  console.error(err)
  process.exitCode = 1
})
