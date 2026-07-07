const LIVE_REGION_ID = "astra-announcer";

export function announce(
  message: string,
  priority: "polite" | "assertive" = "polite",
): void {
  if (typeof document === "undefined") {
    return;
  }

  let region = document.getElementById(LIVE_REGION_ID);
  if (!region) {
    region = document.createElement("div");
    region.id = LIVE_REGION_ID;
    region.setAttribute("aria-live", priority);
    region.setAttribute("aria-atomic", "true");
    region.className = "sr-only";
    document.body.appendChild(region);
  }

  region.textContent = "";
  requestAnimationFrame(() => {
    region.textContent = message;
  });
}

const FOCUSABLE_SELECTOR =
  'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])';

export interface FocusTrapOptions {
  onEscape?: () => void;
  initialFocus?: HTMLElement | null;
}

export interface FocusTrapInstance {
  activate: () => void;
  deactivate: () => void;
  active: () => boolean;
}

export function createFocusTrap(
  element: HTMLElement,
  options?: FocusTrapOptions,
): FocusTrapInstance {
  let active = false;

  function getFocusable(): HTMLElement[] {
    return Array.from(
      element.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR),
    ).filter(
      (el) =>
        el.tabIndex >= 0 &&
        !el.hasAttribute("disabled") &&
        el.getAttribute("aria-hidden") !== "true",
    );
  }

  function onKeyDown(event: KeyboardEvent): void {
    if (event.key === "Escape" && options?.onEscape) {
      event.preventDefault();
      options.onEscape();
      return;
    }

    if (event.key !== "Tab") {
      return;
    }

    const focusable = getFocusable();
    if (focusable.length === 0) {
      event.preventDefault();
      return;
    }

    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    const activeElement = document.activeElement as HTMLElement | null;

    if (event.shiftKey && activeElement === first) {
      event.preventDefault();
      last?.focus();
    } else if (!event.shiftKey && activeElement === last) {
      event.preventDefault();
      first?.focus();
    }
  }

  function activate(): void {
    if (active) {
      return;
    }
    active = true;
    element.setAttribute("role", "dialog");
    element.setAttribute("aria-modal", "true");
    element.addEventListener("keydown", onKeyDown);
    const focusable = getFocusable();
    const target = options?.initialFocus ?? focusable[0];
    target?.focus();
  }

  function deactivate(): void {
    if (!active) {
      return;
    }
    active = false;
    element.removeEventListener("keydown", onKeyDown);
    element.removeAttribute("role");
    element.removeAttribute("aria-modal");
  }

  return { activate, deactivate, active: () => active };
}
