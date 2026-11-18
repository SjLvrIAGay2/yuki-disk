type ConnectionStatus = 'none' | 'direct' | 'relay' | 'http';
let current: ConnectionStatus = 'none';
const listeners = new Set<() => void>();
export const connectionSnapshot = () => current;
export function subscribeConnection(listener: () => void) { listeners.add(listener); return () => { listeners.delete(listener); }; }
export function reportConnection(status: ConnectionStatus) { current = status; for (const listener of listeners) listener(); }
