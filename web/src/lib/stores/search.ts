import { writable } from 'svelte/store';

export interface SearchState {
  query: string;
  timeRange: string; // "15m", "1h", "24h", "7d"
  level: string | null;
  size: number; // Number of results per page
}

export const searchState = writable<SearchState>({
  query: '',
  timeRange: '1h',
  level: null,
  size: 100,
});
