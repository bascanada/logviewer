import Sidebar from './lib/components/Sidebar.svelte';
import { mount } from 'svelte';
import './app.css';

const app = mount(Sidebar, {
  target: document.getElementById('app')!,
});

export default app;
