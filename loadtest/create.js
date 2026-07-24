import http from "k6/http";
import { check } from "k6";
import { Counter } from "k6/metrics";

// Нагрузочный тест пути записи - создания ссылок. Бомбим POST /api/links
// с одного клиента и наблюдаем, как token-bucket лимитер режет поток в 429
// после исчерпания burst.

const API = __ENV.API || "http://localhost:8080";

const created = new Counter("links_created"); // 201
const limited = new Counter("links_limited"); // 429

export const options = {
  vus: 30,
  duration: "20s",
};

export default function () {
  const res = http.post(
    `${API}/api/links`,
    JSON.stringify({ url: `https://example.com/${Math.random()}` }),
    { headers: { "Content-Type": "application/json" } },
  );

  if (res.status === 201) created.add(1);
  else if (res.status === 429) limited.add(1);

  check(res, {
    "201 или 429 (не 5xx)": (r) => r.status === 201 || r.status === 429,
  });
}
