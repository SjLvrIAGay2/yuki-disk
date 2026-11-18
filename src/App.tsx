import { HealthPanel } from './features/health/HealthPanel';
import { useHealth } from './features/health/useHealth';

export function App() {
  const health = useHealth();
  return <main className="status-page"><header className="page-heading"><p className="eyebrow">YUKI DISK</p><h1>服务状态</h1></header>
    <HealthPanel {...health} /></main>;
}
