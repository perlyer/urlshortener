import { useState, type SyntheticEvent } from "react";
import { Link } from "react-router-dom";
import { createLink, type CreateResponse } from "../api";

export function CreatePage() {
  const [url, setUrl] = useState("");
  const [result, setResult] = useState<CreateResponse | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: SyntheticEvent) {
    e.preventDefault();
    setError("");
    setResult(null);
    setLoading(true);
    try {
      setResult(await createLink(url));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="page">
      <div className="hero">
        <h1 className="title">
          Сократи ссылку.<br />
          <span className="muted">Смотри, кто по ней кликает.</span>
        </h1>
        <p className="subtitle">
          Короткие ссылки с аналитикой в реальном времени - устройства, страны,
          источники и уникальные посетители.
        </p>

        <form onSubmit={handleSubmit} className="form">
          <input
            className="input"
            type="url"
            required
            placeholder="https://очень-длинная-ссылка.com/…"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
          />
          <button className="btn btn-primary" type="submit" disabled={loading}>
            {loading ? "…" : "Сократить"}
          </button>
        </form>

        {error && <p className="error">{error}</p>}

        {result && (
          <div className="result card">
            <a className="short-url" href={result.short_url}>
              {result.short_url}
            </a>
            <div className="result-actions">
              <button
                className="btn btn-ghost"
                onClick={() => navigator.clipboard.writeText(result.short_url)}
              >
                Копировать
              </button>
              <Link className="btn btn-ghost" to={`/stats/${result.code}`}>
                Статистика →
              </Link>
            </div>
          </div>
        )}
      </div>
    </main>
  );
}
