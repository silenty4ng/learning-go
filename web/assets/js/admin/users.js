/* ==========================================================================
   iGoBlog 后台 · users.js — 用户管理（超管）
   用户列表、角色切换（设为管理员/降为普通用户）、删除（级联其内容）。
   后端约束：不能操作自己的降级/删除；站点至少保留一名管理员。
   ========================================================================== */

import { api } from '../api.js';
import { getSavedUser } from '../auth.js';
import { escapeHtml, formatDate } from '../format.js';
import { toast, confirmDialog, renderPagination, renderEmpty } from '../ui.js';

const PAGE_SIZE = 20;

export function renderUsers({ outlet }) {
  outlet.innerHTML = `
    <div class="card panel">
      <div class="table-wrap">
        <table class="table">
          <thead>
            <tr><th>ID</th><th>用户名</th><th>昵称</th><th>角色</th><th>注册时间</th><th style="text-align:right">操作</th></tr>
          </thead>
          <tbody data-role="rows"></tbody>
        </table>
      </div>
      <div data-role="empty"></div>
    </div>
    <div data-role="pager"></div>
    <p class="muted" style="margin-top:12px">
      ⚠️ 删除用户将同时删除其全部文章与评论；站点至少保留一名管理员，且不能操作自己的账号。
    </p>
  `;

  const rowsEl = outlet.querySelector('[data-role="rows"]');
  const emptyEl = outlet.querySelector('[data-role="empty"]');
  const pagerEl = outlet.querySelector('[data-role="pager"]');
  const me = getSavedUser();

  let page = 1;

  async function load() {
    rowsEl.innerHTML = `<tr><td colspan="6"><div class="skeleton" style="height:20px;margin:10px 4px"></div></td></tr>`;
    emptyEl.innerHTML = '';
    try {
      const { items, total } = await api.get(`/api/v1/admin/users?page=${page}&page_size=${PAGE_SIZE}`);
      if (!items || items.length === 0) {
        rowsEl.innerHTML = '';
        renderEmpty(emptyEl, '暂无用户', '👥');
        return;
      }
      rowsEl.replaceChildren(...items.map((u) => row(u)));
      renderPagination(pagerEl, { page, pageSize: PAGE_SIZE, total }, (p) => {
        page = p;
        load();
      });
    } catch (err) {
      rowsEl.innerHTML = '';
      renderEmpty(emptyEl, `加载失败：${err.message}`, '⚠️');
    }
  }

  function row(u) {
    const isSelf = me && me.id === u.id;
    const tr = document.createElement('tr');
    tr.innerHTML = `
      <td class="muted">${u.id}</td>
      <td style="font-weight:500">
        <span class="row" style="gap:8px">
          <span class="avatar" style="width:26px;height:26px;font-size:12px">${escapeHtml((u.nickname || u.username)[0].toUpperCase())}</span>
          ${escapeHtml(u.username)}
          ${isSelf ? '<span class="badge badge--muted">我</span>' : ''}
        </span>
      </td>
      <td>${u.nickname ? escapeHtml(u.nickname) : '<span class="muted">-</span>'}</td>
      <td>
        <span class="badge ${u.role === 'admin' ? '' : 'badge--muted'}">
          ${u.role === 'admin' ? '管理员' : '用户'}
        </span>
      </td>
      <td class="muted">${formatDate(u.created_at)}</td>
      <td class="actions">
        ${isSelf ? '<span class="muted">当前登录账号</span>' : `
          <button class="btn btn--ghost btn--sm" data-role-btn>${u.role === 'admin' ? '降为用户' : '设为管理员'}</button>
          <button class="btn btn--danger btn--sm" data-del-btn>删除</button>
        `}
      </td>`;

    const roleBtn = tr.querySelector('[data-role-btn]');
    if (roleBtn) {
      roleBtn.addEventListener('click', async () => {
        const target = u.role === 'admin' ? 'user' : 'admin';
        const ok = await confirmDialog(
          `确定将用户「${u.username}」${target === 'admin' ? '设为管理员' : '降级为普通用户'}吗？`,
          { danger: target !== 'admin', okText: '确定' }
        );
        if (!ok) return;
        try {
          await api.put(`/api/v1/admin/users/${u.id}/role`, { role: target });
          toast('角色已更新', 'success');
          load();
        } catch (err) {
          toast(err.message, 'error'); // 409：最后一名管理员 / 400：自我降级
        }
      });
    }

    const delBtn = tr.querySelector('[data-del-btn]');
    if (delBtn) {
      delBtn.addEventListener('click', async () => {
        const ok = await confirmDialog(
          `确定删除用户「${u.username}」吗？其全部文章与评论将一并删除，此操作不可恢复。`);
        if (!ok) return;
        try {
          await api.del(`/api/v1/admin/users/${u.id}`);
          toast('用户已删除', 'success');
          load();
        } catch (err) {
          toast(err.message, 'error');
        }
      });
    }
    return tr;
  }

  load();
}
