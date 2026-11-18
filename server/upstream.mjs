export class HTTPError extends Error {
  constructor(status, code) { super(code); this.status = status; this.code = code; }
}
export async function readJSON(response, maxBytes = 524288) {
  if (!response.body) throw new Error('Missing body');
  const chunks = []; let length = 0;
  const reader = response.body.getReader();
  try {
    while (true) {
      const { done, value } = await reader.read(); if (done) break;
      length += value.length;
      if (length > maxBytes) { await reader.cancel(); throw new Error('Response too large'); }
      chunks.push(value);
    }
  } finally { reader.releaseLock(); }
  return JSON.parse(Buffer.concat(chunks).toString('utf8'));
}
export async function controlJSON(config, route, options, stopSignal) {
  const response = await fetch(config.control + route, {
    ...options, redirect: 'error', signal: AbortSignal.any([stopSignal, AbortSignal.timeout(5000)]),
  });
  if (!response.ok) {
    await response.body?.cancel();
    throw new HTTPError(response.status === 401 ? 401 : response.status === 403 ? 403 : response.status === 429 ? 429 : 503,
      response.status === 401 ? 'unauthorized' : 'control_unavailable');
  }
  return readJSON(response);
}
