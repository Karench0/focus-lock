import { useId, useState } from "react";
import { BookOpen, Check } from "lucide-react";
import "./storozhenko.css";
import { KotlinText } from "./KotlinText";

type Question = {
  token: string;
  prompt: string;
  options: string[];
  sourceUrl: string;
};

type Props = {
  language: "ru" | "en";
  question?: Question;
  busy: boolean;
  feedback: string;
  onAnswer: (option: number) => void;
  onCancel: () => void;
};

export function KotlinChallenge(props: Props) {
  const ru = props.language === "ru";
  return (
    <div className="kotlin-challenge">
      <div className="modal-symbol">
        <BookOpen size={26} />
      </div>
      <div className="eyebrow">KOTLIN · MIDDLE</div>
      <h2>
        {ru ? "Один вопрос перед выходом" : "One question before you finish"}
      </h2>
      <p>
        {ru
          ? "Выберите один правильный ответ. После ошибки занятие продолжится — можно попробовать снова."
          : "Choose one correct answer. An incorrect answer keeps the exercise running; you can try again. Questions are in Russian."}
      </p>
      {props.question ? (
        <QuestionForm
          key={props.question.token}
          {...props}
          question={props.question}
        />
      ) : (
        <>
          <p className="kotlin-loading" role="status">
            {ru ? "Готовим вопрос…" : "Preparing your question…"}
          </p>
          {props.feedback && (
            <p className="kotlin-feedback" role="status">
              {props.feedback}
            </p>
          )}
          <button
            type="button"
            className="secondary"
            disabled={props.busy}
            onClick={props.onCancel}
          >
            {ru ? "Продолжить занятие" : "Keep studying"}
          </button>
        </>
      )}
    </div>
  );
}

function QuestionForm({
  language,
  question,
  busy,
  feedback,
  onAnswer,
  onCancel,
}: Omit<Props, "question"> & { question: Question }) {
  const ru = language === "ru";
  const [chosen, setChosen] = useState<number | null>(null);
  const radioName = useId();
  return (
    <form
      className="kotlin-question-form"
      onSubmit={(event) => {
        event.preventDefault();
        if (chosen !== null && !busy) onAnswer(chosen);
      }}
    >
      <div
        className="kotlin-question-prompt"
        role="region"
        aria-label={ru ? "Вопрос по Kotlin" : "Kotlin question"}
        tabIndex={0}
      >
        <KotlinText text={question.prompt} />
      </div>
      <div className="kotlin-answer-area">
        <span className="kotlin-options-title" aria-hidden="true">
          {ru ? "Варианты ответа" : "Answer options"}
        </span>
        <div
          className="kotlin-options-scroll"
          role="region"
          aria-label={ru ? "Варианты ответа" : "Answer options"}
          tabIndex={0}
        >
          <fieldset className="kotlin-options" disabled={busy}>
            <legend>{ru ? "Варианты ответа" : "Answer options"}</legend>
            {question.options.map((option, index) => (
              <label
                className={`kotlin-option${chosen === index ? " selected" : ""}`}
                key={index}
              >
                <input
                  type="radio"
                  name={radioName}
                  value={index}
                  checked={chosen === index}
                  onChange={() => setChosen(index)}
                />
                <span className="kotlin-option-letter" aria-hidden="true">
                  {String.fromCharCode(65 + index)}
                </span>
                <span className="kotlin-option-text"><KotlinText text={option} /></span>
                {chosen === index && (
                  <Check
                    size={15}
                    className="kotlin-option-check"
                    aria-hidden="true"
                  />
                )}
              </label>
            ))}
          </fieldset>
        </div>
      </div>
      <div
        className="kotlin-feedback-slot"
        aria-live="polite"
        aria-atomic="true"
      >
        {feedback && <p className="kotlin-feedback">{feedback}</p>}
      </div>
      <div className="button-row storozhenko-actions">
        <button
          type="button"
          className="secondary"
          disabled={busy}
          onClick={onCancel}
        >
          {ru ? "Продолжить занятие" : "Keep studying"}
        </button>
        <button
          type="submit"
          className="primary"
          disabled={chosen === null || busy}
        >
          {busy
            ? ru
              ? "Проверяем…"
              : "Checking…"
            : ru
              ? "Ответить"
              : "Submit answer"}
        </button>
      </div>
    </form>
  );
}
