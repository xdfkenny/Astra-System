import { Button } from "@astra/design-system";
import { useTheme } from "../hooks/useTheme";

export function ThemeToggle(): React.JSX.Element {
  const { theme, setTheme } = useTheme();

  return (
    <div className="flex items-center gap-2 rounded-lg border border-border p-1">
      {(["light", "dark", "system"] as const).map((t) => (
        <Button
          key={t}
          variant={theme === t ? "primary" : "ghost"}
          onClick={() => { setTheme(t); }}
          className="min-h-8 px-2 py-1 text-xs capitalize"
        >
          {t}
        </Button>
      ))}
    </div>
  );
}
