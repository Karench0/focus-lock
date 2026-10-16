void chrome.storage.session.get("language").then(({ language }) => {
  if (language === "en") {
    document.documentElement.lang = "en";
    document.getElementById("title").textContent =
      "This website is unavailable during your focus session";
    document.getElementById("description").textContent =
      "Your attention belongs to another task right now. Wait until your session ends or return to an allowed website.";
    document.getElementById("back").textContent = "Go back";
    document.getElementById("note").textContent =
      "If Focus Lock has stopped, access will restore automatically. Navigate again after restrictions are lifted.";
  }
});
void chrome.runtime.sendMessage({ kind: "blocked" }).catch(() => {});
document.getElementById("back").addEventListener("click", () => history.back());
