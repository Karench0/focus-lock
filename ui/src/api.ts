declare global {
  interface Window {
    go?: {
      main: {
        App: { Call: (command: string, payload: string) => Promise<string> };
      };
    };
    runtime?: {
      EventsOn: (event: string, cb: (...args: unknown[]) => void) => () => void;
    };
  }
}
const token = new URLSearchParams(location.hash.slice(1)).get("token");
export async function call<T>(
  command: string,
  payload: unknown = {},
): Promise<T> {
  if (window.go)
    return JSON.parse(
      await window.go.main.App.Call(command, JSON.stringify(payload)),
    ) as T;
  if (!token)
    throw new Error(
      "Open Focus Lock.exe, or use the authenticated development preview URL.",
    );
  const res = await fetch("/api", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ command, payload }),
  });
  if (!res.ok) throw new Error(await res.text());
  return (await res.json()) as T;
}
