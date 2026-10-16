export class Lease {
  constructor(
    expire,
    clock = Date.now,
    schedule = (fn, ms) => setTimeout(fn, ms),
    cancel = (id) => clearTimeout(id),
  ) {
    this.expire = expire;
    this.clock = clock;
    this.schedule = schedule;
    this.cancel = cancel;
    this.timer = null;
    this.deadline = 0;
  }
  renew(ms) {
    if (!Number.isFinite(ms) || ms < 1000 || ms > 6000)
      throw new Error("Invalid lease");
    this.clear();
    this.deadline = this.clock() + ms;
    this.timer = this.schedule(() => {
      this.deadline = 0;
      this.expire();
    }, ms);
  }
  clear() {
    if (this.timer !== null) this.cancel(this.timer);
    this.timer = null;
    this.deadline = 0;
  }
  check() {
    if (this.deadline && this.clock() >= this.deadline) {
      this.clear();
      this.expire();
    }
  }
}
