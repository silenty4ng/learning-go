/* ==========================================================================
   iGoBlog 后台 · app.js — 后台入口
   登录门禁（未登录跳转前台）→ 渲染用户信息 → 注册后台 hash 路由。
   ========================================================================== */

import { api } from '../api.js';
import { isLoggedIn, getSavedUser, setSavedUser, clearAuth } from '../auth.js';
import { createRouter } from '../router.js';
import { toast } from '../ui.js';
import { renderPosts } from './posts.js';
import { renderEditor } from './editor.js';
import { renderCategories, renderTags } from './taxonomies.js';
import { renderComments } from './comments.js';
import { renderUsers } from './users.js';
import { renderSettings } from './settings.js';

const app = document.getElementById('app');
const titleEl = document.getElementById('adminTitle');

/* ---------- 登录门禁 ---------- */
if (!isLoggedIn()) {
  toast('请先登录后再访问后台', 'warning');
  setTimeout(() => (location.href = '/'), 800);
  throw new Error('unauthenticated'); // 阻止后续模块逻辑
}

(async () => {
  const me = getSavedUser() ?? (await api.currentUser());
  if (!me) {
    clearAuth();
    location.href = '/';
    return;
  }
  setSavedUser(me);
  renderUserChip(me);
})();

function renderUserChip(me) {
  const roleText = me.role === 'admin' ? '管理员' : '用户';
  document.getElementById('adminUser').innerHTML = `
    <div class="admin-user-card">
      <span class="avatar">${(me.username || '?')[0].toUpperCase()}</span>
      <div class="admin-user-card__info">
        <b title="${me.username}">${me.username}</b>
        <small>${roleText}</small>
      </div>
      <button class="btn-icon" id="adminLogout" title="退出登录" aria-label="退出登录">
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor"
          stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
          <polyline points="16 17 21 12 16 7"/>
          <line x1="21" y1="12" x2="9" y2="12"/>
        </svg>
      </button>
    </div>`;
  document.getElementById('adminLogout').addEventListener('click', () => {
    clearAuth();
    location.href = '/';
  });
}

/* ---------- 路由 ---------- */
const TITLES = {
  posts: '文章管理',
  editor: '文章编辑',
  categories: '分类管理',
  tags: '标签管理',
  comments: '评论管理',
  users: '用户管理',
  settings: '站点设置',
};

function markActive(name) {
  document.querySelectorAll('#adminNav a').forEach((a) =>
    a.classList.toggle('active', a.dataset.route === name));
}

const router = createRouter(app, [
  // 默认页：访问 /admin（无 hash）时重定向到文章管理
  { path: '/', view: () => { location.replace('#/posts'); } },
  { path: '/posts', view: (ctx) => (setTitle('posts'), markActive('posts'), renderPosts(ctx)) },
  { path: '/posts/new', view: (ctx) => (setTitle('editor'), markActive('posts'), renderEditor(ctx)) },
  { path: '/posts/:id/edit', view: (ctx) => (setTitle('editor'), markActive('posts'), renderEditor(ctx)) },
  { path: '/categories', view: (ctx) => (setTitle('categories'), markActive('categories'), renderCategories(ctx)) },
  { path: '/tags', view: (ctx) => (setTitle('tags'), markActive('tags'), renderTags(ctx)) },
  { path: '/comments', view: (ctx) => (setTitle('comments'), markActive('comments'), renderComments(ctx)) },
  { path: '/users', view: (ctx) => (setTitle('users'), markActive('users'), renderUsers(ctx)) },
  { path: '/settings', view: (ctx) => (setTitle('settings'), markActive('settings'), renderSettings(ctx)) },
]);
router.start();

function setTitle(key) {
  titleEl.textContent = TITLES[key] ?? '管理后台';
}

/* ---------- 移动端侧边栏 ---------- */
const sidebar = document.getElementById('sidebar');
const mask = document.getElementById('sidebarMask');
document.getElementById('adminNavToggle').addEventListener('click', () => {
  sidebar.classList.toggle('open');
  mask.classList.toggle('show', sidebar.classList.contains('open'));
});
mask.addEventListener('click', closeSidebar);
document.getElementById('adminNav').addEventListener('click', (e) => {
  if (e.target.tagName === 'A') closeSidebar();
});
function closeSidebar() {
  sidebar.classList.remove('open');
  mask.classList.remove('show');
}
