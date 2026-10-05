// API 类型契约 — 与 Go 侧响应字段对齐.
export interface Pagination {
  page: number;
  page_size: number;
  total: number;
  total_page: number;
}

export interface Category {
  id: number;
  name: string;
  slug: string;
  description: string;
}

export interface Image {
  id: number;
  source_url: string;
  width: number | null;
  height: number | null;
  format: string;
  source: string;
  status: string;
  fetch_fails: number;
  category_id: number;
  category: Category | null;
  created_at: string;
  updated_at: string;
}

export interface ImageListResponse {
  data: Image[];
  pagination: Pagination;
}

export interface Channel {
  id: number;
  channel_id: string;
  rate_limit: number;
  allowed_origins: string[] | null;
  status: string;
  created_at: string;
  last_used_at: string;
}

export interface ChannelListResponse {
  data: Channel[];
}

export interface OverviewStats {
  total_images: number;
  active_channels: number;
  today_calls: number;
  total_calls: number;
}

export interface StatLog {
  created_at: string;
  channel_id: string;
}

export interface StatsResponse {
  logs: StatLog[];
  total: number;
}

export interface ImportPreviewCheck {
  url: string;
  ok: boolean;
  detail: string;
}

export interface ImportPreview {
  matched: number;
  samples: string[];
  checks: ImportPreviewCheck[];
  probe_ok: number;
  probe_total: number;
}

export interface ImportResult {
  imported: number;
  skipped: number;
  fetch_pending: number;
}

export interface BatchCreateResult {
  count: number;
  fetch_pending?: number;
}
