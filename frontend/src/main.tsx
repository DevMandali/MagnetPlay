import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { registerSkipButtons, registerStatsButton, configureVhs } from './lib/videoSetup';
import App from './App';
import './App.css';

registerSkipButtons();
registerStatsButton();
configureVhs();

document.addEventListener('keydown', (e) => {
  if ((e.ctrlKey || e.metaKey) && ['s', 'u', 'a'].includes(e.key.toLowerCase()))
    e.preventDefault();
});

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
