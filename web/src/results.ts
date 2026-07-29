import Results from './lib/components/Results.svelte';
import { mount } from 'svelte';
import './app.css';

// Get search params from URL or window
declare global {
  interface Window {
    LOGVIEWER_PORT?: number;
    LOGVIEWER_SEARCH?: {
      contextId: string;
      query?: string;
      timeRange?: string;
      level?: string;
    };
  }
}

const searchParams = window.LOGVIEWER_SEARCH || {
  contextId: 'default',
  query: '',
  timeRange: '1h',
  level: undefined,
};

const app = mount(Results, {
  target: document.getElementById('app')!,
  props: searchParams,
});

export default app;
