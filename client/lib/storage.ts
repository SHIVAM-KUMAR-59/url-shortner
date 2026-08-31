const API_KEY_STORAGE = 'shortener_api_key';
const EMAIL_STORAGE = 'shortener_email';
const LINKS_STORAGE = 'shortener_links';

export interface TrackedLink {
  shortCode: string;
  longUrl: string;
  createdAt: string;
}

export function saveSession(email: string, apiKey: string) {
  localStorage.setItem(EMAIL_STORAGE, email);
  localStorage.setItem(API_KEY_STORAGE, apiKey);
}

export function getSession(): { email: string; apiKey: string } | null {
  const email = localStorage.getItem(EMAIL_STORAGE);
  const apiKey = localStorage.getItem(API_KEY_STORAGE);
  if (!email || !apiKey) return null;
  return { email, apiKey };
}

export function clearSession() {
  localStorage.removeItem(EMAIL_STORAGE);
  localStorage.removeItem(API_KEY_STORAGE);
}

export function getTrackedLinks(): TrackedLink[] {
  const raw = localStorage.getItem(LINKS_STORAGE);
  if (!raw) return [];
  try {
    return JSON.parse(raw);
  } catch {
    return [];
  }
}

export function addTrackedLink(link: TrackedLink) {
  const links = getTrackedLinks();
  links.unshift(link);
  localStorage.setItem(LINKS_STORAGE, JSON.stringify(links));
}