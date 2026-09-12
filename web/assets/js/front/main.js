/* ==========================================================================
   iGoBlog 前台 · main.js — 前台入口
   组装 hash 路由（首页 / 文章详情）、顶部登录态、登录注册弹层、移动端菜单。
   ========================================================================== */

import { api } from '../api.js';
import {
  isLoggedIn, setToken, clearAuth, getSavedUser, setSavedUser,
} from '../auth.js';
import { createRouter } from '../router.js';
import { toast } from '../ui.js';
import { renderHome } from './home.js';
import { renderArticle } from './article.js';

const app = document.getElementById('app');
const authArea = document.getElementById('authArea');

/* ---------- 路由 ---------- */
const router = createRouter(app, [
  { path: '/', view: renderHome },
  { path: '/article/:id', view: renderArticle },
]);
router.start();

/* ---------- 顶部登录态 ---------- */
/** 展示名：昵称优先，未设置时回退用户名。 */
function displayName(user) {
  return (user?.nickname || '').trim() || user?.username || '';
}

export function renderAuthArea() {
  authArea.replaceChildren();
  if (isLoggedIn()) {
    const user = getSavedUser();
    const chip = document.createElement('button');
    chip.className = 'user-chip';
    chip.type = 'button';
    chip.title = '点击修改昵称';
    chip.style.cssText = 'border:none;background:none;cursor:pointer;padding:4px 6px;border-radius:8px;transition:background .18s ease';
    chip.innerHTML = `
      <span class="avatar">${displayName(user)[0].toUpperCase()}</span>
      <b>${displayName(user)}</b>`;
    chip.addEventListener('click', openProfileModal);
    authArea.appendChild(chip);

    const logout = document.createElement('button');
    logout.className = 'link-logout';
    logout.textContent = '退出';
    logout.addEventListener('click', () => {
      clearAuth();
      renderAuthArea();
      toast('已退出登录', 'info');
    });
    authArea.appendChild(logout);
  } else {
    const btn = document.createElement('button');
    btn.className = 'btn btn-login';
    btn.textContent = '登录 / 注册';
    btn.addEventListener('click', () => openAuthModal());
    authArea.appendChild(btn);
  }
}

/* ---------- 个人设置（修改昵称） ---------- */
const profileModal = document.getElementById('profileModal');
const profileInput = document.getElementById('profileNickname');
const profileError = document.getElementById('profileError');

export function openProfileModal() {
  const user = getSavedUser();
  profileInput.value = user?.nickname ?? '';
  profileError.textContent = '';
  profileModal.hidden = false;
  profileInput.focus();
}

document.getElementById('profileSave').addEventListener('click', async (e) => {
  profileError.textContent = '';
  const btn = e.currentTarget;
  btn.disabled = true;
  try {
    const u = await api.put('/api/v1/users/me', { nickname: profileInput.value.trim() });
    setSavedUser(u);
    profileModal.hidden = true;
    renderAuthArea();
    toast(`昵称已更新为「${displayName(u)}」`, 'success');
    // 刷新当前视图（文章作者、评论区等处的展示名）
    window.dispatchEvent(new HashChangeEvent('hashchange'));
  } catch (err) {
    profileError.textContent = err.message;
  } finally {
    btn.disabled = false;
  }
});

profileModal.addEventListener('click', (e) => {
  if (e.target === profileModal) profileModal.hidden = true;
});

/* ---------- 登录 / 注册弹层 ---------- */
const authModal = document.getElementById('authModal');
const authTabs = document.getElementById('authTabs');
const loginForm = document.getElementById('loginForm');
const registerForm = document.getElementById('registerForm');
const regClosedNote = document.getElementById('regClosedNote');

let siteSettings = null;

/** 拉取站点设置（注册开关、站点名称），失败时按默认值兜底。 */
async function loadSiteSettings() {
  try {
    siteSettings = await api.get('/api/v1/site/settings');
    if (siteSettings.site_title) {
      document.title = `${siteSettings.site_title} · 首页`;
      const brand = document.querySelector('.brand');
      // 保留品牌高亮后缀的样式：站点名 + 隐藏的后缀
      brand.innerHTML = `${escapeSiteTitle(siteSettings.site_title)}`;
    }
  } catch {
    siteSettings = null;
  }
}

function escapeSiteTitle(s) {
  const d = document.createElement('div');
  d.textContent = s;
  return d.innerHTML;
}

let currentTab = 'login';

/** 依据当前站点设置应用注册页签的可见性，并切换到目标页签。 */
function applyAuthTab() {
  const regAllowed = siteSettings ? siteSettings.registration_enabled !== false : true;
  // 注册关闭时隐藏注册页签并提示
  authTabs.querySelector('[data-tab="register"]').style.display = regAllowed ? '' : 'none';
  regClosedNote.hidden = regAllowed;
  if (!regAllowed && currentTab === 'register') currentTab = 'login';
  switchTab(currentTab);
}

export function openAuthModal(tab = 'login') {
  currentTab = tab;
  authModal.hidden = false;
  applyAuthTab(); // 立即以已知设置渲染，保证点击即时反馈
  // 设置到达后修正注册开关（首次打开或后台刚改过设置的场景）
  loadSiteSettings().then(applyAuthTab).catch(() => {});
  (currentTab === 'login' ? loginForm : registerForm).querySelector('input')?.focus();
}

function switchTab(tab) {
  authTabs.querySelectorAll('button').forEach((b) =>
    b.classList.toggle('active', b.dataset.tab === tab));
  loginForm.hidden = tab !== 'login';
  registerForm.hidden = tab !== 'register';
}

authTabs.addEventListener('click', (e) => {
  const tab = e.target.dataset?.tab;
  if (tab) switchTab(tab);
});

authModal.addEventListener('click', (e) => {
  if (e.target === authModal) authModal.hidden = true;
});
document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape' && !authModal.hidden) authModal.hidden = true;
});

/* 登录提交 */
loginForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  const errEl = document.getElementById('loginError');
  errEl.textContent = '';
  const username = loginForm.username.value.trim();
  const password = loginForm.password.value;
  if (!username || !password) {
    errEl.textContent = '请输入用户名与密码';
    return;
  }
  try {
    const data = await api.post('/api/v1/auth/login', { username, password });
    setToken(data.token);
    const me = await api.currentUser();
    setSavedUser(me);
    authModal.hidden = true;
    loginForm.reset();
    renderAuthArea();
    toast(`欢迎回来，${me?.username ?? username}`, 'success');
    // 刷新当前视图（如评论区出现发表框）
    window.dispatchEvent(new HashChangeEvent('hashchange'));
  } catch (err) {
    errEl.textContent = err.message;
  }
});

/* 注册提交 */
registerForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  const errEl = document.getElementById('regError');
  errEl.textContent = '';
  const username = registerForm.username.value.trim();
  const password = registerForm.password.value;
  const confirm = registerForm.confirm.value;
  if (username.length < 3 || password.length < 6) {
    errEl.textContent = '用户名至少 3 位，密码至少 6 位';
    return;
  }
  if (password !== confirm) {
    errEl.textContent = '两次输入的密码不一致';
    return;
  }
  try {
    await api.post('/api/v1/auth/register', { username, password });
    const data = await api.post('/api/v1/auth/login', { username, password });
    setToken(data.token);
    const me = await api.currentUser();
    setSavedUser(me);
    authModal.hidden = true;
    registerForm.reset();
    renderAuthArea();
    toast('注册成功，已自动登录', 'success');
    window.dispatchEvent(new HashChangeEvent('hashchange'));
  } catch (err) {
    errEl.textContent = err.message;
  }
});

/* 401（Token 过期）时引导重新登录 */
window.addEventListener('auth:expired', () => {
  renderAuthArea();
  toast('登录状态已失效，请重新登录', 'warning');
});

/* 详情页等视图请求打开登录弹层 */
window.addEventListener('front:open-auth', () => openAuthModal('login'));

/* ---------- 移动端菜单 ---------- */
const navToggle = document.getElementById('navToggle');
const siteNav = document.getElementById('siteNav');
navToggle.addEventListener('click', () => siteNav.classList.toggle('open'));
siteNav.addEventListener('click', (e) => {
  if (e.target.tagName === 'A') siteNav.classList.remove('open');
});

renderAuthArea();
loadSiteSettings();
