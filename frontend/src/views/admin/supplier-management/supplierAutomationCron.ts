export function cronToIntervalSeconds(cronExpression: string): number | null {
  const everyMatch = cronExpression.match(/^@every\s+(\d+)s$/)
  if (everyMatch) return Number(everyMatch[1])

  const parts = cronExpression.trim().split(/\s+/)
  if (parts.length !== 5) return null
  const [minute, hour, dayOfMonth, month, dayOfWeek] = parts
  if (hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    const minuteMatch = minute.match(/^\*\/(\d+)$/)
    if (minuteMatch) return Number(minuteMatch[1]) * 60
    if (minute === '*') return 60
  }
  if (minute === '0' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    const hourMatch = hour.match(/^\*\/(\d+)$/)
    if (hourMatch) return Number(hourMatch[1]) * 3600
    if (hour === '0') return 86400
  }
  if (dayOfMonth === '*' && month === '*' && dayOfWeek === '*' && minute !== '*' && hour !== '*') {
    return 86400
  }
  return null
}

export function formatDurationMinutes(minutes: number): string {
  if (minutes < 60) return `${minutes} 分钟`
  if (minutes < 1440) return `${Math.floor(minutes / 60)} 小时`
  return `${Math.floor(minutes / 1440)} 天`
}
