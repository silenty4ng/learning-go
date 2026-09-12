/* ==========================================================================
   iGoBlog · auth.js — 登录态存取（localStorage）
   只负责存取，不发起网络请求，避免与 api.js 循环依赖。
   ========================================================================== */

const TOKEN_KEY = 'igoblog.token';
const USER_KEY = 'igoblog.user';

/** 读取 token，未登录返回 null。 */
export function getToken() {
  return localStorage.getItem(TOKEN_KEY);
}

/** 保存 token。 */
export function setToken(token) {
  localStorage.setItem(TOKEN_KEY, token);
}

/** 清除登录态。 */
export function clearAuth() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
}

/** 是否已登录（仅判断本地状态，有效性由 api 调用时的 401 保证）。 */
export function isLoggedIn() {
  return Boolean(localStorage.getItem(TOKEN_KEY));
}

/** 读取缓存的当前用户信息（{id, username}）。 */
export function getSavedUser() {
  try {
    return JSON.parse(localStorage.getItem(USER_KEY));
  } catch {
    return null;
  }
}

/** 缓存当前用户信息。 */
export function setSavedUser(user) {
  localStorage.setItem(USER_KEY, JSON.stringify(user));
}
