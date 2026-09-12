/* ==========================================================================
   iGoBlog 后台 · editor.js — 文章新建 / 编辑
   表单：标题、分类下拉、标签复选（胶囊样式）、正文多行输入；
   编辑模式先回填；提交校验与行内错误提示。
   ========================================================================== */

import { api } from '../api.js';
import { escapeHtml } from '../format.js';
import { toast } from '../ui.js';

export function renderEditor({ params, outlet }) {
  const editingId = params.id ? Number(params.id) : null;

  outlet.innerHTML = `
    <div class="card editor-card">
      <div class="field field-title">
        <label for="edTitle">标题（不超过 200 字符）</label>
        <input class="input" id="edTitle" maxlength="200" placeholder="给文章起个标题…" />
      </div>

      <div class="editor-meta">
        <div class="field">
          <label for="edCategory">分类</label>
          <select class="select" id="edCategory"><option value="">加载中…</option></select>
        </div>
        <div class="field">
          <label>标签（可多选，最多 20 个）</label>
          <div class="tag-checks" id="edTags"><span class="muted">加载中…</span></div>
        </div>
      </div>

      <div class="field">
        <label for="edBody">正文</label>
        <textarea class="textarea body-input" id="edBody" placeholder="开始写作…（保存后以纯文本排版展示）"></textarea>
      </div>

      <p class="form-error" id="edError"></p>
      <div class="editor-actions">
        <button class="btn" id="edSave" data-editing="${editingId ?? ''}">
          ${editingId ? '保存修改' : '发布文章'}
        </button>
        <a class="btn btn--ghost" href="#/posts">返回列表</a>
      </div>
    </div>
  `;

  const titleEl = document.getElementById('edTitle');
  const categoryEl = document.getElementById('edCategory');
  const tagsEl = document.getElementById('edTags');
  const bodyEl = document.getElementById('edBody');
  const errorEl = document.getElementById('edError');

  /* ---------- 加载分类与标签 ---------- */
  Promise.all([
    api.get('/api/v1/categories'),
    api.get('/api/v1/tags'),
  ]).then(([categories, tags]) => {
    categoryEl.innerHTML = categories.length
      ? categories.map((c) => `<option value="${c.id}">${escapeHtml(c.name)}</option>`).join('')
      : '<option value="">（暂无分类，请先到分类管理创建）</option>';

    tagsEl.innerHTML = tags.length
      ? tags.map((t) => `
          <label>
            <input type="checkbox" value="${t.id}" /><span># ${escapeHtml(t.name)}</span>
          </label>`).join('')
      : '<span class="muted">（暂无标签）</span>';
  }).catch((err) => toast(err.message, 'error'));

  /* ---------- 编辑模式：回填 ---------- */
  if (editingId) {
    api.get(`/api/v1/articles/${editingId}`).then((a) => {
      titleEl.value = a.title;
      bodyEl.value = a.content;
      categoryEl.value = a.category_id;
      a.tags?.forEach((t) => {
        const box = tagsEl.querySelector(`input[value="${t.id}"]`);
        if (box) box.checked = true;
      });
    }).catch((err) => {
      toast(err.message, 'error');
      location.hash = '#/posts';
    });
  }

  /* ---------- 提交 ---------- */
  document.getElementById('edSave').addEventListener('click', async (e) => {
    errorEl.textContent = '';
    const title = titleEl.value.trim();
    const content = bodyEl.value.trim();
    const categoryId = Number(categoryEl.value);
    const tagIds = [...tagsEl.querySelectorAll('input:checked')].map((i) => Number(i.value));

    if (!title) return void (errorEl.textContent = '标题不能为空');
    if (!content) return void (errorEl.textContent = '正文不能为空');
    if (!categoryId) return void (errorEl.textContent = '请选择分类（可先到分类管理创建）');

    const btn = e.currentTarget;
    btn.disabled = true;
    try {
      if (editingId) {
        await api.put(`/api/v1/articles/${editingId}`, {
          title, content, category_id: categoryId, tag_ids: tagIds,
        });
        toast('文章已保存', 'success');
      } else {
        await api.post('/api/v1/articles', {
          title, content, category_id: categoryId, tag_ids: tagIds,
        });
        toast('文章已发布', 'success');
      }
      location.hash = '#/posts';
    } catch (err) {
      errorEl.textContent = err.message;
      btn.disabled = false;
    }
  });
}
