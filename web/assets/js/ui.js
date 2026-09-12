/* ==========================================================================
   iGoBlog · ui.js — 通用 UI 组件：Toast / 确认弹窗 / 分页器 / 骨架屏 / 空状态
   ========================================================================== */

/* ---------- Toast ---------- */
let toastWrap = null;

/**
 * 显示一条轻提示。
 * @param {string} message 文本
 * @param {'info'|'success'|'error'|'warning'} type 类型
 * @param {number} duration 展示时长（ms）
 */
export function toast(message, type = 'info', duration = 2600) {
  if (!toastWrap) {
    toastWrap = document.createElement('div');
    toastWrap.className = 'toast-wrap';
    document.body.appendChild(toastWrap);
  }
  const el = document.createElement('div');
  el.className = `toast${type === 'info' ? '' : ` toast--${type}`}`;
  el.textContent = message;
  toastWrap.appendChild(el);
  setTimeout(() => {
    el.classList.add('hide');
    setTimeout(() => el.remove(), 300);
  }, duration);
}

/* ---------- 确认弹窗 ---------- */
/**
 * 自定义确认弹窗，替代 window.confirm。
 * @returns {Promise<boolean>} 用户是否确认
 */
export function confirmDialog(message, { danger = true, okText = '删除', title = '确认操作' } = {}) {
  return new Promise((resolve) => {
    const backdrop = document.createElement('div');
    backdrop.className = 'modal-backdrop';
    backdrop.innerHTML = `
      <div class="modal" role="dialog" aria-modal="true">
        <h3 class="modal__title">${title}</h3>
        <p style="color:var(--text-2);margin-bottom:22px;text-align:center">${message}</p>
        <div class="row" style="gap:10px">
          <button class="btn btn--ghost btn--block" data-act="cancel">取消</button>
          <button class="btn ${danger ? 'btn--danger' : ''} btn--block" data-act="ok">${okText}</button>
        </div>
      </div>`;
    const done = (val) => {
      backdrop.remove();
      resolve(val);
    };
    backdrop.addEventListener('click', (e) => {
      if (e.target === backdrop) done(false);
      const act = e.target.dataset?.act;
      if (act === 'ok') done(true);
      if (act === 'cancel') done(false);
    });
    document.body.appendChild(backdrop);
  });
}

/* ---------- 分页器 ---------- */
/**
 * 渲染分页器。
 * @param {HTMLElement} container 容器
 * @param {{page:number, pageSize:number, total:number}} state
 * @param {(page:number)=>void} onPage 翻页回调
 */
export function renderPagination(container, { page, pageSize, total }, onPage) {
  const pages = Math.max(1, Math.ceil(total / pageSize));
  if (pages <= 1) {
    container.replaceChildren();
    return;
  }

  // 计算窗口内的页码：最多显示 7 个
  const windowSize = 7;
  let start = Math.max(1, page - 3);
  const end = Math.min(pages, start + windowSize - 1);
  start = Math.max(1, end - windowSize + 1);

  const nav = document.createElement('nav');
  nav.className = 'pagination';

  const btn = (label, target, opts = {}) => {
    const b = document.createElement('button');
    b.textContent = label;
    if (opts.active) b.classList.add('active');
    if (opts.disabled) b.disabled = true;
    else b.addEventListener('click', () => onPage(target));
    return b;
  };

  nav.appendChild(btn('‹', page - 1, { disabled: page <= 1 }));
  if (start > 1) {
    nav.appendChild(btn('1', 1));
    if (start > 2) nav.appendChild(btn('…', 0, { disabled: true }));
  }
  for (let p = start; p <= end; p++) nav.appendChild(btn(String(p), p, { active: p === page }));
  if (end < pages) {
    if (end < pages - 1) nav.appendChild(btn('…', 0, { disabled: true }));
    nav.appendChild(btn(String(pages), pages));
  }
  nav.appendChild(btn('›', page + 1, { disabled: page >= pages }));

  container.replaceChildren(nav);
}

/* ---------- 骨架屏 ---------- */
/** 渲染 n 张卡片骨架（用于首页加载中）。 */
export function renderCardSkeletons(container, n = 6) {
  container.replaceChildren(
    ...Array.from({ length: n }, () => {
      const card = document.createElement('div');
      card.className = 'card article-card';
      card.innerHTML = `
        <div class="article-card__body">
          <div class="skeleton" style="height:22px;width:70%;margin-bottom:14px"></div>
          <div class="skeleton" style="height:14px;width:95%;margin-bottom:8px"></div>
          <div class="skeleton" style="height:14px;width:60%;margin-bottom:16px"></div>
          <div class="skeleton" style="height:12px;width:45%"></div>
        </div>`;
      return card;
    })
  );
}

/* ---------- 空状态 ---------- */
/** 渲染空状态提示。 */
export function renderEmpty(container, message = '暂无内容', icon = '📄') {
  const el = document.createElement('div');
  el.className = 'empty';
  el.innerHTML = `
    <div class="empty__icon">${icon}</div>
    <p>${message}</p>`;
  container.replaceChildren(el);
}

/* ---------- 状态徽标 ---------- */
/** 渲染"加载中"按钮状态。 */
export function withLoading(btn, promiseFactory) {
  if (btn.dataset.loading === '1') return;
  btn.dataset.loading = '1';
  btn.disabled = true;
  const original = btn.innerHTML;
  btn.innerHTML = '<span class="spin"></span>';
  promiseFactory().finally(() => {
    btn.dataset.loading = '';
    btn.disabled = false;
    btn.innerHTML = original;
  });
}
