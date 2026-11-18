import { createRoot } from 'react-dom/client';
import { App } from './App';
import { ThemeProvider } from './ui';
import { setNonce } from 'get-nonce';
import './style.css';
import './workspace.css';
const styleNonce = document.querySelector<HTMLMetaElement>('meta[name="style-nonce"]')?.content;
if (styleNonce) setNonce(styleNonce);
createRoot(document.getElementById('root')!).render(<ThemeProvider theme="light" tokens={{
  '--yk-primary': '#244d3f', '--yk-primary-hover': '#173b2e', '--yk-primary-soft': '#e7efe3',
  '--yk-text': '#243d37', '--yk-muted': '#718273', '--yk-border': '#dce2d7',
  '--yk-surface': '#f5f6f2', '--yk-surface-hover': '#edf0e8',
}}><App /></ThemeProvider>);
