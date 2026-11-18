import { createRoot } from 'react-dom/client';
import { setNonce } from 'get-nonce';
import { ThemeProvider } from './ui';
import { App } from './App';
import './style.css';

const nonce = document.querySelector<HTMLMetaElement>('meta[name="style-nonce"]')?.content;
if (nonce) setNonce(nonce);
createRoot(document.getElementById('root')!).render(<ThemeProvider theme="light" tokens={{ '--yk-primary': '#244d3f', '--yk-primary-hover': '#173b2e', '--yk-text': '#243d37', '--yk-muted': '#718273', '--yk-border': '#dce2d7' }}><App /></ThemeProvider>);
