import type { DeploymentStatus } from "../../types";

const STATUS_CONFIG: Record<
  DeploymentStatus,
  { label: string; dot: string; text: string }
> = {
  pending: {
    label: "Pending",
    dot: "bg-gray-500",
    text: "text-gray-400",
  },
  building: {
    label: "Building",
    dot: "bg-yellow-400 animate-pulse",
    text: "text-yellow-400",
  },
  deploying: {
    label: "Deploying",
    dot: "bg-blue-400 animate-pulse",
    text: "text-blue-400",
  },
  running: {
    label: "Running",
    dot: "bg-green-400",
    text: "text-green-400",
  },
  failed: {
    label: "Failed",
    dot: "bg-red-500",
    text: "text-red-400",
  },
};

type Props = {
  status: DeploymentStatus;
};

export function StatusBadge({ status }: Props) {
  const { label, dot, text } = STATUS_CONFIG[status];
  return (
    <span
      className={`inline-flex items-center gap-1.5 text-xs font-medium ${text}`}
    >
      <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${dot}`} />
      {label}
    </span>
  );
}
