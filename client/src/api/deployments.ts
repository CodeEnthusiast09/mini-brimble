import type { ApiResponse, Deployment } from "../types";
import { apiFetch } from "./client";

export async function getDeployments(): Promise<Deployment[]> {
  const res = await apiFetch<ApiResponse<Deployment[]>>("/deployments");
  return res.data;
}

export async function createDeployment(
  github_url: string,
): Promise<Deployment> {
  const res = await apiFetch<ApiResponse<Deployment>>("/deployments", {
    method: "POST",
    body: JSON.stringify({ github_url }),
  });
  return res.data;
}

export async function deleteDeployment(id: string): Promise<void> {
  await apiFetch<ApiResponse<null>>(`/deployments/${id}`, { method: "DELETE" });
}
