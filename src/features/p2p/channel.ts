// Small bounded messages preserve WebRTC backpressure; no media permission is requested.
export class Channel {
  private queue: Uint8Array<ArrayBuffer>[] = [];
  private queuedBytes = 0;
  private waiter?: { resolve(data: Uint8Array<ArrayBuffer>): void; reject(reason: unknown): void };
  private failure?: Error;
  constructor(readonly data: RTCDataChannel, private signal: AbortSignal) {
    data.binaryType = 'arraybuffer'; data.bufferedAmountLowThreshold = 65536;
    data.addEventListener('message', this.message);
    data.addEventListener('close', this.closed); data.addEventListener('error', this.closed);
    signal.addEventListener('abort', this.cancel, { once: true }); if (signal.aborted) this.cancel();
  }
  private message = (event: MessageEvent) => {
    if (!(event.data instanceof ArrayBuffer) || event.data.byteLength < 1 || event.data.byteLength > 16385) { this.fail(new Error('传输响应无效')); return; }
    const data = new Uint8Array(event.data);
    if (this.waiter) { const waiter = this.waiter; this.waiter = undefined; waiter.resolve(data); }
    else { this.queuedBytes += data.byteLength; if (this.queuedBytes > 8 * 1024 * 1024) this.fail(new Error('传输缓冲已满')); else this.queue.push(data); }
  };
  private closed = () => this.fail(new Error('连接已关闭'));
  private cancel = () => this.fail(new Error('传输已取消'));
  private fail(error: Error) { this.failure = error; this.waiter?.reject(error); this.waiter = undefined; }
  async read(): Promise<Uint8Array<ArrayBuffer>> {
    this.signal.throwIfAborted();
    const next = this.queue.shift(); if (next) { this.queuedBytes -= next.byteLength; return next; }
    if (this.failure) throw this.failure;
    if (this.waiter) throw new Error('重复读取');
    return new Promise((resolve, reject) => { this.waiter = { resolve, reject }; });
  }
  async send(data: Uint8Array<ArrayBuffer>): Promise<void> {
    this.signal.throwIfAborted();
    if (this.data.bufferedAmount > 262144) await new Promise<void>((resolve, reject) => {
      const cleanup = () => { this.data.removeEventListener('bufferedamountlow', ready); this.data.removeEventListener('close', failed); this.signal.removeEventListener('abort', failed); };
      const ready = () => { cleanup(); resolve(); };
      const failed = () => { cleanup(); reject(new Error('连接已关闭')); };
      this.data.addEventListener('bufferedamountlow', ready); this.data.addEventListener('close', failed); this.signal.addEventListener('abort', failed, { once: true });
      if (this.signal.aborted || this.data.readyState !== 'open') failed(); else if (this.data.bufferedAmount <= 65536) ready();
    });
    this.signal.throwIfAborted();
    if (this.failure || this.data.readyState !== 'open') throw this.failure || new Error('连接已关闭');
    this.data.send(data);
  }
  close() {
    this.data.removeEventListener('message', this.message); this.data.removeEventListener('close', this.closed); this.data.removeEventListener('error', this.closed); this.signal.removeEventListener('abort', this.cancel);
    this.closed(); this.queue = []; this.queuedBytes = 0; this.data.close();
  }
}
