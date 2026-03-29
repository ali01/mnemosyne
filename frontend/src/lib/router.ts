import { writable } from 'svelte/store';

interface Route {
  path: string;
  params: Record<string, string>;
}

function parseHash(): Route {
  const hash = window.location.hash.slice(1) || '/';
  const noteMatch = hash.match(/^\/notes\/([^/]+)$/);
  if (noteMatch) {
    return { path: '/notes/:id', params: { id: noteMatch[1] } };
  }
  return { path: '/', params: {} };
}

export const route = writable<Route>(parseHash());

window.addEventListener('hashchange', () => {
  route.set(parseHash());
});

export function navigate(path: string) {
  window.location.hash = path;
}
