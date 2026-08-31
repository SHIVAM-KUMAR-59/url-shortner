'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { shortenUrl, getStats, getRedirectUrl, ApiError, StatsResponse } from '@/lib/api';
import {
  getSession,
  clearSession,
  saveSession,
  getTrackedLinks,
  addTrackedLink,
  TrackedLink,
} from '@/lib/storage';
import Link from 'next/link';

export default function DashboardPage() {
  const router = useRouter();
  const [session, setSession] = useState<{ email: string; apiKey: string } | null>(null);
  const [keyInput, setKeyInput] = useState('');
  const [emailInput, setEmailInput] = useState('');
  const [longUrl, setLongUrl] = useState('');
  const [links, setLinks] = useState<TrackedLink[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [statsFor, setStatsFor] = useState<string | null>(null);
  const [stats, setStats] = useState<StatsResponse | null>(null);
  const [statsError, setStatsError] = useState<string | null>(null);
  const [copiedCode, setCopiedCode] = useState<string | null>(null);

  useEffect(() => {
    const s = getSession();
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setSession(s);
    if (s) setLinks(getTrackedLinks());
  }, []);

  function handleUseKey(e: React.FormEvent) {
    e.preventDefault();
    if (!keyInput.trim()) return;
    saveSession(emailInput || 'unknown', keyInput.trim());
    const s = getSession();
    setSession(s);
    if (s) setLinks(getTrackedLinks());
  }

  function handleSignOut() {
    clearSession();
    router.push('/');
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!session) return;
    setError(null);
    setCreating(true);
    try {
      const res = await shortenUrl(longUrl, session.apiKey);
      const link: TrackedLink = {
        shortCode: res.short_code,
        longUrl,
        createdAt: new Date().toISOString(),
      };
      addTrackedLink(link);
      setLinks(getTrackedLinks());
      setLongUrl('');
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong.');
    } finally {
      setCreating(false);
    }
  }

  async function handleViewStats(shortCode: string) {
    if (!session) return;
    setStatsFor(shortCode);
    setStats(null);
    setStatsError(null);
    try {
      const res = await getStats(shortCode, session.apiKey);
      setStats(res);
    } catch (err) {
      setStatsError(err instanceof ApiError ? err.message : 'Could not load stats.');
    }
  }

  async function handleCopy(code: string) {
    await navigator.clipboard.writeText(getRedirectUrl(code));
    setCopiedCode(code);
    setTimeout(() => setCopiedCode(null), 1200);
  }

  if (!session) {
    return (
      <main style={styles.page}>
        <div style={styles.authBox}>
          <div style={styles.brand}>shrtn</div>
          <h1 style={styles.authTitle}>Sign in with an API key</h1>
          <p style={styles.hint}>
            Don&rsquo;t have one? <Link href="/">Create one here</Link>.
          </p>
          <form onSubmit={handleUseKey} style={styles.form}>
            <label style={styles.label}>Email (for display only)</label>
            <input
              style={styles.input}
              value={emailInput}
              onChange={(e) => setEmailInput(e.target.value)}
              placeholder="you@domain.com"
            />
            <label style={styles.label}>API key</label>
            <input
              style={styles.input}
              className="mono"
              value={keyInput}
              onChange={(e) => setKeyInput(e.target.value)}
              placeholder="paste your key"
            />
            <button type="submit" style={styles.button}>
              Continue
            </button>
          </form>
        </div>
      </main>
    );
  }

  return (
    <main style={styles.page}>
      <div style={styles.shell}>
        <header style={styles.header}>
          <div style={styles.brand}>shrtn</div>
          <div style={styles.headerRight}>
            <span style={styles.emailTag}>{session.email}</span>
            <button onClick={handleSignOut} style={styles.linkButton}>
              Sign out
            </button>
          </div>
        </header>

        <section style={styles.createSection}>
          <form onSubmit={handleCreate} style={styles.createForm}>
            <input
              style={styles.urlInput}
              type="url"
              required
              placeholder="https://example.com/a-very-long-path-worth-shortening"
              value={longUrl}
              onChange={(e) => setLongUrl(e.target.value)}
            />
            <button type="submit" disabled={creating} style={styles.button}>
              {creating ? 'Shortening…' : 'Shorten'}
            </button>
          </form>
          {error && <div style={styles.error}>{error}</div>}
        </section>

        <section>
          <div style={styles.tableHeader}>
            <span style={{ flex: '0 0 160px' }}>SHORT CODE</span>
            <span style={{ flex: 1 }}>DESTINATION</span>
            <span style={{ flex: '0 0 140px' }}>CREATED</span>
            <span style={{ flex: '0 0 160px' }}></span>
          </div>

          {links.length === 0 ? (
            <div style={styles.empty}>
              Nothing shortened yet in this browser. Create your first
              link above.
            </div>
          ) : (
            links.map((link) => (
              <div key={link.shortCode + link.createdAt}>
                <div style={styles.row}>
                  <span style={{ ...styles.cell, flex: '0 0 160px' }} className="mono">
                    <a href={getRedirectUrl(link.shortCode)} target="_blank" rel="noreferrer">
                      {link.shortCode}
                    </a>
                  </span>
                  <span style={{ ...styles.cell, flex: 1, color: 'var(--text-dim)' }}>
                    {link.longUrl}
                  </span>
                  <span style={{ ...styles.cell, flex: '0 0 140px', color: 'var(--text-dim)' }}>
                    {new Date(link.createdAt).toLocaleDateString()}
                  </span>
                  <span style={{ ...styles.cell, flex: '0 0 160px', display: 'flex', gap: 12 }}>
                    <button onClick={() => handleCopy(link.shortCode)} style={styles.miniButton}>
                      {copiedCode === link.shortCode ? 'Copied' : 'Copy'}
                    </button>
                    <button
                      onClick={() => handleViewStats(link.shortCode)}
                      style={styles.miniButton}
                    >
                      Stats
                    </button>
                  </span>
                </div>

                {statsFor === link.shortCode && (
                  <div style={styles.statsPanel}>
                    {statsError ? (
                      <span style={styles.error}>{statsError}</span>
                    ) : !stats ? (
                      <span style={styles.hint}>Loading…</span>
                    ) : (
                      <div style={styles.statsGrid}>
                        <div>
                          <div style={styles.statsLabel}>TOTAL CLICKS</div>
                          <div style={styles.statsValue} className="mono">
                            {stats.total_clicks}
                          </div>
                        </div>
                        <div>
                          <div style={styles.statsLabel}>CREATED</div>
                          <div style={styles.statsValue} className="mono">
                            {new Date(stats.created_at).toLocaleString()}
                          </div>
                        </div>
                        <div>
                          <div style={styles.statsLabel}>LAST CLICKED</div>
                          <div style={styles.statsValue} className="mono">
                            {stats.last_clicked
                              ? new Date(stats.last_clicked).toLocaleString()
                              : '—'}
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            ))
          )}
        </section>
      </div>
    </main>
  );
}

const styles: Record<string, React.CSSProperties> = {
  page: {
    minHeight: '100vh',
    padding: '0 24px',
  },
  shell: {
    maxWidth: 920,
    margin: '0 auto',
    padding: '48px 0 96px',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 40,
    paddingBottom: 20,
    borderBottom: '1px solid var(--line)',
  },
  brand: {
    fontFamily: 'var(--mono)',
    color: 'var(--signal)',
    fontSize: 14,
  },
  headerRight: {
    display: 'flex',
    alignItems: 'center',
    gap: 16,
  },
  emailTag: {
    fontSize: 13,
    color: 'var(--text-dim)',
  },
  linkButton: {
    background: 'none',
    border: 'none',
    color: 'var(--text-dim)',
    fontSize: 13,
    cursor: 'pointer',
    textDecoration: 'underline',
  },
  createSection: {
    marginBottom: 48,
  },
  createForm: {
    display: 'flex',
    gap: 10,
  },
  urlInput: {
    flex: 1,
    background: 'var(--surface)',
    border: '1px solid var(--line-strong)',
    color: 'var(--text)',
    padding: '12px 14px',
    fontSize: 14,
  },
  button: {
    background: 'var(--signal)',
    color: 'var(--ink)',
    border: 'none',
    padding: '0 20px',
    fontSize: 14,
    fontWeight: 600,
    cursor: 'pointer',
  },
  error: {
    color: 'var(--error)',
    fontSize: 13,
    marginTop: 10,
  },
  tableHeader: {
    display: 'flex',
    gap: 12,
    fontSize: 11,
    letterSpacing: '0.04em',
    color: 'var(--text-dim)',
    borderBottom: '1px solid var(--line)',
    paddingBottom: 10,
    marginBottom: 4,
  },
  row: {
    display: 'flex',
    gap: 12,
    alignItems: 'center',
    padding: '14px 0',
    borderBottom: '1px solid var(--line)',
  },
  cell: {
    fontSize: 13.5,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  miniButton: {
    background: 'transparent',
    border: '1px solid var(--line-strong)',
    color: 'var(--text)',
    fontSize: 12,
    padding: '5px 10px',
    cursor: 'pointer',
  },
  empty: {
    padding: '40px 0',
    color: 'var(--text-dim)',
    fontSize: 14,
    textAlign: 'center',
  },
  statsPanel: {
    background: 'var(--surface)',
    border: '1px solid var(--line)',
    padding: '16px 20px',
    marginBottom: 4,
  },
  statsGrid: {
    display: 'flex',
    gap: 48,
  },
  statsLabel: {
    fontSize: 10,
    letterSpacing: '0.04em',
    color: 'var(--text-dim)',
    marginBottom: 4,
  },
  statsValue: {
    fontSize: 15,
    color: 'var(--signal)',
  },
  authBox: {
    maxWidth: 380,
    margin: '120px auto',
    background: 'var(--surface)',
    border: '1px solid var(--line)',
    padding: 32,
  },
  authTitle: {
    fontSize: 18,
    fontWeight: 600,
    margin: '20px 0 8px',
  },
  hint: {
    fontSize: 13,
    color: 'var(--text-dim)',
    marginBottom: 20,
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: 6,
  },
  label: {
    fontSize: 12,
    color: 'var(--text-dim)',
    marginTop: 8,
  },
  input: {
    background: 'var(--ink)',
    border: '1px solid var(--line-strong)',
    color: 'var(--text)',
    padding: '10px 12px',
    fontSize: 14,
    marginBottom: 8,
  },
};
