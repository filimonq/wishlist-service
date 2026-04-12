import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/api/v1';

const reserveSuccess  = new Counter('reserve_success');
const reserveConflict = new Counter('reserve_conflict');
const reserveError    = new Counter('reserve_error');

export function setup() {
  const headers = { 'Content-Type': 'application/json' };

  const reg = http.post(`${BASE_URL}/auth/register`,
    JSON.stringify({ email: `race_${Date.now()}@example.com`, password: 'password123' }), { headers });

  const login = http.post(`${BASE_URL}/auth/login`,
    JSON.stringify({ email: JSON.parse(reg.body).email, password: 'password123' }), { headers });
  const token = JSON.parse(login.body).token;

  const authHeaders = { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` };

  const wl = http.post(`${BASE_URL}/wishlists`,
    JSON.stringify({ title: 'Race', description: '', event_date: '2026-06-01T00:00:00Z' }), { headers: authHeaders });
  const wishlist = JSON.parse(wl.body);

  const item = http.post(`${BASE_URL}/wishlists/${wishlist.id}/items`,
    JSON.stringify({ name: 'Спорный подарок', description: '', priority: 5 }), { headers: authHeaders });
  const itemData = JSON.parse(item.body);

  return { publicToken: wishlist.public_token, itemID: itemData.id };
}

export const options = {
  vus: 50,
  duration: '15s',
  thresholds: {
    reserve_success:  ['count==1'],
    reserve_error:    ['count==0'],
  },
};

export default function ({ publicToken, itemID }) {
  const res = http.post(`${BASE_URL}/public/${publicToken}/items/${itemID}/reserve`);

  if (res.status === 204)      reserveSuccess.add(1);
  else if (res.status === 409) reserveConflict.add(1);
  else                         reserveError.add(1);

  sleep(0.1);
}