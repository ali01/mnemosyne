import { writable } from 'svelte/store';

export interface Route {
  type: 'home' | 'graph';
  vaultName?: string;
  graphPath?: string;
}

function parsePath(): Route {
  const path = window.location.pathname;

  // /vaultName/graphPath
  const graphMatch = path.match(/^\/([^/]+)\/([^/]+)$/);
  if (graphMatch) {
    return { type: 'graph', vaultName: decodeURIComponent(graphMatch[1]), graphPath: decodeURIComponent(graphMatch[2]) };
  }

  return { type: 'home' };
}

export const route = writable<Route>(parsePath());

window.addEventListener('popstate', () => {
  route.set(parsePath());
});

export function navigate(path: string) {
  window.history.pushState(null, '', path);
  route.set(parsePath());
}
