/**
 * Client-side logging for UI: console output with [ui] prefix and in-memory buffer.
 * Use for debugging and optional export/send to backend.
 */

const PREFIX = '[ui]'
const MAX_BUFFER = 100

export type LogLevel = 'info' | 'warn' | 'error'

export interface LogEntry {
  ts: string
  level: LogLevel
  action: string
  payload?: Record<string, unknown>
}

const buffer: LogEntry[] = []

function push(level: LogLevel, action: string, payload?: Record<string, unknown>) {
  const entry: LogEntry = { ts: new Date().toISOString(), level, action, payload }
  buffer.push(entry)
  if (buffer.length > MAX_BUFFER) buffer.shift()
  const msg = payload ? `${PREFIX} ${action} ${JSON.stringify(payload)}` : `${PREFIX} ${action}`
  if (level === 'error') console.error(msg)
  else if (level === 'warn') console.warn(msg)
  else console.log(msg)
}

export function clientLog(action: string, payload?: Record<string, unknown>) {
  push('info', action, payload)
}

export function clientLogWarn(action: string, payload?: Record<string, unknown>) {
  push('warn', action, payload)
}

export function clientLogError(action: string, payload?: Record<string, unknown>) {
  push('error', action, payload)
}

export function getLogBuffer(): LogEntry[] {
  return [...buffer]
}

export function clearLogBuffer(): void {
  buffer.length = 0
}
