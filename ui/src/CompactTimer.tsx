import { useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import {
  ArrowUpRight,
  Check,
  GripHorizontal,
  Minus,
  ShieldCheck,
  Square,
} from "lucide-react";
import { beginWindowDrag } from "./windowDrag";
import { copy } from "./i18n";
import type { State } from "./types";

const time = (seconds: number) => {
  const s = Math.max(0, seconds);
  return s >= 3600
    ? `${Math.floor(s / 3600)}:${String(Math.floor((s % 3600) / 60)).padStart(2, "0")}:${String(s % 60).padStart(2, "0")}`
    : `${String(Math.floor(s / 60)).padStart(2, "0")}:${String(s % 60).padStart(2, "0")}`;
};

export function CompactTimer({
  data,
  busy,
  error,
  onError,
  onExpand,
  onHide,
  onStop,
  onCancel,
}: {
  onError: (error: string) => void;
  data: State;
  busy: boolean;
  error: string;
  onExpand: () => void;
  onHide: () => void;
  onStop: () => void;
  onCancel: () => void;
}) {
  const t = copy[data.settings.language];
  const reduced = useReducedMotion();
  const [confirm, setConfirm] = useState(false);
  const pending = data.session.state === "EXIT_PENDING";
  const finished = data.session.state === "FINISHED";
  const confirmation = confirm && !pending && !finished;
  return (
    <motion.section
      className={`mini-timer ${pending ? "mini-pending" : ""}`}
      aria-label={t.miniTimer}
      initial={{ opacity: 0, y: reduced ? 0 : 5 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: reduced ? 0 : 0.18 }}
    >
      <header className="mini-header">
        <div
          className="mini-drag"
          title={t.dragTimer}
          onPointerDown={(e) => beginWindowDrag(e, onError)}
          onDoubleClick={(e) => e.stopPropagation()}
        >
          <GripHorizontal size={16} />
          <span>
            focus<span>lock</span>
          </span>
          <span className="mini-state">
            <i />
            {finished ? t.complete : pending ? t.pending : t.activeLabel}
          </span>
        </div>
        <button
          className="mini-icon"
          title={t.expandTimer}
          aria-label={t.expandTimer}
          onClick={onExpand}
          disabled={busy}
        >
          <ArrowUpRight size={17} />
        </button>
        <button
          className="mini-icon"
          title={t.tray}
          aria-label={t.tray}
          onClick={onHide}
          disabled={busy}
        >
          <Minus size={16} />
        </button>
      </header>
      {confirmation ? (
        <div className="mini-confirm">
          <strong>{t.miniStopTitle}</strong>
          <p>
            {t.delay}: {data.settings.exitDelaySeconds} {t.sec}
          </p>
          <div>
            <button
              className="mini-keep"
              onClick={() => setConfirm(false)}
              disabled={busy}
            >
              {t.keep}
            </button>
            <button
              className="mini-end"
              onClick={() => {
                setConfirm(false);
                onStop();
              }}
              disabled={busy}
            >
              {t.miniStop}
            </button>
          </div>
        </div>
      ) : (
        <>
          <div className="mini-body">
            <div className={`mini-emblem ${finished ? "mini-done" : ""}`}>
              {finished ? <Check size={22} /> : <ShieldCheck size={22} />}
            </div>
            <div className="mini-clock">
              <span>
                {finished
                  ? t.finished
                  : pending
                    ? t.miniExitIn
                    : t.miniRemaining}
              </span>
              <strong
                role="timer"
                aria-label={pending ? t.miniExitIn : t.miniRemaining}
              >
                {finished
                  ? "00:00"
                  : time(
                      pending
                        ? data.session.exitRemainingSeconds
                        : data.session.remainingSeconds,
                    )}
              </strong>
            </div>
            {finished ? (
              <button
                className="mini-stop mini-next"
                aria-label={t.expandTimer}
                title={t.expandTimer}
                disabled={busy}
                onClick={onExpand}
              >
                <ArrowUpRight size={21} />
              </button>
            ) : pending ? (
              <button
                className="mini-keep"
                disabled={busy}
                onClick={() => {
                  setConfirm(false);
                  onCancel();
                }}
              >
                {t.miniContinue}
              </button>
            ) : (
              <button
                className="mini-stop"
                title={t.miniStop}
                aria-label={t.miniStop}
                disabled={busy}
                onClick={() => setConfirm(true)}
              >
                <Square size={16} fill="currentColor" />
              </button>
            )}
          </div>
          <footer className="mini-footer">
            <span title={data.session.profile.name}>
              {finished ? t.miniDone : data.session.profile.name}
            </span>
            <span>
              {finished
                ? "✓"
                : pending
                  ? t.miniStillLocked
                  : `${Math.round(data.session.progress * 100)}%`}
            </span>
          </footer>
        </>
      )}
      {error && (
        <button
          className="mini-error"
          role="alert"
          onClick={onExpand}
          title={error}
        >
          {t.miniError}
        </button>
      )}
      <div
        className="mini-progress"
        role="progressbar"
        aria-label={t.focusTime}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(data.session.progress * 100)}
      >
        <i
          style={{ width: `${finished ? 100 : data.session.progress * 100}%` }}
        />
      </div>
    </motion.section>
  );
}
