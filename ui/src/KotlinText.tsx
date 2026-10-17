const tokens = /(\/\/[^\n]*|"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|@[A-Za-z]\w*|\b(?:fun|val|var|class|interface|object|data|sealed|inline|reified|crossinline|noinline|open|override|suspend|return|if|else|when|is|in|out|by|throw|true|false|null|try|catch|while)\b|\b\d+(?:\.\d+)?\b|\b[A-Z]\w*\b|\b[A-Za-z_]\w*(?=\s*\())/g;
const keywords = /^(fun|val|var|class|interface|object|data|sealed|inline|reified|crossinline|noinline|open|override|suspend|return|if|else|when|is|in|out|by|throw|true|false|null|try|catch|while)$/;

function highlight(source: string) {
  return source.split(tokens).map((token, i) => {
    let kind = "";
    if (token.startsWith("//")) kind = "comment";
    else if (/^["']/.test(token)) kind = "string";
    else if (token.startsWith("@")) kind = "annotation";
    else if (keywords.test(token)) kind = "keyword";
    else if (/^\d/.test(token)) kind = "number";
    else if (/^[A-Z]\w*$/.test(token)) kind = "type";
    else if (/^[A-Za-z_]\w*$/.test(token)) kind = "function";
    return kind ? <span className={`kt-${kind}`} key={i}>{token}</span> : token;
  });
}

function inline(source: string) {
  return source.split(/`([^`\n]+)`/g).map((part, i) =>
    i % 2 ? <code className="kotlin-inline" key={i}>{part}</code> : part,
  );
}

// Only two plain-text formats are supported. React escapes all text; no HTML
// or external Markdown renderer is used for questions from the backend.
export function KotlinText({ text }: { text: string }) {
  return text.split(/```(?:kotlin)?\r?\n([\s\S]*?)```/g).map((part, i) =>
    i % 2 ? (
      <div className="kotlin-code" key={i}>
        <div className="kotlin-code-heading"><span className="kotlin-code-dot" />KOTLIN</div>
        <pre tabIndex={0} aria-label="Kotlin"><code>{highlight(part.replace(/\n$/, ""))}</code></pre>
      </div>
    ) : part.trim() ? <span className="kotlin-prose" key={i}>{inline(part.trim())}</span> : null,
  );
}
