export type TransferDetail = { transferred: number; total: number; phase: 'preparing' | 'transfer' | 'saving' | 'retry' };
export type TransferProgress = (percent: number, label: string, detail?: TransferDetail) => void;
// Completed/received useful file bytes, not wire bytes across replicas or retries.
export class TransferMeter {
  private samples: { time: number; bytes: number }[] = [];
  private detail?: TransferDetail;
  update(detail: TransferDetail, now = performance.now()) {
    if (!Number.isFinite(detail.transferred) || !Number.isFinite(detail.total) || detail.transferred < 0 || detail.total < detail.transferred) return;
    const previous = this.detail;
    if (!previous || previous.phase !== 'transfer' || detail.transferred < previous.transferred) this.samples = [{time: now, bytes: previous?.phase === 'transfer' ? detail.transferred : 0}];
    this.detail = detail;
    if (detail.phase !== 'transfer') { this.samples=[]; return; }
    this.samples.push({time:now,bytes:detail.transferred});
    this.trim(now);
  }
  private trim(now: number) { while(this.samples.length>1 && this.samples[1].time<=now-5000)this.samples.shift(); }
  snapshot(now = performance.now()) {
    this.trim(now);const d=this.detail,first=this.samples[0],last=this.samples.at(-1);
    const elapsed=first?(now-first.time)/1000:0;
    const speed=d?.phase==='transfer'&&first&&last&&elapsed>=.25?(now-last.time>=5000?0:Math.max(0,(last.bytes-first.bytes)/elapsed)):undefined;
    return { transferred:d?.transferred, size:d?.total, speed, remaining:speed&&d&&d.total>d.transferred?(d.total-d.transferred)/speed:undefined };
  }
}
export function duration(seconds?: number) {
  if(seconds===undefined||!Number.isFinite(seconds))return '估算中';
  const n=Math.max(1,Math.ceil(seconds));if(n<60)return `${n} 秒`;if(n<3600)return `${Math.ceil(n/60)} 分钟`;return `${Math.floor(n/3600)} 小时 ${Math.ceil(n%3600/60)} 分钟`;
}
