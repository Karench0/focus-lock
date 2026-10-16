void chrome.runtime
  .sendMessage({ kind: "status" })
  .then(async (state) => {
    const { language } = await chrome.storage.session.get("language");
    const en = language === "en";
    document.getElementById("status").textContent = state.connected
      ? state.active
        ? en
          ? "Focus session active"
          : "Фокус-сессия активна"
        : en
          ? "Connected, ready"
          : "Подключено, готово"
      : en
        ? "App disconnected"
        : "Приложение не подключено";
    document.getElementById("detail").textContent = state.connected
      ? en
        ? "Rules are controlled by the desktop app. Access restores automatically if the app stops."
        : "Правила задаёт настольное приложение. При его завершении доступ восстанавливается автоматически."
      : en
        ? "Open Focus Lock and connect the local bridge in Settings."
        : "Откройте Focus Lock и подключите локальный мост в настройках приложения.";
  })
  .catch(() => {
    document.getElementById("status").textContent =
      "Disconnected / Не подключено";
  });
