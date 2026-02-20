import { writable } from 'svelte/store';
import type { LogEntry, QueryMeta } from '$lib/api';

export const logs = writable<LogEntry[]>([]);
export const meta = writable<QueryMeta | null>(null);
export const isLoading = writable(false);
export const error = writable<string | null>(null);
