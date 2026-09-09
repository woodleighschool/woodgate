import { nonEmpty } from "@lib/utils";

export function checkinPersonLabel(person: { name: string; email: string }): string {
  return nonEmpty(person.name) ?? person.email;
}
