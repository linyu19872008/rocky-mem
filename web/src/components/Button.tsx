import type { ButtonHTMLAttributes, ReactNode } from "react";

type Props = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "default" | "primary" | "ok" | "bad" | "warn" | "ghost";
  size?: "sm" | "md";
  children: ReactNode;
};

export function Button({ variant = "default", size = "md", className = "", children, ...rest }: Props) {
  const cls = ["btn", variant !== "default" ? variant : "", size === "sm" ? "sm" : "", className]
    .filter(Boolean)
    .join(" ");
  return (
    <button type={rest.type ?? "button"} className={cls} {...rest}>
      {children}
    </button>
  );
}
