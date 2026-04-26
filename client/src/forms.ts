import { invalidate, invalidateKey, invalidateTag } from "./navigation.ts";
import { emitRuntimeSignal } from "./runtime-signals.ts";

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

const activeControllers = new WeakMap<HTMLFormElement, AbortController>();
const submitEpoch = new WeakMap<HTMLFormElement, number>();

async function applyRevalidationHints(
  payload: ActionEnhanceSuccess,
): Promise<void> {
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
    const safeField =
      typeof CSS !== "undefined" && typeof CSS.escape === "function"
        ? CSS.escape(field)
        : field.replace(/["\\]/g, "\\$&");
    const target = form.querySelector(
      `[name="${safeField}"]`,
    ) as HTMLElement | null;
    if (!target) continue;
    target.setAttribute("aria-invalid", "true");
    target.setAttribute("data-gospa-error", message);
  }
}

function validationFromPayload(
  payload: unknown,
): ActionValidationError | undefined {
  if (!payload || typeof payload !== "object") {
    return undefined;
  }

  const typed = payload as ActionEnhanceSuccess;
  if (typed.validation) {
    return typed.validation;
  }

  const data = typed.data;
  if (!data || typeof data !== "object") {
    return undefined;
  }

  const source = data as {
    fieldErrors?: Record<string, unknown>;
    formError?: unknown;
  };
  const fieldErrors: Record<string, string> = {};
  if (source.fieldErrors && typeof source.fieldErrors === "object") {
    for (const [key, value] of Object.entries(source.fieldErrors)) {
      if (typeof value === "string" && value) {
        fieldErrors[key] = value;
      }
    }
  }
  const formError =
    typeof source.formError === "string" ? source.formError : undefined;
  if (!formError && Object.keys(fieldErrors).length === 0) {
    return undefined;
  }
  return {
    fieldErrors: Object.keys(fieldErrors).length > 0 ? fieldErrors : undefined,
    formError,
  };
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
    const target =
      actionOverride ||
      options.action ||
      form.action ||
      window.location.pathname;
    const url = new URL(target, window.location.origin);
    const actionName =
      submitter?.getAttribute("data-gospa-action") ||
      submitter?.value ||
      form.dataset.gospaAction ||
      "default";
    url.searchParams.set("_action", actionName);

    const priorController = activeControllers.get(form);
    if (priorController) {
      priorController.abort();
    }
    const controller = new AbortController();
    activeControllers.set(form, controller);
    const nextEpoch = (submitEpoch.get(form) ?? 0) + 1;
    submitEpoch.set(form, nextEpoch);
    const epoch = nextEpoch;

    options.onPending?.(form);
    options.optimistic?.(form, formData);
    emitRuntimeSignal("gospa:action-pending", {
      action: actionName,
      path: url.pathname,
      method: (form.method || "POST").toUpperCase(),
    });

    let response: Response | undefined;
    try {
      const method = (form.method || "POST").toUpperCase();
      const requestInit: RequestInit = {
        method,
        credentials: "same-origin",
        signal: controller.signal,
        headers: {
          "X-Gospa-Enhance": "1",
          Accept: "application/json",
        },
      };
      if (method === "GET" || method === "HEAD") {
        for (const [key, value] of formData.entries()) {
          if (typeof value === "string") {
            url.searchParams.append(key, value);
          }
        }
      } else {
        requestInit.body = formData;
      }

      response = await fetch(url.toString(), requestInit);
    } catch (error) {
      if ((error as Error)?.name === "AbortError") {
        emitRuntimeSignal("gospa:action-aborted", { action: actionName });
        return;
      }
      const msg = error instanceof Error ? error.message : "Network error";
      options.onError?.(msg, form);
      emitRuntimeSignal("gospa:action-error", {
        action: actionName,
        path: url.pathname,
        error: msg,
      });
      return;
    }
    if (epoch !== submitEpoch.get(form)) {
      return;
    }

    let payload: ActionEnhanceSuccess<T> | undefined;
    try {
      payload = (await response.json()) as ActionEnhanceSuccess<T>;
    } catch {
      payload = undefined;
    }

    if (!response.ok) {
      const validation = validationFromPayload(payload);
      if (validation) {
        applyFieldErrors(form, validation);
        options.onValidation?.(validation, form, response);
        emitRuntimeSignal("gospa:action-validation", {
          action: actionName,
          path: url.pathname,
          status: response.status,
          validation,
        });
        return;
      }
      const errMsg =
        payload &&
        "error" in payload &&
        typeof (payload as any).error === "string"
          ? (payload as any).error
          : `Action failed with HTTP ${response.status}`;
      options.onError?.(errMsg, form, response);
      emitRuntimeSignal("gospa:action-error", {
        action: actionName,
        path: url.pathname,
        status: response.status,
        error: errMsg,
      });
      return;
    }

    const result = payload ?? {};
    await applyRevalidationHints(result);

    if (result.validation) {
      applyFieldErrors(form, result.validation);
      options.onValidation?.(result.validation, form, response);
      emitRuntimeSignal("gospa:action-validation", {
        action: actionName,
        path: url.pathname,
        status: response.status,
        validation: result.validation,
      });
      return;
    }

    clearFieldErrors(form);

    if (result.redirect?.to) {
      options.onRedirect?.(result.redirect, form, response);
      emitRuntimeSignal("gospa:action-redirect", {
        action: actionName,
        from: url.pathname,
        to: result.redirect.to,
        status: result.redirect.status ?? response.status,
      });
      if (!options.onRedirect) {
        window.location.assign(result.redirect.to);
      }
      return;
    }

    options.onSuccess?.(result, form, response);
    emitRuntimeSignal("gospa:action-success", {
      action: actionName,
      path: url.pathname,
      status: response.status,
    });
  };

  form.addEventListener("submit", onSubmit);
  return () => {
    form.removeEventListener("submit", onSubmit);
    const active = activeControllers.get(form);
    if (active) {
      active.abort();
      activeControllers.delete(form);
    }
    submitEpoch.delete(form);
  };
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
