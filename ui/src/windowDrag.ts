import type { PointerEvent } from "react";
import { call } from "./api";

// Coalesce moves and apply the final position even when a fast drag finishes
// before the initial native position response arrives.
export function beginWindowDrag(
  event: PointerEvent,
  onError: (error: string) => void,
) {
  if (!window.go || event.button !== 0 || !event.isPrimary) return;
  event.preventDefault();
  event.stopPropagation();
  const target = event.currentTarget;
  const pointer = event.pointerId;
  target.setPointerCapture(pointer);
  const start = { x: event.screenX, y: event.screenY };
  const scale = window.devicePixelRatio || 1;
  const origin = call<{ x: number; y: number }>("windowPosition");
  let latest = { x: 0, y: 0 },
    applied = latest,
    sending = false;
  const flush = async () => {
    if (sending) return;
    sending = true;
    try {
      const base = await origin;
      while (applied !== latest) {
        applied = latest;
        await call("moveWindow", {
          x: base.x + Math.round(applied.x * scale),
          y: base.y + Math.round(applied.y * scale),
        });
      }
    } catch (error) {
      onError(String(error));
    } finally {
      sending = false;
    }
  };
  const move = (e: globalThis.PointerEvent) => {
    if (e.pointerId !== pointer) return;
    latest = { x: e.screenX - start.x, y: e.screenY - start.y };
    void flush();
  };
  const end = (e: globalThis.PointerEvent) => {
    if (e.pointerId !== pointer) return;
    if (e.type === "pointerup") move(e);
    target.removeEventListener("pointermove", move as EventListener);
    target.removeEventListener("pointerup", end as EventListener);
    target.removeEventListener("pointercancel", end as EventListener);
    target.removeEventListener("lostpointercapture", end as EventListener);
    if (target.hasPointerCapture(pointer))
      target.releasePointerCapture(pointer);
  };
  target.addEventListener("pointermove", move as EventListener);
  target.addEventListener("pointerup", end as EventListener);
  target.addEventListener("pointercancel", end as EventListener);
  target.addEventListener("lostpointercapture", end as EventListener);
  void flush();
}
