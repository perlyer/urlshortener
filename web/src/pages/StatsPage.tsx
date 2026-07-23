import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  Bar,
  BarChart,
  LabelList,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { fetchStats, type StatItem, type StatsResponse } from "../api";

const DIMENSIONS: { key: string; title: string }[] = [
  { key: "country", title: "Страны" },
  { key: "device", title: "Устройства" },
  { key: "browser", title: "Браузеры" },
  { key: "os", title: "Операционные системы" },
  { key: "referer", title: "Источники" },
  { key: "language", title: "Языки" },
  { key: "hour", title: "По времени" },
];

function DimensionChart({
  title,
  data,
  formatLabel,
}: {
  title: string;
  data: StatItem[];
  formatLabel?: (v: string) => string;
}) {
  const sorted = [...data]
    .sort((a, b) => b.count - a.count)
    .slice(0, 8)
    .map((d) => ({ ...d, value: formatLabel ? formatLabel(d.value) : d.value }));
  const height = Math.max(120, sorted.length * 40 + 16);

  return (
    <div className="card chart-card">
      <h3 className="chart-title">{title}</h3>
      <ResponsiveContainer width="100%" height={height}>
        <BarChart data={sorted} layout="vertical" margin={{ left: 0, right: 44, top: 4, bottom: 4 }}>
          <XAxis type="number" hide />
          <YAxis
            type="category"
            dataKey="value"
            width={116}
            tick={{ fill: "#8a8a8a", fontSize: 13 }}
            axisLine={false}
            tickLine={false}
          />
          <Tooltip
            cursor={{ fill: "rgba(255,255,255,0.04)" }}
            contentStyle={{
              background: "#1b1b1b",
              border: "1px solid rgba(255,255,255,0.12)",
              borderRadius: 8,
              color: "#ededed",
            }}
            labelStyle={{ color: "#8a8a8a" }}
          />
          <Bar dataKey="count" fill="#c4f135" radius={[0, 4, 4, 0]} barSize={18}>
            <LabelList dataKey="count" position="right" fill="#ededed" fontSize={13} />
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

export function StatsPage() {
  const { code } = useParams<{ code: string }>();
  const [data, setData] = useState<StatsResponse | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!code) return;
    fetchStats(code)
      .then(setData)
      .catch((e) => setError(e instanceof Error ? e.message : "Ошибка"));
  }, [code]);

  if (error) {
    return (
      <main className="page page-wide">
        <p className="error">{error}</p>
        <Link className="back-link" to="/">← назад</Link>
      </main>
    );
  }

  if (!data) {
    return (
      <main className="page page-wide">
        <p style={{ color: "var(--text-dim)" }}>Загрузка…</p>
      </main>
    );
  }

  const total = data.stats.total?.[0]?.count ?? 0;

  return (
    <main className="page page-wide">
      <header className="stats-header">
        <div>
          <div className="stats-code">/{data.code}</div>
          <Link className="back-link" to="/">← сократить ещё</Link>
        </div>
      </header>

      <div className="tiles">
        <div className="tile card">
          <div className="tile-value accent">{total}</div>
          <div className="tile-label">Всего кликов</div>
        </div>
        <div className="tile card">
          <div className="tile-value">{data.unique_visitors}</div>
          <div className="tile-label">Уникальные посетители</div>
        </div>
      </div>

      <div className="charts-grid">
        {DIMENSIONS.map(
          (d) =>
            data.stats[d.key]?.length > 0 && (
              <DimensionChart
                key={d.key}
                title={d.title}
                data={data.stats[d.key]}
                formatLabel={d.key === "hour" ? (v) => `${v.slice(11)}:00` : undefined}
              />
            ),
        )}
      </div>
    </main>
  );
}
