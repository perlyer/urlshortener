import http from "k6/http";
import { check } from "k6";

// Нагрузочный тест горячего пути - редиректа. Сначала создаём пул коротких
// ссылок через api, затем бомбим redirector случайными кодами.

const API = __ENV.API || "http://localhost:8080";
const REDIRECT = __ENV.REDIRECT || "http://localhost:8081";

export const options = {
  vus: 50,
  duration: "30s",
  thresholds: {
    http_req_duration: ["p(95)<50"], // 95% редиректов быстрее 50 мс
    checks: ["rate>0.99"],
  },
};

export function setup() {
  const codes = [];
  for (let i = 0; i < 20; i++) {
    const res = http.post(
      `${API}/api/links`,
      JSON.stringify({ url: `https://example.com/page/${i}` }),
      { headers: { "Content-Type": "application/json" } },
    );
    if (res.status === 201) codes.push(res.json("code"));
  }
  if (codes.length === 0) {
    throw new Error("не удалось создать ссылки - запущен ли api на " + API + "?");
  }
  return { codes };
}

export default function (data) {
  const code = data.codes[Math.floor(Math.random() * data.codes.length)];
  const res = http.get(`${REDIRECT}/${code}`, { redirects: 0 });
  check(res, { "статус 302": (r) => r.status === 302 });
}
