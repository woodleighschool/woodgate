import type { Validator } from "react-admin";
import { z } from "zod";

const nonEmptyTrimmedString = z.string().trim().min(1);

export const trimmedRequired =
  (label: string): Validator =>
  (value: unknown): string | undefined => {
    const parsedValue = typeof value === "string" ? value : "";
    return nonEmptyTrimmedString.safeParse(parsedValue).success ? undefined : `${label} is required`;
  };
