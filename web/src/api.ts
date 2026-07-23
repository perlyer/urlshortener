// Слой доступа к API сокращателя. Базовый адрес api-сервиса.
const API_BASE = "http://localhost:8080";

export interface CreateResponse {
  code: string;
  short_url: string;
}

export interface StatItem {
  value: string;
  count: number;
}

export interface StatsResponse {
  code: string;
  unique_visitors: number;
  stats: Record<string, StatItem[]>;
}

export async function createLink(url: string): Promise<CreateResponse> {
  const res = await fetch(`${API_BASE}/api/links`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url }),
  });
  if (!res.ok) {
    throw new Error(`Не удалось создать ссылку (${res.status})`);
  }
  return res.json();
}

export async function fetchStats(code: string): Promise<StatsResponse> {
  const res = await fetch(`${API_BASE}/api/links/${code}/stats`);
  if (!res.ok) {
    throw new Error(`Не удалось получить статистику (${res.status})`);
  }
  return res.json();
}
