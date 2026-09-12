/* ==========================================================================
   iGoBlog 后台 · posts.js — 文章管理列表
   分页表格：标题、分类、浏览量、日期、编辑/删除。
   ========================================================================== */

import { api } from '../api.js';
import { escapeHtml, formatCount, formatDate } from '../format.js';
import { toast, confirmDialog, renderPagination, renderEmpty } from '../ui.js';

const PAGE_SIZE = 10;

export function renderPosts({ outlet }) {
  outlet.innerHTML = `
    <div class="toolbar">
      <div class="search">
        <input class="input" data-role="keyword" placeholder="搜索标题…" />
      </div>
      <a class="btn" href="#/posts/new">＋ 新建文章</a>
    </div>
    <div class="card panel">
      <div class="table-wrap">
        <table class="table">
          <thead>
            <tr><th>标题</th><th>分类</th><th>浏览量</th><th>创建时间</th><th style="text-align:right">操作</th></tr>
          </thead>
          <tbody data-role="rows"></tbody>
        </table>
      </div>
      <div data-role="empty"></div>
    </div>
    <div data-role="pager"></div>
  `;

  const rowsEl = outlet.querySelector('[data-role="rows"]');
  const emptyEl = outlet.querySelector('[data-role="empty"]');
  const pagerEl = outlet.querySelector('[data-role="pager"]');
  const keywordInput = outlet.querySelector('[data-role="keyword"]');

  let page = 1;
  let keyword = '';

  async function load() {
    rowsEl.innerHTML = `<tr><td colspan="5"><div class="skeleton" style="height:20px;margin:10px 4px"></div></td></tr>`;
    emptyEl.innerHTML = '';
    try {
      const qs = new URLSearchParams({ page, page_size: PAGE_SIZE, ...(keyword && { keyword }) });
      const { items, total } = await api.get(`/api/v1/articles?${qs}`);

      if (!items || items.length === 0) {
        rowsEl.innerHTML = '';
        renderEmpty(emptyEl, '还没有文章，点击右上角「新建文章」开始写作', '📝');
        renderPagination(pagerEl, { page, pageSize: PAGE_SIZE, total: 0 }, () => {});
        return;
      }
      rowsEl.replaceChildren(...items.map(row));
      renderPagination(pagerEl, { page, pageSize: PAGE_SIZE, total }, (p) => {
        page = p;
        load();
      });
    } catch (err) {
      rowsEl.innerHTML = '';
      renderEmpty(emptyEl, `加载失败：${err.message}`, '⚠️');
    }
  }

  function row(a) {
    const tr = document.createElement('tr');
    tr.innerHTML = `
      <td><a href="/#/article/${a.id}" target="_blank" style="font-weight:500">${escapeHtml(a.title)}</a></td>
      <td><span class="badge">${escapeHtml(a.category || '-')}</span></td>
      <td class="muted">${formatCount(a.view_count)}</td>
      <td class="muted">${formatDate(a.created_at)}</td>
      <td class="actions">
        <a class="btn btn--ghost btn--sm" href="#/comments?article_id=${a.id}">评论</a>
        <a class="btn btn--ghost btn--sm" href="#/posts/${a.id}/edit">编辑</a>
        <button class="btn btn--danger btn--sm" data-del="${a.id}" data-title="${escapeHtml(a.title)}">删除</button>
      </td>`;

    tr.querySelector('[data-del]').addEventListener('click', async (e) => {
      const ok = await confirmDialog(`确定删除文章《${e.currentTarget.dataset.title}》吗？此操作不可恢复。`);
      if (!ok) return;
      try {
        await api.del(`/api/v1/articles/${a.id}`);
        toast('文章已删除', 'success');
        // 若当前页删空则回退一页
        load();
      } catch (err) {
        toast(err.message, 'error');
      }
    });
    return tr;
  }

  let debounceTimer = 0;
  keywordInput.addEventListener('input', () => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => {
      keyword = keywordInput.value.trim();
      page = 1;
      load();
    }, 350);
  });

  load();
}
