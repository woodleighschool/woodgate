import { z } from "zod";

export function requiredString(label: string) {
  return z.string().trim().min(1, `${label} is required.`);
}

export function emailAddress(message = "Enter a valid email.") {
  return z.string().trim().pipe(z.email(message));
}

export function firstErrorMessage(errors: readonly unknown[]) {
  for (const error of errors) {
    if (typeof error === "string") return error;
    if (error && typeof error === "object" && "message" in error) {
      const message = (error as { message?: unknown }).message;
      if (typeof message === "string") return message;
    }
  }
  return undefined;
}
