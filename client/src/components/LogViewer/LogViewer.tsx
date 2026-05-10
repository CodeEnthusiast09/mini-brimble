import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { getLogs, getLogStreamUrl } from "../../api/logs";
import type { Deployment } from "../../types";
import { parseGitHubRepo } from "../../lib/utils";

const TERMINAL_STATUSES = new Set<string>(["running", "failed"]);

type Tab = "build" | "runtime";

type Props = {
  deployment: Deployment;
};

function getLineClass(line: string): string {
  const lower = line.toLowerCase();
  if (/\b(error|failed|fatal|exception)\b/.test(lower)) return "text-red-400";
  if (/\b(warn|warning)\b/.test(lower)) return "text-yellow-400";
  if (
    /\b(success|successfully|done|complete|built|pushed|pulled)\b/.test(lower)
  )
    return "text-green-400";
  if (/^(step|>>>|==>|---|\[)/.test(line.trimStart())) return "text-blue-300";
  return "text-gray-300";
}

export function LogViewer({ deployment }: Props) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [streamedLogs, setStreamedLogs] = useState<string[]>([]);
  const [activeTab, setActiveTab] = useState<Tab>("build");
  const parsed = parseGitHubRepo(deployment.github_url);

  const { data: existingLogs, isLoading } = useQuery({
    queryKey: ["logs", deployment.id],
    queryFn: () => getLogs(deployment.id),
  });

  // Reset state when switching deployments
  useEffect(() => {
    setStreamedLogs([]);
    setActiveTab("build");
  }, [deployment.id]);

  // Auto-switch to Runtime tab when the app starts running
  useEffect(() => {
    if (deployment.status === "running") {
      setActiveTab("runtime");
    }
  }, [deployment.status]);

  // SSE stream while non-terminal
  useEffect(() => {
    if (TERMINAL_STATUSES.has(deployment.status)) return;

    const es = new EventSource(getLogStreamUrl(deployment.id));

    es.addEventListener("log", (e: MessageEvent) => {
      setStreamedLogs((prev) => [...prev, String(e.data)]);
    });

    es.onerror = () => es.close();

    return () => es.close();
  }, [deployment.id, deployment.status]);

  // Auto-scroll to bottom
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [existingLogs, streamedLogs, activeTab]);

  const existingMessages = existingLogs?.map((l) => l.message) ?? [];
  const allLogs = [...existingMessages, ...streamedLogs];
  const isStreaming = !TERMINAL_STATUSES.has(deployment.status);

  return (
    <div className="rounded-xl border border-white/7 overflow-hidden">
      {/* Terminal titlebar */}
      <div className="flex items-center justify-between px-4 py-2.5 border-b border-white/6 bg-[#111111]">
        <div className="flex items-center gap-3">
          <div className="flex gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-white/8" />
            <span className="w-2.5 h-2.5 rounded-full bg-white/8" />
            <span className="w-2.5 h-2.5 rounded-full bg-white/8" />
          </div>
          <span className="text-xs text-gray-500 font-mono">
            {parsed
              ? `${parsed.owner}/${parsed.repo}`
              : deployment.id.slice(0, 8)}
          </span>
        </div>
        {isStreaming ? (
          <span className="inline-flex items-center gap-1.5 text-xs text-yellow-400">
            <span className="w-1.5 h-1.5 rounded-full bg-yellow-400 animate-pulse" />
            Live
          </span>
        ) : (
          <span className="inline-flex items-center gap-1.5 text-xs text-gray-600">
            <span
              className={`w-1.5 h-1.5 rounded-full ${deployment.status === "failed" ? "bg-red-500" : "bg-green-500"}`}
            />
            {deployment.status === "failed" ? "Failed" : "Done"}
          </span>
        )}
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-1 px-4 pt-2 border-b border-white/6 bg-[#0d0d0d]">
        {(["build", "runtime"] as Tab[]).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-3 py-1.5 text-xs font-medium rounded-t transition-colors capitalize ${activeTab === tab
                ? "text-white bg-white/7 border border-b-0 border-white/8"
                : "text-gray-600 hover:text-gray-400"
              }`}
          >
            {tab === "build" ? "Build" : "Runtime"}
          </button>
        ))}
      </div>

      {/* Log content */}
      <div
        ref={scrollRef}
        className="overflow-y-auto bg-[#0d0d0d] p-4 font-mono text-[11px] leading-relaxed min-h-64 max-h-140"
      >
        {isLoading ? (
          <span className="text-gray-700">Loading…</span>
        ) : allLogs.length === 0 ? (
          <span className="text-gray-700">Waiting for logs…</span>
        ) : (
          allLogs.map((msg, i) => (
            <div
              key={i}
              className="flex gap-4 hover:bg-white/2 px-1 -mx-1 rounded"
            >
              <span className="shrink-0 w-6 text-right text-gray-700 select-none">
                {i + 1}
              </span>
              <span
                className={`whitespace-pre-wrap break-all ${getLineClass(msg)}`}
              >
                {msg}
              </span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
