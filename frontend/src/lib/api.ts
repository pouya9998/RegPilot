import axios from "axios";

const api = axios.create({
  baseURL: "/api",
  headers: { "Content-Type": "application/json" },
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem("regpilot_token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (r) => r,
  (error) => {
    if (error.response?.status === 401) {
      const path = window.location.pathname;
      if (!path.startsWith("/login")) {
        localStorage.removeItem("regpilot_token");
        localStorage.removeItem("regpilot_user");
        window.location.href = "/login";
      }
    }
    const msg =
      error.response?.data?.error?.message ||
      error.message ||
      "Request failed";
    return Promise.reject(new Error(msg));
  }
);

export default api;

export type User = {
  id: number;
  username: string;
  role: "ADMIN" | "VIEWER";
  disabled?: boolean;
};

export type Registry = {
  id: number;
  name: string;
  url: string;
  username?: string;
  has_password: boolean;
  insecure_skip_verify: boolean;
  created_at: string;
  updated_at: string;
};

export type Health = {
  status: "healthy" | "unhealthy" | "unauthorized" | "unreachable";
  latency_ms: number;
  message?: string;
};

export type RepoSummary = {
  name: string;
  tag_count: number;
  latest_tag?: string;
};

export type TagInfo = {
  tag: string;
  digest?: string;
  created?: string;
  size?: number;
};

export type Manifest = {
  digest: string;
  media_type: string;
  schema_version: number;
  architecture?: string;
  os?: string;
  created?: string;
  layers: { media_type: string; digest: string; size: number }[];
  total_size: number;
};

export type AuditLog = {
  id: number;
  user_id?: number;
  username: string;
  action: string;
  registry_id?: number;
  repository?: string;
  tag?: string;
  digest?: string;
  metadata?: string;
  result?: string;
  created_at: string;
};

export type StorageInfo = {
  available: boolean;
  message: string;
  detail: string;
};

export type Dashboard = {
  registry_count: number;
  repository_count: number;
  tag_count: number;
  tag_count_partial?: boolean;
  storage: StorageInfo;
  /** @deprecated use storage */
  storage_available: boolean;
  /** @deprecated use storage */
  storage_message: string;
  registries: {
    id: number;
    name: string;
    status: Health["status"];
    latency_ms: number;
    repositories: number;
    tags: number;
  }[];
  recent_activity: AuditLog[];
};

export type GCResult = {
  supported: boolean;
  dry_run: boolean;
  message: string;
  reclaimable_blobs?: number;
  reclaimable_bytes?: number;
  reclaimable_human?: string;
  output?: string;
};

export type CleanupPreview = {
  repository: string;
  tags_would_delete: string[];
  count: number;
  estimated_reclaim_bytes: number;
  estimated_reclaim_human: string;
  keep_latest: number;
  delete_older_days: number;
};
