import { writable, derived } from 'svelte/store';
import type { Context } from '$lib/api';

export const contexts = writable<Context[]>([]);
export const selectedContextId = writable<string | null>(null);

export const selectedContext = derived(
  [contexts, selectedContextId],
  ([$contexts, $selectedContextId]) =>
    $contexts.find((c) => c.id === $selectedContextId) || null
);
