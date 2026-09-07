import { cn } from "@lib/utils";

type LogoSize = "sm" | "md";

interface LogoProps {
  className?: string;
  size?: LogoSize;
}

const sizeClass: Record<LogoSize, string> = {
  sm: "size-9",
  md: "size-12",
};

export function Logo({ className, size = "sm" }: LogoProps) {
  return (
    <div
      className={cn(
        "relative flex shrink-0 items-center justify-center",
        sizeClass[size],
        className,
      )}
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        fill="currentColor"
        viewBox="0 0 1024 1024"
        className="size-full!"
      >
        <circle cx="512" cy="242.04" r="110.05" />
        <path d="M598.56 892H425.45c-13.738 0-25.337-10.21-27.071-23.839l-41.43-325.36c-11.244-88.31 42.869-169.89 122.44-169.89h65.239c79.568 0 133.68 81.579 122.44 169.89l-41.43 325.36C623.904 881.79 612.304 892 598.567 892" />
      </svg>
    </div>
  );
}
