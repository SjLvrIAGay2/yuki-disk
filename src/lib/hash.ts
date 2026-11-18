export const hex = (data: Uint8Array): string => Array.from(data, n => n.toString(16).padStart(2, '0')).join('');

export async function sha256(data: ArrayBuffer): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', data);
  return hex(new Uint8Array(digest));
}
