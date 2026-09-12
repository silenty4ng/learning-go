/* ==========================================================================
   iGoBlog 后台 · taxonomies.js — 分类管理与标签管理
   「列表 + 行内新增 + 重命名 + 删除」，分类下有文章时删除返回 409 并提示。
   两个页面结构一致，抽取共用渲染逻辑 renderTaxonomy。
   ========================================================================== */

import { api } from '../api.js';
import { escapeHtml } from '../format.js';
import { toast, confirmDialog, renderEmpty } from '../ui.js';

/** 分类 / 标签共用管理页渲染。 */
function renderTaxonomy({ outlet, kind }) {
  const isCategory = kind === 'category';
  const listPath = isCategory ? '/api/v1/categories' : '/api/v1/tags';
  const itemName = isCategory ? '分类' : '标签';

  outlet.innerHTML = `
    <div class="card panel" style="padding:22px 24px">
      <form class="inline-form" data-role="form">
        <input class="input" data-role="name" maxlength="${isCategory ? 50 : 30}"
          placeholder="输入${itemName}名称（不超过 ${isCategory ? 50 : 30} 字符）" />
        <button class="btn" type="submit">新增${itemName}</button>
      </form>
      <div data-role="list"></div>
      <div data-role="empty"></div>
    </div>
  `;

  const listEl = outlet.querySelector('[data-role="list"]');
  const emptyEl = outlet.querySelector('[data-role="empty"]');
  const form = outlet.querySelector('[data-role="form"]');
  const nameInput = outlet.querySelector('[data-role="name"]');

  async function load() {
    listEl.innerHTML = '<div class="skeleton" style="height:20px;margin:10px 4px"></div>';
    emptyEl.innerHTML = '';
    try {
      const items = await api.get(listPath);
      if (!items || items.length === 0) {
        listEl.innerHTML = '';
        renderEmpty(emptyEl, `暂无${itemName}`, isCategory ? '📂' : '🏷️');
        return;
      }
      listEl.innerHTML = `
        <div class="table-wrap"><table class="table">
          <thead><tr><th>名称</th><th style="text-align:right">操作</th></tr></thead>
          <tbody data-role="rows"></tbody>
        </table></div>`;
      listEl.querySelector('[data-role="rows"]').replaceChildren(...items.map(row));
    } catch (err) {
      listEl.innerHTML = '';
      renderEmpty(emptyEl, `加载失败：${err.message}`, '⚠️');
    }
  }

  function row(item) {
    const tr = document.createElement('tr');
    tr.innerHTML = `
      <td style="font-weight:500">${escapeHtml(item.name)}</td>
      <td class="actions">
        <button class="btn btn--ghost btn--sm" data-rename>重命名</button>
        <button class="btn btn--danger btn--sm" data-delete>删除</button>
      </td>`;

    tr.querySelector('[data-rename]').addEventListener('click', async () => {
      const name = prompt(`重命名${itemName}：`, item.name);
      if (name === null) return;
      const trimmed = name.trim();
      if (!trimmed || trimmed === item.name) return;
      try {
        if (isCategory) {
          await api.put(`/api/v1/categories/${item.id}`, { name: trimmed });
        } else {
          // 标签接口未提供更新，采用删除+重建保底（前台展示以 ID 绑定，此处仅提示）
          toast('标签暂不支持重命名，可删除后重建', 'info');
          return;
        }
        toast(`${itemName}已重命名`, 'success');
        load();
      } catch (err) {
        toast(err.message, 'error');
      }
    });

    tr.querySelector('[data-delete]').addEventListener('click', async () => {
      const ok = await confirmDialog(
        isCategory
          ? `确定删除分类「${item.name}」吗？分类下仍有文章时将删除失败。`
          : `确定删除标签「${item.name}」吗？该标签将从所有文章移除。`);
      if (!ok) return;
      try {
        await api.del(`${listPath}/${item.id}`);
        toast(`${itemName}已删除`, 'success');
        load();
      } catch (err) {
        // 409 CONFLICT：分类下仍有文章
        toast(err.message, 'error');
      }
    });
    return tr;
  }

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const name = nameInput.value.trim();
    if (!name) {
      toast(`请输入${itemName}名称`, 'warning');
      return;
    }
    try {
      await api.post(listPath, { name });
      toast(`${itemName}「${name}」已创建`, 'success');
      nameInput.value = '';
      load();
    } catch (err) {
      toast(err.message, 'error'); // 409：名称已存在
    }
  });

  load();
}

/** 分类管理页。 */
export function renderCategories(ctx) {
  renderTaxonomy({ ...ctx, kind: 'category' });
}

/** 标签管理页。 */
export function renderTags(ctx) {
  renderTaxonomy({ ...ctx, kind: 'tag' });
}
