// A session ID is only a pool partition, never a credential. Requests and new
// signaling tickets still pass the current access token through server auth.
export function connectionScope(token: string): string {
  if (!token) return '';
  try {
    const claims = JSON.parse(atob(token.split('.')[0].replace(/-/g, '+').replace(/_/g, '/')));
    if (claims.v === 1 && claims.purpose === 'session' && typeof claims.jti === 'string' && /^[a-f0-9]{48}$/.test(claims.jti))
      return 'session:' + claims.jti;
  } catch { /* Opaque capabilities must not share the login-session partition. */ }
  return token;
}
