/* ==========================================================================
   iGoBlog 后台 · comments.js — 评论管理
   后端评论接口按文章组织，故此处先选择文章，再查看/删除其评论。
   支持从文章管理页跳转携带 ?article_id= 直接定位到该文章的评论。
   文章选择器一次拉取 100 篇（分页加载更多），选择后展示评论卡片列表。
   ========================================================================== */

import { api } from '../api.js';
import { escapeHtml, formatDateTime } from '../format.js';
import { toast, confirmDialog, renderEmpty } from '../ui.js';

export function renderComments({ outlet, query = {} }) {
  outlet.innerHTML = `
    <div class="card panel" style="padding:22px 24px">
      <div class="row spread" style="margin-bottom:16px;flex-wrap:wrap;gap:10px">
        <div class="row" style="flex-wrap:wrap;gap:10px">
          <select class="select" data-role="article" style="min-width:260px">
            <option value="">选择文章…</option>
          </select>
          <button class="btn btn--ghost btn--sm" data-role="more">加载更多文章</button>
          <a class="btn btn--ghost btn--sm" href="#/posts" title="返回文章管理">← 文章管理</a>
        </div>
        <small>评论较多时可点击「加载更多文章」</small>
      </div>
      <p class="muted" data-role="cur" style="margin-bottom:14px" hidden></p>
      <div data-role="list"></div>
      <div data-role="empty"></div>
    </div>
  `;

  const select = outlet.querySelector('[data-role="article"]');
  const moreBtn = outlet.querySelector('[data-role="more"]');
  const curEl = outlet.querySelector('[data-role="cur"]');
  const listEl = outlet.querySelector('[data-role="list"]');
  const emptyEl = outlet.querySelector('[data-role="empty"]');

  let articlePage = 1;
  const PAGE = 100;

  async function loadArticles() {
    try {
      const { items, total } = await api.get(`/api/v1/articles?page=${articlePage}&page_size=${PAGE}`);
      const existing = new Set([...select.options].map((o) => o.value));
      items.forEach((a) => {
        if (existing.has(String(a.id))) return;
        const opt = document.createElement('option');
        opt.value = a.id;
        opt.textContent = `${a.title}（${a.view_count} 浏览）`;
        select.appendChild(opt);
      });
      if (articlePage * PAGE >= total) {
        moreBtn.disabled = true;
        moreBtn.textContent = '已全部加载';
      }
      articlePage++;
    } catch (err) {
      toast(err.message, 'error');
    }
  }

  async function loadComments(articleId) {
    // 展示当前文章
    const opt = select.selectedOptions[0];
    if (opt && opt.value) {
      curEl.textContent = `当前文章：${opt.textContent}`;
      curEl.hidden = false;
    }
    emptyEl.innerHTML = '';
    listEl.innerHTML = '<div class="skeleton" style="height:64px;margin-bottom:10px"></div><div class="skeleton" style="height:64px"></div>';
    try {
      let page = 1;
      let all = [];
      // 拉取该文章全部评论（每页 50，循环取完）
      for (;;) {
        const { items, total } = await api.get(`/api/v1/articles/${articleId}/comments?page=${page}&page_size=50`);
        all = all.concat(items ?? []);
        if (all.length >= total || (items ?? []).length === 0) break;
        page++;
      }

      if (all.length === 0) {
        listEl.innerHTML = '';
        renderEmpty(emptyEl, '该文章还没有评论', '💬');
        return;
      }

      listEl.replaceChildren(...all.map((c) => {
        const el = document.createElement('div');
        el.className = 'card comment-admin-card';
        el.innerHTML = `
          <div class="row">
            <span class="avatar" style="width:26px;height:26px;font-size:12px">${escapeHtml((c.username || '?')[0].toUpperCase())}</span>
            <b style="font-size:14px">${escapeHtml(c.username)}</b>
            <small>${formatDateTime(c.created_at)}</small>
            <button class="btn btn--danger btn--sm" style="margin-left:auto" data-del="${c.id}">删除</button>
          </div>
          <p class="comment__content">${escapeHtml(c.content)}</p>`;
        el.querySelector('[data-del]').addEventListener('click', async () => {
          const ok = await confirmDialog(`确定删除用户「${c.username}」的这条评论吗？`);
          if (!ok) return;
          try {
            await api.del(`/api/v1/comments/${c.id}`);
            toast('评论已删除', 'success');
            loadComments(articleId);
          } catch (err) {
            toast(err.message, 'error');
          }
        });
        return el;
      }));
    } catch (err) {
      listEl.innerHTML = '';
      renderEmpty(emptyEl, `加载失败：${err.message}`, '⚠️');
    }
  }

  function selectArticle(articleId) {
    if (!articleId) return;
    select.value = String(articleId);
    if (select.value) loadComments(select.value);
  }

  select.addEventListener('change', () => {
    const v = select.value;
    if (v) loadComments(v);
    else { curEl.hidden = true; listEl.innerHTML = ''; emptyEl.innerHTML = ''; }
  });
  moreBtn.addEventListener('click', async () => {
    await loadArticles();
    // 「加载更多」后补定位：目标文章可能刚被加载进来
    selectArticle(query.article_id);
  });

  // 初始化：加载文章列表，若携带 article_id 则直接定位（文章管理页跳转入口）
  (async () => {
    await loadArticles();
    selectArticle(query.article_id);
  })();
}
