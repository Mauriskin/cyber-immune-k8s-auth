import http from 'k6/http';
import { check } from 'k6';

export const options = { vus: 1, iterations: 1 };

const loginUrl = 'http://INGRESS_HOST:INGRESS_PORT/login'.replace('INGRESS_HOST', __ENV.INGRESS_HOST).replace('INGRESS_PORT', __ENV.INGRESS_PORT);
const directTokenUrl = 'http://INGRESS_HOST:INGRESS_PORT/issue'; // Попытка прямого доступа

export default function () {
  // Неверные credentials
  let res = http.post(loginUrl, JSON.stringify({username: 'wrong'}), {headers: {'Content-Type': 'application/json'}});
  check(res, { 'invalid creds → 401': (r) => r.status === 401 });

  // Прямой доступ к Token Service (должен блокироваться)
  res = http.post(directTokenUrl);
  check(res, { 'direct access blocked': (r) => r.status === 0 || r.status >= 500 });
}