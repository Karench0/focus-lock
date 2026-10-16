import { Maximize2, Minus, Minimize2, X } from "lucide-react";
import { beginWindowDrag } from "./windowDrag";
import { copy } from "./i18n";

export function WindowBar({
  onError,
  language,
  canCompact,
  busy,
  onCompact,
  onMinimise,
  onMaximise,
  onClose,
}: {
  onError: (error: string) => void;
  language: "ru" | "en";
  canCompact: boolean;
  busy: boolean;
  onCompact: () => void;
  onMinimise: () => void;
  onMaximise: () => void;
  onClose: () => void;
}) {
  const t = copy[language];
  return (
    <header className="window-bar">
      <div
        className="window-drag"
        onPointerDown={(e) => beginWindowDrag(e, onError)}
      >
        <span className="window-logo" /> Focus Lock
      </div>
      <button
        className="window-mini"
        onClick={onCompact}
        disabled={!canCompact || busy}
        title={canCompact ? t.miniTimer : t.miniStartFirst}
        aria-label={t.miniTimer}
      >
        <Minimize2 size={14} />
        <span>{t.miniTimer}</span>
      </button>
      <div className="window-controls">
        <button
          title={t.minimiseWindow}
          aria-label={t.minimiseWindow}
          onClick={onMinimise}
        >
          <Minus size={14} />
        </button>
        <button
          title={t.maximiseWindow}
          aria-label={t.maximiseWindow}
          onClick={onMaximise}
        >
          <Maximize2 size={12} />
        </button>
        <button
          className="window-close"
          title={t.closeWindow}
          aria-label={t.closeWindow}
          onClick={onClose}
        >
          <X size={15} />
        </button>
      </div>
    </header>
  );
}
