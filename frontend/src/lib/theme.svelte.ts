type Theme = "light" | "dark" | "system";

let theme = $state<Theme>("system");

export function setTheme(t: Theme) {
  theme = t;
  applyTheme();
}

export function getTheme(): Theme {
  return theme;
}

export function getResolved(): "light" | "dark" {
  if (theme === "system") {
    return typeof window !== "undefined" && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }
  return theme;
}

function applyTheme() {
  if (typeof document === "undefined") return;

  const html = document.documentElement;
  html.classList.remove("light", "dark");

  if (theme !== "system") {
    html.classList.add(theme);
  }
}

if (typeof window !== "undefined") {
  window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", () => {
    if (theme === "system") {
      applyTheme();
    }
  });
}
