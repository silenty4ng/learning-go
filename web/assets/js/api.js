/* ==========================================================================
   iGoBlog · api.js — 统一 API 封装
   - 自动携带 Bearer Token；
   - 解析后端统一响应：成功 {message, data}；失败 {code, message}；
   - 401 清除登录态并派发事件（供页面弹出登录层 / 跳转）；
   - 网络异常 / 限流统一转为带语义的错误对象。
   ========================================================================== */

import { getToken, clearAuth } from './auth.js';

/** 业务错误：status 为 HTTP 状态码，code 为后端业务错误码。 */
export class ApiError extends Error {
  constructor(status, code, message) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

async function request(method, path, body) {
  const headers = {};
  const token = getToken();
  if (token) headers['Authorization'] = `Bearer ${token}`;
  if (body !== undefined) headers['Content-Type'] = 'application/json';

  let res;
  try {
    res = await fetch(path, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch {
    throw new ApiError(0, 'NETWORK_ERROR', '网络连接失败，请确认服务已启动');
  }

  let payload = {};
  try {
    payload = await res.json();
  } catch {
    /* 非 JSON 响应（如空 body），按空对象处理 */
  }

  if (!res.ok) {
    // 登录态失效：清除本地凭据并广播事件
    if (res.status === 401) {
      clearAuth();
      window.dispatchEvent(new CustomEvent('auth:expired'));
    }
    const message =
      res.status === 429
        ? '请求过于频繁，请稍后再试'
        : payload.message || '请求失败';
    throw new ApiError(res.status, payload.code || 'ERROR', message);
  }

  return payload.data;
}

export const api = {
  get: (path) => request('GET', path),
  post: (path, body) => request('POST', path, body ?? {}),
  put: (path, body) => request('PUT', path, body ?? {}),
  del: (path) => request('DELETE', path),

  /** 获取当前登录用户信息（未登录返回 null）。 */
  async currentUser() {
    if (!getToken()) return null;
    try {
      return await this.get('/api/v1/users/me');
    } catch {
      return null;
    }
  },
};
