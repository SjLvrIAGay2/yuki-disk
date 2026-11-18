export class Queue {
  private active = 0;
  private pending: (() => Promise<void>)[] = [];
  constructor(private concurrency = 2) { if (concurrency < 1) throw new Error('invalid concurrency'); }
  add(job: () => Promise<void>) { this.pending.push(job); this.pump(); }
  private pump() {
    while (this.active < this.concurrency && this.pending.length) {
      const job = this.pending.shift()!; this.active++;
      void job().catch(() => {}).finally(() => { this.active--; this.pump(); });
    }
  }
}
export async function parallel<T>(items: T[], concurrency: number, run: (item: T, index: number) => Promise<void>): Promise<void> {
  let index = 0; let failed = false;
  const results = await Promise.allSettled(Array.from({ length: Math.min(Math.max(1, concurrency), items.length) }, async () => {
    while (!failed && index < items.length) {
      const current = index++;
      try { await run(items[current], current); } catch (e) { failed = true; throw e; }
    }
  }));
  for (const r of results) if (r.status === 'rejected') throw r.reason;
}
