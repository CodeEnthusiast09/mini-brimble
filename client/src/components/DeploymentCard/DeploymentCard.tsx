import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deleteDeployment } from "../../api/deployments";
import { StatusBadge } from "../StatusBadge";
import type { Deployment } from "../../types";
import { parseGitHubRepo, formatRelativeTime } from "../../lib/utils";

type Props = {
  deployment: Deployment;
  isSelected: boolean;
  onSelect: (id: string) => void;
};

type IconButtonProps = {
  onClick: (e: React.MouseEvent) => void;
  title: string;
  children: React.ReactNode;
};

function IconButton({ onClick, title, children }: IconButtonProps) {
  return (
    <button
      onClick={onClick}
      title={title}
      className="p-1.5 rounded-md text-gray-600 hover:text-gray-300 hover:bg-white/6 transition-all"
    >
      {children}
    </button>
  );
}

export function DeploymentCard({ deployment, isSelected, onSelect }: Props) {
  const queryClient = useQueryClient();
  const parsed = parseGitHubRepo(deployment.github_url);
  const [copied, setCopied] = useState(false);

  const stopMutation = useMutation({
    mutationFn: () => deleteDeployment(deployment.id),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["deployments"] }),
  });

  const handleCopy = (e: React.MouseEvent) => {
    e.stopPropagation();
    navigator.clipboard.writeText(deployment.id);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  const handleOpen = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (deployment.live_url)
      window.open(deployment.live_url, "_blank", "noopener,noreferrer");
  };

  return (
    <div
      onClick={() => onSelect(deployment.id)}
      className={`cursor-pointer rounded-xl border p-4 transition-all ${isSelected
          ? "border-white/20 bg-white/5"
          : "border-white/7 bg-[#111111] hover:border-white/12 hover:bg-white/3"
        }`}
    >
      <div className="flex items-start justify-between gap-3">
        {/* Info */}
        <div className="min-w-0 flex-1 space-y-2.5">
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0">
              {parsed ? (
                <>
                  <p className="text-sm font-medium text-white truncate">
                    {parsed.repo}
                  </p>
                  <p className="text-xs text-gray-500 truncate">
                    {parsed.owner}
                  </p>
                </>
              ) : (
                <p className="text-xs text-gray-400 font-mono truncate">
                  {deployment.github_url}
                </p>
              )}
            </div>
            <StatusBadge status={deployment.status} />
          </div>

          <div className="flex items-center gap-2 text-xs text-gray-600 font-mono">
            <span>{deployment.id.slice(0, 8)}</span>
            {deployment.image_tag && (
              <>
                <span>·</span>
                <span className="truncate max-w-24">
                  {deployment.image_tag}
                </span>
              </>
            )}
            <span>·</span>
            <span>{formatRelativeTime(deployment.created_at)}</span>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-0.5 shrink-0 -mr-1">
          <IconButton
            onClick={handleCopy}
            title={copied ? "Copied!" : "Copy deployment ID"}
          >
            {copied ? (
              <svg
                width="13"
                height="13"
                viewBox="0 0 13 13"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="text-green-400"
              >
                <path d="M2 7l3.5 3.5L11 3" />
              </svg>
            ) : (
              <svg
                width="13"
                height="13"
                viewBox="0 0 13 13"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <rect x="4.5" y="4.5" width="7" height="7" rx="1.5" />
                <path d="M8.5 4.5V3A1.5 1.5 0 007 1.5h-4A1.5 1.5 0 001.5 3v4A1.5 1.5 0 003 8.5h1.5" />
              </svg>
            )}
          </IconButton>

          {deployment.live_url && deployment.status === "running" && (
            <IconButton onClick={handleOpen} title="Open live URL">
              <svg
                width="13"
                height="13"
                viewBox="0 0 13 13"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M2 11L11 2M11 2H5.5M11 2V7.5" />
              </svg>
            </IconButton>
          )}

          <button
            onClick={(e) => {
              e.stopPropagation();
              stopMutation.mutate();
            }}
            disabled={stopMutation.isPending}
            className="ml-1.5 rounded-md px-2.5 py-1 text-xs font-medium text-gray-500 border border-white/[0.07] hover:text-red-400 hover:border-red-500/30 hover:bg-red-500/5 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
          >
            {stopMutation.isPending ? "…" : "Stop"}
          </button>
        </div>
      </div>
    </div>
  );
}
