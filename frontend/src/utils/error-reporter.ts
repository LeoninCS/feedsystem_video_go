export function reportError(error: Error, context?: Record<string, unknown>) {
  if (import.meta.env.DEV) {
    console.error('[ErrorReporter]', error.message, context)
    return
  }
  fetch('/api/error-report', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      message: error.message,
      stack: error.stack,
      context,
      timestamp: new Date().toISOString(),
    }),
  }).catch(() => {
    /* 静默失败，避免错误上报自身导致循环 */
  })
}
