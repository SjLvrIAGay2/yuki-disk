// Per-download feedback; counts verified bytes delivered to the destination.
// Probe two extra slots promptly, retain them only when throughput improves. Never cancel
// an already-authorized read merely because the target window shrinks.
export class DownloadWindow {
  private limit: number;
  private started?: number;
  private bytes = 0;
  private samples = 0;
  private baseline = 0;
  private trial?: { limit: number; rate: number };
  private cooldown = 0;
  private retryCooldown = 0;
  constructor(initial: number, private enabled = true, totalBytes = 0) {
    if (!Number.isInteger(initial) || initial < 1 || initial > 6) throw new Error('无效并发设置');
    // Large files can amortize parallel handshakes. Open their full bounded
    // window initially, rather than stalling its middle on newly added peers.
    this.limit = enabled && totalBytes >= 64*1024*1024 ? 6 : initial;
  }
  get concurrency() { return this.limit; }
  private reset(now: number) { this.started = now; this.bytes = 0; this.samples = 0; }
  completed(bytes: number, now = performance.now()) {
    if (!this.enabled || !Number.isFinite(bytes) || bytes <= 0) return;
    // Exclude connection establishment from the first throughput baseline.
    if (this.started === undefined || now < this.cooldown) { this.reset(now); return; }
    this.bytes += bytes; this.samples++;
    const seconds = (now - this.started) / 1000;
    if (seconds < .5 || this.samples < 2) return;
    const rate = this.bytes / seconds;
    this.reset(now);
    if (this.trial) {
      if (rate < this.trial.rate * 1.05) {
        this.limit = this.trial.limit; this.baseline = rate;
        this.trial = undefined; this.cooldown = now + 1500; return;
      }
      this.trial = undefined;
    } else if (this.baseline && rate < this.baseline * .75) {
      this.limit = Math.max(1, this.limit - 1); this.baseline = rate;
      this.cooldown = now + 1500; return;
    }
    this.baseline = rate;
    if (this.limit < 6) { this.trial = {limit:this.limit, rate}; this.limit=Math.min(6,this.limit+2); }
  }
  retry(now = performance.now()) {
    if (!this.enabled || now < this.retryCooldown) return;
    this.limit = Math.max(1, Math.floor(this.limit / 2));
    this.trial = undefined; this.baseline = 0; this.cooldown = now + 2000;this.retryCooldown=now+2000;
    this.reset(now);
  }
}
