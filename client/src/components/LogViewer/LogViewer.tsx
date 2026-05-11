import { useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { getLogs, getLogStreamUrl } from "../../api/logs";
import type { Deployment, LogEntry } from "../../types";
import { parseGitHubRepo } from "../../lib/utils";

const TERMINAL_STATUSES = new Set<string>(["running", "failed"]);

type Tab = "build" | "runtime";

type LogEvent = {
  id: string;
  message: string;
  created_at: string;
};

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
  const [streamedLogs, setStreamedLogs] = useState<LogEvent[]>([]);
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
      try {
        const parsed = JSON.parse(String(e.data)) as LogEvent;
        setStreamedLogs((prev) => [...prev, parsed]);
      } catch {
        // ignore malformed payloads
      }
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

  const allLogs = useMemo(() => {
    const seen = new Set<string>();
    const merged: { id: string; message: string }[] = [];

    for (const entry of existingLogs ?? ([] as LogEntry[])) {
      if (!seen.has(entry.id)) {
        seen.add(entry.id);
        merged.push({ id: entry.id, message: entry.message });
      }
    }

    for (const entry of streamedLogs) {
      if (!seen.has(entry.id)) {
        seen.add(entry.id);
        merged.push({ id: entry.id, message: entry.message });
      }
    }

    return merged;
  }, [existingLogs, streamedLogs]);

  const isStreaming = !TERMINAL_STATUSES.has(deployment.status);

  return (
    <div className="rounded-xl border border-white/7 overflow-hidden">
      <div className="flex items-center justify-between px-4 py-2.5 border-b border-white/6 bg-[#111111]">
        <span className="text-xs text-gray-500 font-mono">
          {parsed ? `${parsed.owner}/${parsed.repo}` : deployment.id.slice(0, 8)}
        </span>
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
          allLogs.map((entry, i) => (
            <div
              key={entry.id}
              className="flex gap-4 hover:bg-white/2 px-1 -mx-1 rounded"
            >
              <span className="shrink-0 w-6 text-right text-gray-700 select-none">
                {i + 1}
              </span>
              <span
                className={`whitespace-pre-wrap break-all ${getLineClass(entry.message)}`}
              >
                {entry.message}
              </span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
