import { invalidate, invalidateKey, invalidateTag } from "./navigation.ts";

export interface ActionValidationError {
  fieldErrors?: Record<string, string>;
  formError?: string;
}

export interface ActionRedirect {
  to: string;
  status?: number;
}

export interface ActionEnhanceSuccess<T = unknown> {
  data?: T;
  code?: string;
  redirect?: ActionRedirect;
  validation?: ActionValidationError;
  revalidate?: string[];
  revalidateTags?: string[];
  revalidateKeys?: string[];
}

export interface FormEnhanceOptions<T = unknown> {
  action?: string;
  optimistic?: (form: HTMLFormElement, formData: FormData) => void;
  onPending?: (form: HTMLFormElement) => void;
  onSuccess?: (
    result: ActionEnhanceSuccess<T>,
    form: HTMLFormElement,
    response: Response,
  ) => void;
  onError?: (error: string, form: HTMLFormElement, response?: Response) => void;
  onValidation?: (
    validation: ActionValidationError,
    form: HTMLFormElement,
    response: Response,
  ) => void;
  onRedirect?: (
    redirect: ActionRedirect,
    form: HTMLFormElement,
    response: Response,
  ) => void;
}

async function applyRevalidationHints(payload: ActionEnhanceSuccess): Promise<void> {
  if (payload.revalidate) {
    for (const path of payload.revalidate) {
      await invalidate(path);
    }
  }
  if (payload.revalidateTags) {
    for (const tag of payload.revalidateTags) {
      await invalidateTag(tag);
    }
  }
  if (payload.revalidateKeys) {
    for (const key of payload.revalidateKeys) {
      await invalidateKey(key);
    }
  }
}

function clearFieldErrors(form: HTMLFormElement): void {
  form
    .querySelectorAll("[data-gospa-error], [aria-invalid='true']")
    .forEach((el) => {
      el.removeAttribute("data-gospa-error");
      el.removeAttribute("aria-invalid");
    });
}

function applyFieldErrors(
  form: HTMLFormElement,
  validation: ActionValidationError,
): void {
  clearFieldErrors(form);
  for (const [field, message] of Object.entries(validation.fieldErrors ?? {})) {
    const target = form.querySelector(
      `[name="${CSS.escape(field)}"]`,
    ) as HTMLElement | null;
    if (!target) continue;
    target.setAttribute("aria-invalid", "true");
    target.setAttribute("data-gospa-error", message);
  }
}

export function enhanceForm<T = unknown>(
  form: HTMLFormElement,
  options: FormEnhanceOptions<T> = {},
): () => void {
  const onSubmit = async (event: Event): Promise<void> => {
    event.preventDefault();
    const submitter = (event as SubmitEvent).submitter as
      | HTMLButtonElement
      | HTMLInputElement
      | null;
    const formData = new FormData(form);

    if (submitter && submitter.name) {
      formData.set(submitter.name, submitter.value ?? "");
    }

    const actionOverride = submitter?.getAttribute("formaction") ?? undefined;
    const target = actionOverride || options.action || form.action || window.location.pathname;
    const url = new URL(target, window.location.origin);
    const actionName =
      submitter?.getAttribute("data-gospa-action") ||
      submitter?.value ||
      form.dataset.gospaAction ||
      "default";
    url.searchParams.set("_action", actionName);

    options.onPending?.(form);
    options.optimistic?.(form, formData);

    let response: Response | undefined;
    try {
      response = await fetch(url.toString(), {
        method: (form.method || "POST").toUpperCase(),
        body: formData,
        credentials: "same-origin",
        headers: {
          "X-Gospa-Enhance": "1",
          Accept: "application/json",
        },
      });
    } catch (error) {
      const msg = error instanceof Error ? error.message : "Network error";
      options.onError?.(msg, form);
      return;
    }

    let payload: ActionEnhanceSuccess<T> | undefined;
    try {
      payload = (await response.json()) as ActionEnhanceSuccess<T>;
    } catch {
      payload = undefined;
    }

    if (!response.ok) {
      const errMsg =
        payload && "error" in payload && typeof (payload as any).error === "string"
          ? (payload as any).error
          : `Action failed with HTTP ${response.status}`;
      options.onError?.(errMsg, form, response);
      return;
    }

    const result = payload ?? {};
    await applyRevalidationHints(result);

    if (result.validation) {
      applyFieldErrors(form, result.validation);
      options.onValidation?.(result.validation, form, response);
      return;
    }

    clearFieldErrors(form);

    if (result.redirect?.to) {
      options.onRedirect?.(result.redirect, form, response);
      if (!options.onRedirect) {
        window.location.assign(result.redirect.to);
      }
      return;
    }

    options.onSuccess?.(result, form, response);
  };

  form.addEventListener("submit", onSubmit);
  return () => form.removeEventListener("submit", onSubmit);
}

export function enhanceForms<T = unknown>(
  selector = "form[data-gospa-enhance]",
  options: FormEnhanceOptions<T> = {},
): () => void {
  const forms = Array.from(document.querySelectorAll(selector)).filter(
    (el): el is HTMLFormElement => el instanceof HTMLFormElement,
  );
  const cleanups = forms.map((form) => enhanceForm(form, options));
  return () => {
    for (const cleanup of cleanups) cleanup();
  };
}

