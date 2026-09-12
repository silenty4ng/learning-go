/* ==========================================================================
   iGoBlog 前台 · home.js — 首页（文章列表）
   分页 + 分类/标签筛选 + 防抖关键词搜索；筛选状态写入 hash query（可分享、可回退）。
   ========================================================================== */

import { api } from '../api.js';
import { buildHash, parseHash } from '../router.js';
import { escapeHtml, formatDate, formatCount } from '../format.js';
import {
  toast, renderPagination, renderCardSkeletons, renderEmpty,
} from '../ui.js';

const PAGE_SIZE = 10;

/**
 * 渲染首页。ctx 由路由器注入：query 为 hash 查询参数。
 * @returns {() => void} 清理函数（路由切换时调用）
 */
export function renderHome({ query, outlet }) {
  outlet.innerHTML = `
    <div class="page-head">
      <h1>全部文章</h1>
      <span class="result-count" data-role="count"></span>
    </div>

    <div class="filter-bar">
      <select class="select" data-role="category" aria-label="按分类筛选">
        <option value="">全部分类</option>
      </select>
      <select class="select" data-role="tag" aria-label="按标签筛选">
        <option value="">全部标签</option>
      </select>
      <div class="search">
        <input class="input" data-role="keyword" placeholder="搜索标题或正文…" value="${escapeHtml(query.keyword || '')}" />
      </div>
    </div>

    <div class="article-grid" data-role="list"></div>
    <div data-role="pager"></div>
  `;

  const state = {
    page: Math.max(1, parseInt(query.page, 10) || 1),
    categoryId: query.category_id || '',
    tagId: query.tag_id || '',
    keyword: query.keyword || '',
  };

  const listEl = outlet.querySelector('[data-role="list"]');
  const pagerEl = outlet.querySelector('[data-role="pager"]');
  const countEl = outlet.querySelector('[data-role="count"]');
  const categorySel = outlet.querySelector('[data-role="category"]');
  const tagSel = outlet.querySelector('[data-role="tag"]');
  const keywordInput = outlet.querySelector('[data-role="keyword"]');

  /* ---------- 数据加载 ---------- */
  async function loadOptions() {
    try {
      const [categories, tags] = await Promise.all([
        api.get('/api/v1/categories'),
        api.get('/api/v1/tags'),
      ]);
      categories.forEach((c) => {
        const opt = document.createElement('option');
        opt.value = c.id;
        opt.textContent = c.name;
        categorySel.appendChild(opt);
      });
      tags.forEach((t) => {
        const opt = document.createElement('option');
        opt.value = t.id;
        opt.textContent = `# ${t.name}`;
        tagSel.appendChild(opt);
      });
      categorySel.value = state.categoryId;
      tagSel.value = state.tagId;
    } catch (err) {
      toast(err.message, 'error');
    }
  }

  async function load() {
    renderCardSkeletons(listEl, 4);
    countEl.textContent = '';
    try {
      const qs = new URLSearchParams({
        page: state.page,
        page_size: PAGE_SIZE,
        ...(state.categoryId && { category_id: state.categoryId }),
        ...(state.tagId && { tag_id: state.tagId }),
        ...(state.keyword && { keyword: state.keyword }),
      });
      const { items, total } = await api.get(`/api/v1/articles?${qs}`);
      countEl.textContent = `共 ${total} 篇`;

      if (!items || items.length === 0) {
        renderEmpty(listEl, '没有找到匹配的文章，换个关键词试试？', '🔍');
      } else {
        listEl.replaceChildren(...items.map(articleCard));
      }
      renderPagination(pagerEl, { page: state.page, pageSize: PAGE_SIZE, total }, (p) => {
        state.page = p;
        syncHash();
        window.scrollTo({ top: 0, behavior: 'smooth' });
      });
    } catch (err) {
      renderEmpty(listEl, `加载失败：${err.message}`, '⚠️');
    }
  }

  /** 将筛选状态同步到 hash（替换历史记录，避免每次敲字都产生历史）。 */
  function syncHash() {
    const target = buildHash('/', {
      page: state.page > 1 ? state.page : '',
      category_id: state.categoryId,
      tag_id: state.tagId,
      keyword: state.keyword,
    });
    if (location.hash !== target) {
      history.replaceState(null, '', target);
    }
  }

  /* ---------- 文章卡片 ---------- */
  function articleCard(a) {
    const card = document.createElement('article');
    card.className = 'card article-card';
    card.setAttribute('tabindex', '0');
    card.setAttribute('role', 'link');
    card.innerHTML = `
      <div class="article-card__body">
        <h2 class="article-card__title">${escapeHtml(a.title)}</h2>
        <p class="article-card__excerpt">${escapeHtml(a.content.slice(0, 120))}</p>
        <div class="article-card__tags">
          <span class="badge">${escapeHtml(a.category || '未分类')}</span>
          ${(a.tags || []).map((t) => `<span class="badge badge--muted"># ${escapeHtml(t.name)}</span>`).join('')}
        </div>
        <div class="article-card__meta">
          <span>✍️ ${escapeHtml(a.author || '')}</span>
          <span class="dot">${formatDate(a.created_at)}</span>
          <span class="dot">👁 ${formatCount(a.view_count)}</span>
        </div>
      </div>`;
    const go = () => (location.hash = `#/article/${a.id}`);
    card.addEventListener('click', go);
    card.addEventListener('keydown', (e) => e.key === 'Enter' && go());
    return card;
  }

  /* ---------- 交互 ---------- */
  categorySel.addEventListener('change', () => {
    state.categoryId = categorySel.value;
    state.page = 1;
    syncHash();
    load();
  });
  tagSel.addEventListener('change', () => {
    state.tagId = tagSel.value;
    state.page = 1;
    syncHash();
    load();
  });

  let debounceTimer = 0;
  keywordInput.addEventListener('input', () => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => {
      state.keyword = keywordInput.value.trim();
      state.page = 1;
      syncHash();
      load();
    }, 350);
  });

  loadOptions();
  load();

  // 清理函数：路由切走后，悬挂的防抖不再触发请求
  return () => clearTimeout(debounceTimer);
}
