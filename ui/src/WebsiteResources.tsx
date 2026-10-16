import { useEffect, useState } from "react";
import { Code2, Plus } from "lucide-react";
import { copy } from "./i18n";
import type { Website } from "./types";

// Resource permissions are explicit, persisted with the profile, and do not
// grant permission to navigate to the CDN itself.
export function WebsiteResources({
  site,
  language,
  disabled,
  onSave,
}: {
  site: Website;
  language: "ru" | "en";
  disabled: boolean;
  onSave: (resources: string[]) => void;
}) {
  const t = copy[language];
  const saved = (site.resources || []).join(", ");
  const [draft, setDraft] = useState(saved);
  useEffect(() => setDraft(saved), [saved]);
  const suggested =
    site.domain === "github.com"
      ? [
          "github.githubassets.com",
          "avatars.githubusercontent.com",
          "user-images.githubusercontent.com",
          "private-user-images.githubusercontent.com",
        ]
      : [];
  const missing = suggested.filter(
    (domain) => !site.resources?.includes(domain),
  );
  return (
    <details
      className="resource-details"
      open={suggested.length > 0 && missing.length > 0 ? true : undefined}
    >
      <summary>
        <Code2 size={14} />
        {t.resourceDomains} <span>({site.resources?.length || 0})</span>
      </summary>
      <p>{t.resourceHint}</p>
      {missing.length > 0 && (
        <div className="resource-suggestion">
          <p>{t.githubResourcesHint}</p>
          <code>{missing.join(", ")}</code>
          <button
            type="button"
            className="secondary"
            disabled={disabled}
            onClick={() =>
              onSave([...new Set([...(site.resources || []), ...suggested])])
            }
          >
            <Plus size={14} />
            {t.addGithubResources}
          </button>
        </div>
      )}
      <form
        onSubmit={(e) => {
          e.preventDefault();
          onSave([...new Set(draft.split(/[\s,;]+/).filter(Boolean))]);
        }}
      >
        <label>
          {t.resourceDomains}
          <textarea
            aria-label={`${t.resourceDomains}: ${site.domain}`}
            rows={2}
            placeholder="cdn.example.com, static.example.com"
            value={draft}
            disabled={disabled}
            onChange={(e) => setDraft(e.target.value)}
          />
        </label>
        <button className="secondary" disabled={disabled || draft === saved}>
          {t.save}
        </button>
      </form>
    </details>
  );
}
