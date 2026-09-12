/* ==========================================================================
   iGoBlog 前台 · article.js — 文章详情 + 评论区
   正文纯文本渲染（white-space: pre-wrap）；评论分页、发表（需登录）、
   评论者本人或文章作者可删除。
   ========================================================================== */

import { api } from '../api.js';
import { isLoggedIn, getSavedUser } from '../auth.js';
import { escapeHtml, formatDate, formatDateTime, formatCount } from '../format.js';
import { toast, confirmDialog, renderPagination, renderEmpty } from '../ui.js';

const COMMENT_PAGE_SIZE = 10;

export function renderArticle({ params, outlet }) {
  const id = Number(params.id);

  outlet.innerHTML = `
    <div class="article-full">
      <p class="breadcrumb"><a href="#/">← 返回文章列表</a></p>
      <article class="card" data-role="article"></article>
    </div>
    <section class="comments" data-role="comments"></section>
  `;

  const articleEl = outlet.querySelector('[data-role="article"]');
  const commentsEl = outlet.querySelector('[data-role="comments"]');
  let authorId = 0; // 文章作者 ID，用于评论删除权限判断

  /* ---------- 文章详情 ---------- */
  async function loadArticle() {
    articleEl.innerHTML = `
      <div class="skeleton" style="height:30px;width:65%;margin-bottom:16px"></div>
      <div class="skeleton" style="height:14px;width:40%;margin-bottom:20px"></div>
      <div class="skeleton" style="height:300px"></div>`;
    try {
      const a = await api.get(`/api/v1/articles/${id}`);
      authorId = a.author_id;
      document.title = `${a.title} · iGoBlog`;
      articleEl.innerHTML = `
        <h1>${escapeHtml(a.title)}</h1>
        <div class="article-meta">
          <span>✍️ ${escapeHtml(a.author)}</span>
          <span class="dot">${formatDate(a.created_at)}</span>
          <span class="dot">👁 ${formatCount(a.view_count)} 次浏览</span>
          <span class="dot">📂 ${escapeHtml(a.category)}</span>
        </div>
        <div class="article-content">${escapeHtml(a.content)}</div>
        ${a.tags?.length ? `
          <div class="article-tags">
            ${a.tags.map((t) => `<a class="badge badge--muted" href="#/?tag_id=${t.id}"># ${escapeHtml(t.name)}</a>`).join('')}
          </div>` : ''}
      `;
    } catch (err) {
      renderEmpty(articleEl,
        err.status === 404 ? '文章不存在或已被删除' : `加载失败：${err.message}`, '📄');
    }
  }

  /* ---------- 评论区 ---------- */
  let commentPage = 1;

  async function loadComments(page = 1) {
    commentPage = page;
    commentsEl.innerHTML = `
      <div class="skeleton" style="height:70px;margin-bottom:12px"></div>
      <div class="skeleton" style="height:70px"></div>`;
    try {
      const { items, total } = await api.get(
        `/api/v1/articles/${id}/comments?page=${commentPage}&page_size=${COMMENT_PAGE_SIZE}`
      );
      commentsEl.innerHTML = `
        <div class="comments__head">
          <h2>评论 <span class="muted">(${total})</span></h2>
        </div>
        <div data-role="comment-form"></div>
        <div data-role="comment-list"></div>
        <div data-role="comment-pager"></div>
      `;
      renderCommentForm(commentsEl.querySelector('[data-role="comment-form"]'));
      renderCommentList(commentsEl.querySelector('[data-role="comment-list"]'), items);
      renderPagination(
        commentsEl.querySelector('[data-role="comment-pager"]'),
        { page: commentPage, pageSize: COMMENT_PAGE_SIZE, total },
        (p) => {
          loadComments(p);
          commentsEl.scrollIntoView({ behavior: 'smooth' });
        }
      );
    } catch (err) {
      commentsEl.innerHTML = '';
      if (err.status !== 404) toast(err.message, 'error');
    }
  }

  function renderCommentForm(container) {
    if (!isLoggedIn()) {
      container.innerHTML = `
        <div class="card comment-form" style="text-align:center;color:var(--text-3)">
          登录后即可发表评论，来抢沙发吧 —
          <a href="#" data-role="goto-login">立即登录 / 注册</a>
        </div>`;
      container.querySelector('[data-role="goto-login"]').addEventListener('click', (e) => {
        e.preventDefault();
        window.dispatchEvent(new CustomEvent('front:open-auth'));
      });
      return;
    }

    container.innerHTML = `
      <div class="card comment-form">
        <textarea class="textarea" data-role="content" maxlength="1000"
          placeholder="写下你的评论…（最多 1000 字）"></textarea>
        <div class="row spread" style="margin-top:12px">
          <small><span data-role="count">0</span>/1000</small>
          <button class="btn btn--sm" data-role="submit">发表评论</button>
        </div>
      </div>`;

    const ta = container.querySelector('[data-role="content"]');
    const counter = container.querySelector('[data-role="count"]');
    ta.addEventListener('input', () => (counter.textContent = ta.value.length));

    container.querySelector('[data-role="submit"]').addEventListener('click', async (e) => {
      const content = ta.value.trim();
      if (!content) {
        toast('评论内容不能为空', 'warning');
        return;
      }
      const btn = e.currentTarget;
      btn.disabled = true;
      try {
        await api.post(`/api/v1/articles/${id}/comments`, { content });
        toast('评论发表成功', 'success');
        await loadComments(1);
      } catch (err) {
        toast(err.message, 'error');
        btn.disabled = false;
      }
    });
  }

  function renderCommentList(container, items) {
    if (!items || items.length === 0) {
      container.innerHTML = '<p class="muted" style="text-align:center;padding:18px 0">还没有评论</p>';
      return;
    }
    const me = getSavedUser();
    container.replaceChildren(...items.map((c) => {
      // 评论者本人或文章作者可删除
      const canDelete = me && (me.id === c.user_id || me.id === authorId);
      const el = document.createElement('div');
      el.className = 'card comment';
      el.innerHTML = `
        <div class="comment__head">
          <span class="avatar" style="width:26px;height:26px;font-size:12px">${escapeHtml((c.username || '?')[0].toUpperCase())}</span>
          <span class="comment__author">${escapeHtml(c.username)}</span>
          <span class="comment__time">${formatDateTime(c.created_at)}</span>
          ${canDelete ? `<button class="comment__del" data-id="${c.id}">删除</button>` : ''}
        </div>
        <p class="comment__content">${escapeHtml(c.content)}</p>`;

      const delBtn = el.querySelector('.comment__del');
      if (delBtn) {
        delBtn.addEventListener('click', async () => {
          const ok = await confirmDialog('确定删除这条评论吗？');
          if (!ok) return;
          try {
            await api.del(`/api/v1/comments/${c.id}`);
            toast('评论已删除', 'success');
            loadComments(commentPage);
          } catch (err) {
            toast(err.message, 'error');
          }
        });
      }
      return el;
    }));
  }

  /* ---------- 启动 ---------- */
  loadArticle();
  loadComments();
}
