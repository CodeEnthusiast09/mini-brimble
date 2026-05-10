export type DeploymentStatus =
  | "pending"
  | "building"
  | "deploying"
  | "running"
  | "failed";

export type Deployment = {
  id: string;
  github_url: string;
  status: DeploymentStatus;
  image_tag?: string;
  container_port?: number;
  container_id?: string;
  live_url?: string;
  created_at: string;
  updated_at: string;
};

export type LogEntry = {
  id: string;
  deployment_id: string;
  message: string;
  created_at: string;
};

export type ApiResponse<T> = {
  success: boolean;
  message: string;
  data: T;
};
