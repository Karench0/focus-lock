import { useId, useRef, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { BookOpen, Check, Info, ShieldCheck } from "lucide-react";
import "./storozhenko.css";

type Language = "ru" | "en";

export function StorozhenkoSettings({
  language,
  active,
  locked,
  busy,
  onEnable,
  onDisable,
}: {
  language: Language;
  active: boolean;
  locked: boolean;
  busy: boolean;
  onEnable: () => void;
  onDisable: () => void;
}) {
  const ru = language === "ru";
  const [revealed, setRevealed] = useState(false);
  const clicks = useRef<number[]>([]);
  const panelId = useId();
  const reduced = useReducedMotion();
  const visible = revealed || active;

  function reveal() {
    const now = performance.now();
    const previous = clicks.current;
    clicks.current =
      previous.length &&
      now - previous[previous.length - 1] <= 800 &&
      now - previous[0] <= 1600
        ? [...previous, now]
        : [now];
    if (clicks.current.length === 3) {
      setRevealed(true);
      clicks.current = [];
    }
  }

  return (
    <div className="storozhenko-settings">
      {visible && (
        <motion.section
          id={panelId}
          className="settings-section storozhenko-panel"
          aria-label={ru ? "Дополнительные настройки" : "Additional settings"}
          initial={{ opacity: 0, y: reduced ? 0 : 6 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: reduced ? 0 : 0.18 }}
        >
          <div className="storozhenko-panel-heading">
            <span className="storozhenko-book">
              <BookOpen size={18} />
            </span>
            <div>
              <span className="eyebrow">
                {ru ? "ДОПОЛНИТЕЛЬНО" : "ADDITIONAL"}
              </span>
              <h3 id={`${panelId}-label`}>Storozhenko mode</h3>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={active}
              aria-labelledby={`${panelId}-label`}
              aria-describedby={`${panelId}-description`}
              className={`toggle${active ? " on" : ""}`}
              disabled={busy || (!active && locked)}
              onClick={active ? onDisable : onEnable}
            >
              <span />
            </button>
          </div>
          <p id={`${panelId}-description`}>
            {ru
              ? "Личное занятие Kotlin для Стороженко. Включается только после отдельного подтверждения."
              : "A personal Kotlin exercise for Storozhenko. Requires a separate confirmation to start."}
          </p>
          {active ? (
            <div className="storozhenko-panel-note active">
              <Check size={14} />
              <span>
                {ru
                  ? "Занятие идёт. Чтобы закончить, ответьте на один вопрос по Kotlin."
                  : "Exercise in progress. Answer one Kotlin question to finish."}
              </span>
            </div>
          ) : locked ? (
            <div className="storozhenko-panel-note">
              <Info size={14} />
              <span>
                {ru
                  ? "Сначала завершите текущую сессию."
                  : "Finish the current session first."}
              </span>
            </div>
          ) : null}
        </motion.section>
      )}
      <div className="storozhenko-secret-row">
        <button
          type="button"
          className="storozhenko-secret"
          aria-label={
            ru
              ? "Дополнительные настройки: нажмите три раза подряд"
              : "Additional settings: activate three times in a row"
          }
          aria-expanded={visible}
          aria-controls={visible ? panelId : undefined}
          onClick={reveal}
        />
      </div>
    </div>
  );
}

export function StorozhenkoConsent({
  language,
  busy,
  blocked = [],
  browserReady,
  onConfirm,
  onCancel,
}: {
  language: Language;
  busy: boolean;
  blocked?: { name: string; path: string }[];
  browserReady: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  const ru = language === "ru";
  const [agreed, setAgreed] = useState(false);
  return (
    <div className="storozhenko-consent">
      <div className="modal-symbol">
        <ShieldCheck size={26} />
      </div>
      <div className="eyebrow">STOROZHENKO MODE</div>
      <h2>{ru ? "Вы Стороженко?" : "Are you Storozhenko?"}</h2>
      <p>
        {ru
          ? "Это личный режим для занятий Kotlin. Включайте его только для себя и после сохранения работы."
          : "This is a personal Kotlin study mode. Enable it only for yourself, after saving your work."}
      </p>
      <ul className="storozhenko-effects">
        <li>
          {ru
            ? "Будут доступны только сайты и приложения для Kotlin. YouTube, TikTok, Steam и Dota 2 будут заблокированы."
            : "Only Kotlin study websites and apps will be available. YouTube, TikTok, Steam and Dota 2 will be blocked."}
        </li>
        <li>
          {ru
            ? "Другие приложения могут быть закрыты, а открытые посторонние сайты заменены страницей блокировки. Сначала сохраните документы и незавершённую работу."
            : "Other apps may close, and open unrelated websites will be replaced by the blocked page. Save documents and unfinished work first."}
        </li>
        <li>
          {ru
            ? "В браузере по умолчанию откроется документация kotlinlang.org."
            : "Kotlin documentation at kotlinlang.org will open in your default browser."}
        </li>
        <li>
          {ru
            ? "Для обычного завершения нужно правильно ответить на один вопрос по Kotlin из восьми вариантов. Ошибка оставит занятие включённым."
            : "To finish normally, answer one Kotlin question correctly from eight options. An incorrect answer keeps the exercise running."}
        </li>
      </ul>
      {blocked.length > 0 && (
        <details className="storozhenko-preflight">
          <summary>
            {ru
              ? `Открытые приложения, которые будут закрыты: ${blocked.length}`
              : `Open apps that will close: ${blocked.length}`}
          </summary>
          <ul>
            {blocked.map((app, index) => (
              <li key={`${app.path}-${index}`}>
                <strong>{app.name}</strong>
                <span>{app.path}</span>
              </li>
            ))}
          </ul>
        </details>
      )}
      <p className="storozhenko-preserved">
        {ru
          ? "Ваши обычные профили и настройки сохранятся."
          : "Your normal profiles and settings will be preserved."}
      </p>
      {!browserReady && (
        <div role="status" className="storozhenko-browser-warning">
          <Info size={16} />
          <span>
            {ru
              ? "Обновите и подключите расширение Focus Lock в Chrome или Edge, выбранном браузером по умолчанию."
              : "Update and connect the Focus Lock extension in your default Chrome or Edge browser."}
          </span>
        </div>
      )}
      <label className="storozhenko-attestation">
        <input
          type="checkbox"
          checked={agreed}
          disabled={busy}
          onChange={(event) => setAgreed(event.target.checked)}
        />
        <span>
          {ru
            ? "Да, я Стороженко и хочу включить этот режим"
            : "Yes, I am Storozhenko and I want to enable this mode"}
        </span>
      </label>
      <div className="button-row storozhenko-actions">
        <button
          type="button"
          className="secondary"
          disabled={busy}
          onClick={onCancel}
        >
          {ru ? "Отмена" : "Cancel"}
        </button>
        <button
          type="button"
          className="primary"
          disabled={busy || !agreed || !browserReady}
          onClick={onConfirm}
        >
          {busy
            ? ru
              ? "Включаем…"
              : "Enabling…"
            : ru
              ? "Включить режим"
              : "Enable mode"}
        </button>
      </div>
      <p className="storozhenko-emergency">
        {ru
          ? "Аварийный выход: завершите Focus Lock через Диспетчер задач. Это добровольное ограничение."
          : "Emergency exit: end Focus Lock in Task Manager. This is a voluntary restriction."}
      </p>
    </div>
  );
}
