// Tiny class-name combiner (clsx-shaped API) without external dependencies.
// Accepts strings, falsy values, and records of boolean flags.

type ClassValue =
  | string
  | number
  | null
  | undefined
  | false
  | Record<string, boolean | null | undefined>
  | ClassValue[];

export function cn(...inputs: ClassValue[]): string {
  const out: string[] = [];

  const walk = (value: ClassValue): void => {
    if (!value) return;
    if (typeof value === "string" || typeof value === "number") {
      out.push(String(value));
      return;
    }
    if (Array.isArray(value)) {
      for (const item of value) {
        walk(item);
      }
      return;
    }
    if (typeof value === "object") {
      for (const key of Object.keys(value)) {
        if (value[key]) {
          out.push(key);
        }
      }
    }
  };

  for (const input of inputs) {
    walk(input);
  }

  return out.join(" ");
}
