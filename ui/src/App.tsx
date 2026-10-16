import { useCallback, useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import { AnimatePresence, motion } from "framer-motion";
import {
  ArrowRight,
  BarChart3,
  Check,
  CheckCheck,
  ChevronRight,
  Clock3,
  Code2,
  Copy,
  ExternalLink,
  Globe2,
  LayoutDashboard,
  LockKeyhole,
  Monitor,
  Plus,
  RefreshCw,
  Search,
  Settings2,
  ShieldCheck,
  ShieldOff,
  SlidersHorizontal,
  Trash2,
  X,
  Zap,
} from "lucide-react";
import type { Application, Page, Profile, Settings, State } from "./types";
import { call } from "./api";
import { copy } from "./i18n";

const pages: Page[] = [
  "dashboard",
  "applications",
  "websites",
  "profiles",
  "statistics",
  "settings",
];
const pageIcons = [
  LayoutDashboard,
  Monitor,
  Globe2,
  SlidersHorizontal,
  BarChart3,
  Settings2,
];
const displayPath = (path: string) =>
  path.replace(/([A-Z]:\\Users\\)[^\\]+/i, "$1…");
const formatTime = (s: number) =>
  [Math.floor(s / 3600), Math.floor((s % 3600) / 60), s % 60]
    .map((n) => String(Math.max(0, n)).padStart(2, "0"))
    .join(":");
function Mark() {
  return (
    <svg viewBox="0 0 40 40" fill="none" aria-hidden="true">
      <path
        d="M14 5H7a2 2 0 0 0-2 2v7m21-9h7a2 2 0 0 1 2 2v7M5 26v7a2 2 0 0 0 2 2h7m12 0h7a2 2 0 0 0 2-2v-7"
        stroke="currentColor"
        strokeWidth="2.5"
      />
      <rect x="13" y="17" width="14" height="12" rx="3" fill="currentColor" />
      <path
        d="M16 17v-4a4 4 0 0 1 8 0v4"
        stroke="currentColor"
        strokeWidth="2.5"
      />
    </svg>
  );
}
function Modal({
  children,
  onClose,
  label,
}: {
  children: ReactNode;
  onClose?: () => void;
  label: string;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const closeRef = useRef(onClose);
  closeRef.current = onClose;
  useEffect(() => {
    const before = document.activeElement as HTMLElement;
    const root = ref.current;
    const first = root?.querySelector<HTMLElement>("button,input,select");
    first?.focus();
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape" && closeRef.current) {
        e.preventDefault();
        closeRef.current();
      }
      if (e.key === "Tab" && root) {
        const els = Array.from(
          root.querySelectorAll<HTMLElement>(
            'button:not(:disabled),input:not(:disabled),select:not(:disabled),[tabindex="0"]',
          ),
        );
        if (!els.length) return;
        const first = els[0],
          last = els[els.length - 1];
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    };
    document.addEventListener("keydown", handler);
    return () => {
      document.removeEventListener("keydown", handler);
      before?.focus();
    };
  }, []);
  return (
    <motion.div
      className="overlay"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
    >
      <motion.div
        ref={ref}
        role="dialog"
        aria-modal="true"
        aria-label={label}
        className="modal"
        initial={{ y: 12, scale: 0.98 }}
        animate={{ y: 0, scale: 1 }}
        transition={{ duration: 0.16 }}
      >
        {onClose && (
          <button
            className="icon-button modal-close"
            aria-label="Close"
            onClick={onClose}
          >
            <X size={18} />
          </button>
        )}
        {children}
      </motion.div>
    </motion.div>
  );
}
function Toggle({
  on,
  disabled,
  label,
  onClick,
}: {
  on: boolean;
  disabled?: boolean;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      className={`toggle ${on ? "on" : ""}`}
      role="switch"
      aria-checked={on}
      aria-label={label}
      disabled={disabled}
      onClick={onClick}
    >
      <span />
    </button>
  );
}
function Empty({
  icon,
  title,
  text,
}: {
  icon: ReactNode;
  title: string;
  text?: string;
}) {
  return (
    <div className="empty">
      <div className="empty-icon">{icon}</div>
      <strong>{title}</strong>
      {text && <p>{text}</p>}
    </div>
  );
}

export function App() {
  const [data, setData] = useState<State>();
  const [page, setPage] = useState<Page>("dashboard");
  const [profileID, setProfileID] = useState(
    localStorage.getItem("profile") || "",
  );
  const [apps, setApps] = useState<Application[]>([]);
  const [scanning, setScanning] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [minutes, setMinutes] = useState(25);
  const [search, setSearch] = useState("");
  const [domain, setDomain] = useState("");
  const [subdomains, setSubdomains] = useState(false);
  const [modal, setModal] = useState<
    "exit" | "close" | "preflight" | "profile" | "delete" | null
  >(null);
  const [preflight, setPreflight] = useState<
    { pid: number; name: string; path: string }[]
  >([]);
  const [profileName, setProfileName] = useState("");
  const [editID, setEditID] = useState("");
  const [step, setStep] = useState(0);
  const [period, setPeriod] = useState(7);
  const [dismissedSession, setDismissedSession] = useState("");
  const t = copy[data?.settings.language || "ru"];
  const active =
    data?.session.state === "ACTIVE" || data?.session.state === "EXIT_PENDING";
  const pending = data?.session.state === "EXIT_PENDING";
  const profile =
    data?.profiles.find((p) => p.id === profileID) || data?.profiles[0];
  const sessionProfile = active ? data?.session.profile : profile;
  const browserReady = data?.browsers.some((b) => b.connected);
  const refresh = useCallback(async () => {
    try {
      setData(await call<State>("state"));
    } catch (e) {
      setError(String(e));
    }
  }, []);
  useEffect(() => {
    void refresh();
    const timer = setInterval(() => void refresh(), 1000);
    return () => clearInterval(timer);
  }, [refresh]);
  useEffect(() => {
    document.documentElement.lang = data?.settings.language || "ru";
  }, [data?.settings.language]);
  useEffect(() => {
    if (data?.session.state === "FINISHED")
      setModal((m) =>
        m === "close" || m === "exit" || m === "preflight" ? null : m,
      );
  }, [data?.session.state]);
  useEffect(() => {
    const off = window.runtime?.EventsOn("close-request", () =>
      setModal("close"),
    );
    return () => off?.();
  }, []);
  useEffect(() => {
    if (!notice) return;
    const timer = setTimeout(() => setNotice(""), 2400);
    return () => clearTimeout(timer);
  }, [notice]);
  const run = async (fn: () => Promise<void>) => {
    setBusy(true);
    setError("");
    try {
      await fn();
      await refresh();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };
  const selectProfile = (id: string) => {
    setProfileID(id);
    localStorage.setItem("profile", id);
  };
  const saveProfile = async (p: Profile) => {
    await call("saveProfile", p);
  };
  const updateProfile = (p: Partial<Profile>) => {
    if (profile) void run(async () => saveProfile({ ...profile, ...p }));
  };
  const saveSettings = (v: Partial<Settings>) => {
    if (data)
      void run(async () => {
        await call("saveSettings", { ...data.settings, ...v });
      });
  };
  const scan = async () => {
    setScanning(true);
    try {
      setApps(await call<Application[]>("discover"));
    } catch (e) {
      setError(String(e));
    } finally {
      setScanning(false);
    }
  };
  const go = (p: Page) => {
    setPage(p);
    if (p === "applications" && apps.length === 0) void scan();
  };
  const addExe = () =>
    void run(async () => {
      const a = await call<Application | null>("pickApplication");
      if (
        a &&
        profile &&
        !profile.applications.some(
          (x) => x.path.toLowerCase() === a.path.toLowerCase(),
        )
      ) {
        await saveProfile({
          ...profile,
          applications: [...profile.applications, a],
        });
        setApps((prev) =>
          prev.some((x) => x.path === a.path) ? prev : [a, ...prev],
        );
      }
    });
  const toggleApp = (a: Application) => {
    if (!profile) return;
    const has = profile.applications.some(
      (x) => x.path.toLowerCase() === a.path.toLowerCase(),
    );
    updateProfile({
      applications: has
        ? profile.applications.filter(
            (x) => x.path.toLowerCase() !== a.path.toLowerCase(),
          )
        : [...profile.applications, a],
    });
  };
  const addSite = () => {
    if (!profile || !domain.trim()) return;
    void run(async () => {
      await saveProfile({
        ...profile,
        websites: [...profile.websites, { domain: domain.trim(), subdomains }],
      });
      setDomain("");
    });
  };
  const start = () => {
    if (!profile) return;
    void run(async () => {
      const ps = await call<typeof preflight>("preflight", { id: profile.id });
      setPreflight(ps);
      if (ps.length) {
        setModal("preflight");
        return;
      }
      await call("start", { id: profile.id, minutes });
      setModal(null);
      setDismissedSession("");
    });
  };
  const editProfile = (p?: Profile) => {
    setEditID(p?.id || "");
    setProfileName(p?.name || "");
    setModal("profile");
  };
  const confirmProfile = () =>
    void run(async () => {
      const existing = data?.profiles.find((p) => p.id === editID);
      await saveProfile(
        existing
          ? { ...existing, name: profileName }
          : {
              id: "",
              name: profileName,
              mode: "strict",
              applications: [],
              websites: [],
            },
      );
      setModal(null);
    });
  const appList = [
    ...(profile?.applications || []).map((a) => ({
      ...a,
      icon:
        a.icon ||
        apps.find((x) => x.path.toLowerCase() === a.path.toLowerCase())?.icon,
    })),
    ...apps.filter(
      (a) =>
        !profile?.applications.some(
          (x) => x.path.toLowerCase() === a.path.toLowerCase(),
        ),
    ),
  ].filter((a) =>
    `${a.name} ${a.path}`.toLowerCase().includes(search.toLowerCase()),
  );
  const applicationsPanel = (
    <>
      <div className="toolbar">
        <div className="search">
          <Search size={16} />
          <input
            aria-label={t.search}
            placeholder={t.search}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <button
          className="secondary"
          onClick={() => void scan()}
          disabled={scanning}
        >
          <RefreshCw size={15} className={scanning ? "spin" : ""} />
          {t.scan}
        </button>
        <button className="primary" onClick={addExe} disabled={busy || active}>
          <Plus size={16} />
          {t.addExe}
        </button>
      </div>
      <div className="hint">
        <ShieldCheck size={16} />
        {t.browserWarning}
      </div>
      <div className="list-card app-list">
        {scanning && <p className="muted center">{t.scanning}</p>}
        {appList.length === 0 && !scanning ? (
          <Empty
            icon={<Monitor size={23} />}
            title={search ? t.noResults : t.appEmpty}
            text={t.appEmptyHint}
          />
        ) : (
          appList.map((a) => {
            const allowed = profile?.applications.find(
              (x) => x.path.toLowerCase() === a.path.toLowerCase(),
            );
            return (
              <div className="app-row" key={a.path}>
                <div className="app-icon">
                  {a.icon ? <img src={a.icon} alt="" /> : <Monitor size={20} />}
                </div>
                <div className="row-body">
                  <strong>{a.name}</strong>
                  <span className="path" title={a.path}>
                    {displayPath(a.path)}
                  </span>
                  {allowed && (
                    <label className="child-option">
                      <input
                        type="checkbox"
                        checked={allowed.allowChildren}
                        disabled={busy || active}
                        onChange={() =>
                          updateProfile({
                            applications: profile!.applications.map((x) =>
                              x.path === allowed.path
                                ? { ...x, allowChildren: !x.allowChildren }
                                : x,
                            ),
                          })
                        }
                      />
                      {t.children}
                    </label>
                  )}
                </div>
                <Toggle
                  on={!!allowed}
                  disabled={busy || active}
                  label={`${a.name}: ${t.applications}`}
                  onClick={() => toggleApp(a)}
                />
              </div>
            );
          })
        )}
      </div>
    </>
  );
  const websitePanel = (
    <>
      <div className="mode-selector">
        <button
          className={profile?.mode === "strict" ? "selected" : ""}
          disabled={active || busy}
          onClick={() => updateProfile({ mode: "strict" })}
        >
          <ShieldCheck size={17} />
          {t.strict}
          <span>A</span>
        </button>
        <button
          className={profile?.mode === "blocklist" ? "selected" : ""}
          disabled={active || busy}
          onClick={() => updateProfile({ mode: "blocklist" })}
        >
          <ShieldOff size={17} />
          {t.blocklist}
          <span>B</span>
        </button>
      </div>
      <form
        className="website-form"
        onSubmit={(e) => {
          e.preventDefault();
          addSite();
        }}
      >
        <div className="domain-input">
          <Globe2 size={18} />
          <input
            aria-label={t.domain}
            placeholder={t.domain}
            value={domain}
            disabled={active}
            onChange={(e) => setDomain(e.target.value)}
          />
        </div>
        <button className="primary" disabled={busy || active || !domain.trim()}>
          <Plus size={16} />
          {t.add}
        </button>
      </form>
      <label className="check-label">
        <input
          type="checkbox"
          checked={subdomains}
          disabled={active}
          onChange={(e) => setSubdomains(e.target.checked)}
        />
        {t.subdomains}
      </label>
      <div className="list-card website-list">
        {!profile?.websites.length ? (
          <Empty
            icon={<Globe2 size={23} />}
            title={t.siteEmpty}
            text={t.siteEmptyHint}
          />
        ) : (
          profile.websites.map((w) => (
            <div className="site-row" key={w.domain}>
              <div className="site-icon">
                <Globe2 size={18} />
              </div>
              <div className="row-body">
                <strong>{w.domain}</strong>
                <span>{w.subdomains ? t.subdomains : "Exact domain"}</span>
              </div>
              <button
                aria-label={`${t.remove} ${w.domain}`}
                className="icon-button"
                disabled={busy || active}
                onClick={() =>
                  updateProfile({
                    websites: profile.websites.filter(
                      (x) => x.domain !== w.domain,
                    ),
                  })
                }
              >
                <Trash2 size={16} />
              </button>
            </div>
          ))
        )}
      </div>
      <div className="hint">
        <Code2 size={18} />
        {t.dependencies}
      </div>
    </>
  );
  const extensionPanel = (
    <div className="settings-section">
      <div className="section-heading">
        <Globe2 size={20} />
        <h3>{t.extension}</h3>
        <span className={`badge ${browserReady ? "good" : ""}`}>
          {browserReady ? t.extensionReady : t.extensionMissing}
        </span>
      </div>
      <p>{t.extensionSteps}</p>
      <div className="button-row">
        <button
          className="secondary"
          disabled={busy || active || data?.preview}
          onClick={() =>
            void run(async () => {
              await call("registerBridge");
              setNotice(t.saved);
            })
          }
        >
          <CheckCheck size={16} />
          {t.register}
        </button>
        <button
          className="secondary"
          disabled={data?.preview}
          onClick={() =>
            void run(async () => {
              await call("openExtension");
            })
          }
        >
          <ExternalLink size={16} />
          {t.openExtension}
        </button>
      </div>
      {data?.browsers.map((b) => (
        <div className="connection" key={`${b.browser}${b.pid}`}>
          <span className={b.connected ? "status-dot blue" : "status-dot"} />
          {b.browser} <span className="muted">PID {b.pid}</span>
        </div>
      ))}
    </div>
  );
  const delayPanel = (
    <div className="settings-section">
      <div className="section-heading">
        <Clock3 size={20} />
        <h3>{t.delay}</h3>
      </div>
      <p>{t.delayHint}</p>
      <div className="chips">
        {[0, 15, 30, 60, 120, 300].map((n) => (
          <button
            key={n}
            className={data?.settings.exitDelaySeconds === n ? "selected" : ""}
            disabled={active || busy}
            onClick={() => saveSettings({ exitDelaySeconds: n })}
          >
            {n === 0 ? t.noDelay : `${n} ${t.sec}`}
          </button>
        ))}
      </div>
      <label className="custom-value">
        <input
          aria-label={t.delay}
          type="number"
          min={0}
          max={3600}
          key={data?.settings.exitDelaySeconds}
          defaultValue={data?.settings.exitDelaySeconds}
          disabled={active || busy}
          onBlur={(e) => {
            const v = Number(e.target.value);
            if (v !== data?.settings.exitDelaySeconds)
              saveSettings({ exitDelaySeconds: v });
          }}
        />
        {t.sec}
      </label>
    </div>
  );
  if (!data)
    return (
      <div className="loading">
        <div className="brand-mark">
          <Mark />
        </div>
        <h2>{t.loading}</h2>
        {error && (
          <>
            <p>{error}</p>
            <button className="primary" onClick={() => void refresh()}>
              {t.retryLoad}
            </button>
          </>
        )}
      </div>
    );
  const onboarding = !data.settings.onboarded;
  const finished =
    data.session.state === "FINISHED" && dismissedSession !== data.session.id;
  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark">
            <Mark />
          </div>
          <div>
            focus<span>lock</span>
            <small>OWN YOUR ATTENTION</small>
          </div>
        </div>
        <div className="nav-label">{t.workspace}</div>
        <nav>
          {pages.map((p, i) => {
            const Icon = pageIcons[i];
            return (
              <button
                key={p}
                className={page === p ? "nav-item current" : "nav-item"}
                onClick={() => go(p)}
              >
                <Icon size={18} />
                <span>{t[p]}</span>
                {p === "dashboard" && active && (
                  <span className="status-dot blue" />
                )}
              </button>
            );
          })}
        </nav>
        <div className="sidebar-bottom">
          <div className="side-note">
            <LockKeyhole size={15} />
            {t.local}
          </div>
          <span className="version">v0.1.0 · Windows</span>
        </div>
      </aside>
      <main>
        <div className="topbar">
          <div className="breadcrumb">
            Focus Lock <ChevronRight size={13} />
            <span>{t[page]}</span>
          </div>
          <div className="top-status">
            <span className={`status-dot ${active ? "blue" : ""}`} />
            {active ? t.activeLabel : t.ready}
          </div>
        </div>
        <div className="main-scroll">
          {data.preview && (
            <div className="preview-banner">
              <Code2 size={14} />
              {t.preview}
            </div>
          )}
          {(error || data.error) && (
            <div className="error-banner" role="alert">
              <span>{error || data.error}</span>
              <button
                aria-label={t.clearError}
                className="icon-button"
                onClick={() => {
                  setError("");
                  void call("clearError");
                }}
              >
                <X size={16} />
              </button>
            </div>
          )}
          <AnimatePresence mode="wait">
            <motion.div
              className={`page page-${page}`}
              key={page}
              initial={{ opacity: 0, y: 5 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.14 }}
            >
              <header className="page-header">
                <div>
                  <div className="eyebrow">
                    {page === "dashboard"
                      ? "FOCUS, WITH INTENTION"
                      : "FOCUS LOCK"}
                  </div>
                  <h1>{page === "dashboard" ? t.tagline : t[page]}</h1>
                  <p>
                    {page === "dashboard"
                      ? t.today
                      : page === "applications"
                        ? t.appSubtitle
                        : page === "websites"
                          ? t.siteSubtitle
                          : page === "profiles"
                            ? t.profileSubtitle
                            : page === "statistics"
                              ? t.statsSubtitle
                              : t.settingsSubtitle}
                  </p>
                </div>
                {["applications", "websites"].includes(page) && (
                  <select
                    className="profile-select compact"
                    aria-label={t.profile}
                    disabled={active}
                    value={profile?.id || ""}
                    onChange={(e) => selectProfile(e.target.value)}
                  >
                    {data.profiles.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.name}
                      </option>
                    ))}
                  </select>
                )}
              </header>
              {active && page !== "dashboard" && (
                <div className="lock-banner">
                  <LockKeyhole size={15} />
                  {t.locked}
                </div>
              )}
              {page === "dashboard" && (
                <>
                  <div
                    className={`dashboard-grid ${active ? "session-active" : ""}`}
                  >
                    <section className="timer-card">
                      <div className="card-top">
                        <span className="eyebrow">
                          <span
                            className={`status-dot ${active ? "blue" : ""}`}
                          />
                          {active ? t.focus : t.ready}
                        </span>
                        <span className="tiny-tag">{sessionProfile?.name}</span>
                      </div>
                      <div className="timer-area">
                        <svg
                          className="timer-ring"
                          viewBox="0 0 360 360"
                          aria-hidden="true"
                        >
                          <circle
                            className="ring-track"
                            cx="180"
                            cy="180"
                            r="156"
                          />
                          {Array.from({ length: 60 }, (_, i) => (
                            <line
                              key={i}
                              x1="180"
                              y1="12"
                              x2="180"
                              y2={i % 5 === 0 ? 21 : 17}
                              transform={`rotate(${i * 6} 180 180)`}
                              className={i % 5 === 0 ? "tick major" : "tick"}
                            />
                          ))}
                          <circle
                            className={`ring-progress ${active ? "is-active" : ""}`}
                            cx="180"
                            cy="180"
                            r="156"
                            strokeDasharray={980.18}
                            strokeDashoffset={
                              active ? 980.18 * data.session.progress : 0
                            }
                            transform="rotate(-90 180 180)"
                          />
                        </svg>
                        <div className="timer-value">
                          <LockKeyhole size={19} />
                          <div
                            className="digits"
                            role="timer"
                            aria-label={t.remaining}
                          >
                            {formatTime(
                              active
                                ? data.session.remainingSeconds
                                : minutes * 60,
                            )}
                          </div>
                          <span>{active ? t.remaining : t.duration}</span>
                        </div>
                      </div>
                      {active ? (
                        <>
                          <div className="session-progress">
                            <span>
                              {Math.round(data.session.progress * 100)}%
                            </span>
                            <div>
                              <i
                                style={{
                                  width: `${data.session.progress * 100}%`,
                                }}
                              />
                            </div>
                            <span>
                              {Math.round(data.session.durationSeconds / 60)}{" "}
                              {t.minutes}
                            </span>
                          </div>
                          <button
                            className="early-button"
                            onClick={() => setModal("exit")}
                            disabled={pending}
                          >
                            <X size={15} />
                            {t.early}
                          </button>
                        </>
                      ) : (
                        <>
                          <div className="duration-chips">
                            {[15, 25, 30, 45, 60, 90, 120].map((n) => (
                              <button
                                key={n}
                                className={minutes === n ? "selected" : ""}
                                onClick={() => setMinutes(n)}
                              >
                                {n}
                              </button>
                            ))}
                            <label>
                              <input
                                aria-label={t.duration}
                                type="number"
                                min="1"
                                max="1440"
                                value={minutes}
                                onChange={(e) =>
                                  setMinutes(Number(e.target.value))
                                }
                              />
                              <span>{t.minutes}</span>
                            </label>
                          </div>
                          <button
                            className="primary start-button"
                            disabled={busy || minutes < 1 || minutes > 1440}
                            onClick={start}
                          >
                            <LockKeyhole size={17} />
                            {t.start}
                            <ArrowRight size={18} />
                          </button>
                        </>
                      )}
                    </section>
                    <div className="dashboard-right">
                      <section className="config-card">
                        <div className="section-heading">
                          <SlidersHorizontal size={17} />
                          <h3>{t.profile}</h3>
                        </div>
                        {active ? (
                          <div className="active-profile">
                            {sessionProfile?.name}
                            <LockKeyhole size={15} />
                          </div>
                        ) : (
                          <select
                            className="profile-select"
                            aria-label={t.profile}
                            value={profile?.id || ""}
                            onChange={(e) => selectProfile(e.target.value)}
                          >
                            {data.profiles.map((p) => (
                              <option key={p.id} value={p.id}>
                                {p.name}
                              </option>
                            ))}
                          </select>
                        )}
                        <div className="summary-row">
                          <div className="summary-icon">
                            <Monitor size={17} />
                          </div>
                          <div>
                            <strong>
                              {sessionProfile?.applications.length || 0}{" "}
                              <span>{t.apps}</span>
                            </strong>
                            <button onClick={() => go("applications")}>
                              {t.editApps}
                              <ChevronRight size={12} />
                            </button>
                          </div>
                        </div>
                        <div className="summary-row">
                          <div className="summary-icon">
                            <Globe2 size={17} />
                          </div>
                          <div>
                            <strong>
                              {sessionProfile?.websites.length || 0}{" "}
                              <span>{t.sites}</span>
                            </strong>
                            <button onClick={() => go("websites")}>
                              {t.editSites}
                              <ChevronRight size={12} />
                            </button>
                          </div>
                        </div>
                        <div className="config-details">
                          <div>
                            <span>{t.mode}</span>
                            <strong>
                              {sessionProfile?.mode === "strict"
                                ? t.strict
                                : t.blocklist}
                            </strong>
                          </div>
                          <div>
                            <span>{t.delay}</span>
                            <strong>
                              {data.settings.exitDelaySeconds} {t.sec}
                            </strong>
                          </div>
                        </div>
                      </section>
                      <button
                        className={`extension-card ${browserReady ? "connected" : ""}`}
                        onClick={() => go("settings")}
                      >
                        <div>
                          <span
                            className={`status-dot ${browserReady ? "blue" : ""}`}
                          />
                          <strong>
                            {browserReady
                              ? t.extensionReady
                              : t.extensionMissing}
                          </strong>
                          <ChevronRight size={15} />
                        </div>
                        <p>{t.extensionHint}</p>
                      </button>
                    </div>
                  </div>
                  <div className="dashboard-footer">
                    <div className="footer-symbol">
                      <ShieldCheck size={20} />
                    </div>
                    <div>
                      <span className="eyebrow">{t.control}</span>
                      <p>{t.controlText}</p>
                    </div>
                    <span className="footer-note">{t.voluntary}</span>
                  </div>
                  {active && (
                    <div className="hint">
                      <Monitor size={17} />
                      {t.appWarn}
                    </div>
                  )}
                </>
              )}
              {page === "applications" && applicationsPanel}
              {page === "websites" && websitePanel}
              {page === "profiles" && (
                <>
                  <div className="toolbar justify-end">
                    <button
                      className="primary"
                      disabled={active || busy}
                      onClick={() => editProfile()}
                    >
                      <Plus size={17} />
                      {t.newProfile}
                    </button>
                  </div>
                  <div className="profiles-grid">
                    {data.profiles.map((p, i) => (
                      <div
                        key={p.id}
                        className={`profile-card ${profile?.id === p.id ? "chosen" : ""}`}
                      >
                        <button
                          className="profile-card-main"
                          onClick={() => !active && selectProfile(p.id)}
                        >
                          <div className="profile-card-icon">
                            {i % 3 === 0 ? (
                              <Code2 size={22} />
                            ) : i % 3 === 1 ? (
                              <Monitor size={22} />
                            ) : (
                              <Zap size={22} />
                            )}
                          </div>
                          {profile?.id === p.id && (
                            <Check className="profile-check" size={17} />
                          )}
                          <h3>{p.name}</h3>
                          <p>
                            {p.applications.length} {t.apps} <span>·</span>{" "}
                            {p.websites.length} {t.sites}
                          </p>
                          <span className="badge">
                            {p.mode === "strict" ? t.strict : t.blocklist}
                          </span>
                        </button>
                        <div className="profile-actions">
                          <button
                            disabled={active || busy}
                            onClick={() => editProfile(p)}
                          >
                            {t.name}
                          </button>
                          <button
                            aria-label={t.duplicate}
                            disabled={active || busy}
                            onClick={() =>
                              void run(async () =>
                                saveProfile({
                                  ...p,
                                  id: "",
                                  name: `${p.name} (2)`,
                                }),
                              )
                            }
                          >
                            <Copy size={15} />
                          </button>
                          <button
                            aria-label={t.remove}
                            disabled={
                              active || busy || data.profiles.length <= 1
                            }
                            onClick={() => {
                              setEditID(p.id);
                              setModal("delete");
                            }}
                          >
                            <Trash2 size={15} />
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </>
              )}
              {page === "statistics" && (
                <>
                  <div className="stats-grid">
                    {[
                      {
                        label: t.total,
                        value: `${Math.floor(data.statistics.seconds / 3600)}h ${Math.floor((data.statistics.seconds % 3600) / 60)}m`,
                        icon: Clock3,
                      },
                      {
                        label: t.complete,
                        value: data.statistics.completed,
                        icon: CheckCheck,
                      },
                      {
                        label: t.interrupted,
                        value: data.statistics.interrupted,
                        icon: Clock3,
                      },
                      {
                        label: t.blocked,
                        value:
                          data.statistics.appBlocks +
                          data.statistics.siteBlocks,
                        icon: ShieldCheck,
                      },
                    ].map((c) => (
                      <div className="stat-card" key={c.label}>
                        <div>
                          <span>{c.label}</span>
                          <c.icon size={17} />
                        </div>
                        <strong>{c.value}</strong>
                      </div>
                    ))}
                  </div>
                  <section className="chart-card">
                    <div className="chart-header">
                      <span className="eyebrow">{t.focusTime}</span>
                      <div className="segmented">
                        {[7, 30].map((n) => (
                          <button
                            key={n}
                            className={period === n ? "selected" : ""}
                            onClick={() => setPeriod(n)}
                          >
                            {n === 7 ? t.days7 : t.days30}
                          </button>
                        ))}
                      </div>
                    </div>
                    <div
                      className="chart"
                      role="img"
                      aria-label={`${t.focusTime}: ${data.statistics.days
                        .slice(-period)
                        .map(
                          (d) =>
                            `${d.date}: ${Math.round(d.seconds / 60)} ${t.minutes}`,
                        )
                        .join(", ")}`}
                    >
                      <div className="chart-grid">
                        <span>120m</span>
                        <span>60m</span>
                        <span>0</span>
                      </div>
                      <div className="bars">
                        {data.statistics.days.slice(-period).map((d) => (
                          <div
                            className="bar-column"
                            key={d.date}
                            title={`${d.date}: ${Math.round(d.seconds / 60)} ${t.minutes}`}
                          >
                            <div className="bar-track">
                              <i
                                style={{
                                  height: `${Math.min(100, (d.seconds / Math.max(7200, ...data.statistics.days.slice(-period).map((d) => d.seconds))) * 100)}%`,
                                }}
                              />
                            </div>
                            <span>
                              {period === 7
                                ? new Date(
                                    d.date + "T12:00:00",
                                  ).toLocaleDateString(data.settings.language, {
                                    weekday: "short",
                                  })
                                : d.date.slice(-2)}
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>
                  </section>
                  <section className="list-card history">
                    <h3>{t.history}</h3>
                    {!data.statistics.history.length ? (
                      <Empty
                        icon={<BarChart3 size={23} />}
                        title={t.noHistory}
                      />
                    ) : (
                      <table>
                        <thead>
                          <tr>
                            <th>{t.profile}</th>
                            <th>{t.date}</th>
                            <th>{t.duration}</th>
                            <th>{t.result}</th>
                          </tr>
                        </thead>
                        <tbody>
                          {data.statistics.history.map((r) => (
                            <tr key={r.id}>
                              <td>{r.profile}</td>
                              <td>
                                {new Date(r.startedAt).toLocaleString(
                                  data.settings.language,
                                )}
                              </td>
                              <td>
                                {Math.floor(r.seconds / 60)}m {r.seconds % 60}s
                              </td>
                              <td>
                                <span
                                  className={`badge ${r.outcome === "completed" ? "good" : ""}`}
                                >
                                  {r.outcome === "completed"
                                    ? t.completedLabel
                                    : r.outcome === "recovered"
                                      ? t.recoveredLabel
                                      : t.interruptedLabel}
                                </span>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    )}
                  </section>
                </>
              )}
              {page === "settings" && (
                <div className="settings-stack">
                  <div className="settings-section language-section">
                    <div>
                      <h3>{t.language}</h3>
                      <p>Русский / English</p>
                    </div>
                    <select
                      aria-label={t.language}
                      disabled={active || busy}
                      value={data.settings.language}
                      onChange={(e) =>
                        saveSettings({
                          language: e.target.value as "ru" | "en",
                        })
                      }
                    >
                      <option value="ru">Русский</option>
                      <option value="en">English</option>
                    </select>
                  </div>
                  {delayPanel}
                  {extensionPanel}
                  <div className="settings-section">
                    <div className="section-heading">
                      <ShieldCheck size={20} />
                      <h3>{t.privacy}</h3>
                    </div>
                    <p>{t.privacyText}</p>
                    <p>{t.voluntary}</p>
                  </div>
                  <div className="settings-bottom">
                    <span>{t.version}</span>
                    <button
                      className="secondary"
                      disabled={busy}
                      onClick={() =>
                        active ? setModal("close") : void call("quit")
                      }
                    >
                      {t.quit}
                    </button>
                  </div>
                </div>
              )}
            </motion.div>
          </AnimatePresence>
        </div>
      </main>
      <AnimatePresence>
        {notice && (
          <motion.div
            role="status"
            className="toast"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0 }}
          >
            <Check size={16} />
            {notice}
          </motion.div>
        )}
      </AnimatePresence>
      <AnimatePresence>
        {modal === "exit" && (
          <Modal label={t.exitTitle} onClose={() => setModal(null)}>
            <div className="modal-symbol">
              <Clock3 size={26} />
            </div>
            <span className="eyebrow">PAUSE & REFLECT</span>
            <h2>{t.exitTitle}</h2>
            <p>{t.exitText}</p>
            <div className="delay-note">
              <Clock3 size={16} />
              {t.delay}: {data.settings.exitDelaySeconds} {t.sec}
            </div>
            <button className="primary full" onClick={() => setModal(null)}>
              {t.keep}
              <ArrowRight size={17} />
            </button>
            <button
              className="text-button danger full"
              disabled={busy}
              onClick={() =>
                void run(async () => {
                  await call("requestExit");
                  setModal(null);
                })
              }
            >
              {t.exitAnyway}
            </button>
          </Modal>
        )}
        {pending && modal !== "exit" && (
          <Modal label={t.pending}>
            <div className="modal-symbol">
              <Clock3 size={26} />
            </div>
            <h2>{t.pending}</h2>
            <div className="exit-countdown" role="timer">
              {formatTime(data.session.exitRemainingSeconds)}
            </div>
            <p>{t.pendingText}</p>
            <button
              className="primary full"
              disabled={busy}
              onClick={() =>
                void run(async () => {
                  await call("cancelExit");
                })
              }
            >
              {t.changedMind}
            </button>
          </Modal>
        )}
        {modal === "close" && !pending && (
          <Modal label={t.closeTitle} onClose={() => setModal(null)}>
            <div className="modal-symbol">
              <LockKeyhole size={26} />
            </div>
            <h2>{t.closeTitle}</h2>
            <p>{t.closeText}</p>
            <button
              className="primary full"
              onClick={() =>
                void run(async () => {
                  await call("tray");
                  setModal(null);
                })
              }
            >
              {t.tray}
            </button>
            <button
              className="text-button full"
              onClick={() => setModal("exit")}
            >
              {t.closeSession}
            </button>
          </Modal>
        )}
        {modal === "preflight" && (
          <Modal label={t.preflightTitle} onClose={() => setModal(null)}>
            <div className="modal-symbol">
              <Monitor size={26} />
            </div>
            <h2>{t.preflightTitle}</h2>
            <p>{t.preflightText}</p>
            <div className="preflight-list">
              {preflight.map((p) => (
                <div key={p.pid}>
                  <strong>{p.name}</strong>
                  <span title={p.path}>{p.path}</span>
                </div>
              ))}
            </div>
            <p className="small muted">{t.appWarn}</p>
            <button className="primary full" disabled={busy} onClick={start}>
              {t.retry}
              <RefreshCw size={16} />
            </button>
          </Modal>
        )}
        {modal === "profile" && (
          <Modal label={t.profile} onClose={() => setModal(null)}>
            <h2>{editID ? t.profile : t.newProfile}</h2>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                confirmProfile();
              }}
            >
              <label className="field-label">
                {t.name}
                <input
                  autoFocus
                  maxLength={80}
                  value={profileName}
                  onChange={(e) => setProfileName(e.target.value)}
                />
              </label>
              <button
                className="primary full"
                disabled={busy || !profileName.trim()}
              >
                {t.save}
              </button>
            </form>
          </Modal>
        )}
        {modal === "delete" && (
          <Modal label={t.removeProfile} onClose={() => setModal(null)}>
            <h2>{t.removeProfile}</h2>
            <div className="button-row">
              <button className="secondary" onClick={() => setModal(null)}>
                {t.cancel}
              </button>
              <button
                className="secondary danger"
                disabled={busy}
                onClick={() =>
                  void run(async () => {
                    await call("deleteProfile", { id: editID });
                    setModal(null);
                  })
                }
              >
                {t.remove}
              </button>
            </div>
          </Modal>
        )}
        {finished && !modal && !onboarding && (
          <Modal
            label={t.finished}
            onClose={() => setDismissedSession(data.session.id)}
          >
            <div className="modal-symbol success">
              <CheckCheck size={28} />
            </div>
            <span className="eyebrow">SESSION COMPLETE</span>
            <h2>{t.finished}</h2>
            <p>{t.finishedText}</p>
            <div className="completion-time">
              {Math.round(
                (data.session.durationSeconds * data.session.progress) / 60,
              )}{" "}
              <span>{t.minutes}</span>
            </div>
            <button
              className="primary full"
              onClick={() => {
                setDismissedSession(data.session.id);
                go("dashboard");
              }}
            >
              {t.newSession}
              <ArrowRight size={17} />
            </button>
            <button
              className="text-button full"
              onClick={() => {
                setDismissedSession(data.session.id);
                go("statistics");
              }}
            >
              {t.seeStats}
            </button>
          </Modal>
        )}
        {onboarding && !modal && (
          <Modal label={t.welcome}>
            <div className={`onboarding step-${step}`}>
              <div className="onboarding-progress">
                {[0, 1, 2, 3, 4].map((n) => (
                  <span key={n} className={n <= step ? "lit" : ""} />
                ))}
              </div>
              <span className="eyebrow">FOCUS LOCK / 0{step + 1}</span>
              <h2>
                {
                  [
                    t.onboardingTitle,
                    t.chooseApps,
                    t.setupBrowser,
                    t.chooseSites,
                    t.setupDelay,
                  ][step]
                }
              </h2>
              {step === 0 ? (
                <>
                  <div className="onboarding-brand">
                    <Mark />
                  </div>
                  <p>{t.onboardingText}</p>
                  <div className="hint">
                    <ShieldCheck size={18} />
                    {t.voluntary}
                  </div>
                </>
              ) : step === 1 ? (
                <>
                  <p>{t.onboardingApps}</p>
                  {applicationsPanel}
                </>
              ) : step === 2 ? (
                extensionPanel
              ) : step === 3 ? (
                <>
                  <p>{t.onboardingSites}</p>
                  {websitePanel}
                </>
              ) : (
                delayPanel
              )}
              <div className="onboarding-footer">
                {step > 0 && (
                  <button
                    className="text-button"
                    onClick={() => setStep(step - 1)}
                  >
                    {t.back}
                  </button>
                )}
                <button
                  className="primary"
                  disabled={busy}
                  onClick={() => {
                    if (step === 4) saveSettings({ onboarded: true });
                    else {
                      setStep(step + 1);
                      if (step === 0 && apps.length === 0) void scan();
                    }
                  }}
                >
                  {step === 4 ? t.done : t.next}
                  <ArrowRight size={16} />
                </button>
              </div>
            </div>
          </Modal>
        )}
      </AnimatePresence>
    </div>
  );
}
