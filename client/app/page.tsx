'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { registerUser, ApiError } from '@/lib/api';
import { saveSession } from '@/lib/storage';

const STATS = [
  { value: '41.4k', label: 'redirects / sec, single node' },
  { value: '2.1ms', label: 'p99 redirect latency' },
  { value: '95%+', label: 'target cache hit ratio' },
];

export default function HomePage() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [apiKey, setApiKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);

  async function handleRegister(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await registerUser(email);
      setApiKey(res.api_key);
      saveSession(res.email, res.api_key);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong.');
    } finally {
      setLoading(false);
    }
  }

  function handleContinue() {
    router.push('/dashboard');
  }

  async function handleCopy() {
    if (!apiKey) return;
    await navigator.clipboard.writeText(apiKey);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  return (
    <main style={styles.page}>
      <div style={styles.grid}>
        <section style={styles.hero}>
          <div style={styles.brand}>shrtn</div>
          <h1 style={styles.headline}>
            Short links, built to survive real traffic.
          </h1>
          <p style={styles.subhead}>
            Snowflake IDs, Redis cache-aside, horizontal scaling
            coordinated through etcd, and an analytics pipeline that
            never touches the redirect path. This dashboard talks
            directly to that API.
          </p>

          <dl style={styles.statRow}>
            {STATS.map((s) => (
              <div key={s.label} style={styles.statItem}>
                <dt style={styles.statValue} className="mono">
                  {s.value}
                </dt>
                <dd style={styles.statLabel}>{s.label}</dd>
              </div>
            ))}
          </dl>
        </section>

        <section style={styles.panel}>
          {!apiKey ? (
            <>
              <h2 style={styles.panelTitle}>Get an API key</h2>
              <p style={styles.panelHint}>
                No password. Your key is shown once — save it somewhere
                safe.
              </p>
              <form onSubmit={handleRegister} style={styles.form}>
                <label style={styles.label} htmlFor="email">
                  Email
                </label>
                <input
                  id="email"
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="you@domain.com"
                  style={styles.input}
                />
                {error && <div style={styles.error}>{error}</div>}
                <button type="submit" disabled={loading} style={styles.button}>
                  {loading ? 'Creating key…' : 'Create API key'}
                </button>
              </form>

              <div style={styles.divider} />

              <p style={styles.panelHint}>
                Already have a key?{' '}
                <a href="/dashboard">Go to the dashboard</a> and paste it
                there.
              </p>
            </>
          ) : (
            <>
              <h2 style={styles.panelTitle}>Your API key</h2>
              <p style={styles.panelHint}>
                This is shown once. It won&rsquo;t be retrievable again —
                copy it now.
              </p>
              <div style={styles.keyBox} className="mono">
                {apiKey}
              </div>
              <button onClick={handleCopy} style={styles.buttonSecondary}>
                {copied ? 'Copied' : 'Copy key'}
              </button>
              <button onClick={handleContinue} style={styles.button}>
                Continue to dashboard
              </button>
            </>
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
    display: 'flex',
    alignItems: 'center',
  },
  grid: {
    width: '100%',
    maxWidth: 1080,
    margin: '0 auto',
    display: 'grid',
    gridTemplateColumns: 'minmax(0, 1.3fr) minmax(280px, 380px)',
    gap: 64,
    alignItems: 'start',
    padding: '96px 0',
  },
  brand: {
    fontFamily: 'var(--mono)',
    color: 'var(--signal)',
    fontSize: 14,
    letterSpacing: '0.02em',
    marginBottom: 28,
  },
  headline: {
    fontSize: 42,
    lineHeight: 1.15,
    fontWeight: 600,
    margin: '0 0 20px',
    maxWidth: 560,
    color: 'var(--text)',
  },
  subhead: {
    fontSize: 16,
    lineHeight: 1.6,
    color: 'var(--text-dim)',
    maxWidth: 480,
    margin: '0 0 48px',
  },
  statRow: {
    display: 'flex',
    gap: 40,
    margin: 0,
    borderTop: '1px solid var(--line)',
    paddingTop: 24,
  },
  statItem: {
    display: 'flex',
    flexDirection: 'column',
    gap: 6,
  },
  statValue: {
    fontSize: 22,
    color: 'var(--signal)',
    margin: 0,
  },
  statLabel: {
    fontSize: 12,
    color: 'var(--text-dim)',
    margin: 0,
    maxWidth: 120,
    lineHeight: 1.4,
  },
  panel: {
    background: 'var(--surface)',
    border: '1px solid var(--line)',
    padding: 32,
  },
  panelTitle: {
    fontSize: 18,
    fontWeight: 600,
    margin: '0 0 8px',
  },
  panelHint: {
    fontSize: 13,
    color: 'var(--text-dim)',
    lineHeight: 1.6,
    margin: '0 0 20px',
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: 8,
  },
  label: {
    fontSize: 12,
    color: 'var(--text-dim)',
  },
  input: {
    background: 'var(--ink)',
    border: '1px solid var(--line-strong)',
    color: 'var(--text)',
    padding: '10px 12px',
    fontSize: 14,
    marginBottom: 12,
    fontFamily: 'var(--sans)',
  },
  button: {
    background: 'var(--signal)',
    color: 'var(--ink)',
    border: 'none',
    padding: '11px 16px',
    fontSize: 14,
    fontWeight: 600,
    cursor: 'pointer',
    width: '100%',
    marginTop: 4,
  },
  buttonSecondary: {
    background: 'transparent',
    color: 'var(--text)',
    border: '1px solid var(--line-strong)',
    padding: '11px 16px',
    fontSize: 14,
    cursor: 'pointer',
    width: '100%',
    marginBottom: 10,
  },
  keyBox: {
    background: 'var(--ink)',
    border: '1px solid var(--line-strong)',
    padding: '14px 12px',
    fontSize: 13,
    wordBreak: 'break-all',
    marginBottom: 16,
    color: 'var(--signal)',
  },
  error: {
    color: 'var(--error)',
    fontSize: 13,
    marginBottom: 8,
  },
  divider: {
    height: 1,
    background: 'var(--line)',
    margin: '24px 0',
  },
};