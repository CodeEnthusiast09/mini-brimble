import type { ApiResponse, LogEntry } from "../types";
import { apiFetch, API_BASE_URL } from "./client";

export async function getLogs(deploymentId: string): Promise<LogEntry[]> {
  const res = await apiFetch<ApiResponse<LogEntry[]>>(
    `/deployments/${deploymentId}/logs`,
  );
  return res.data;
}

export function getLogStreamUrl(deploymentId: string): string {
  return `${API_BASE_URL}/deployments/${deploymentId}/logs/stream`;
}
