/* ==========================================================================
   iGoBlog 后台 · settings.js — 站点设置（超管）
   站点名称/描述、注册开关（开关样式），保存后即时生效。
   ========================================================================== */

import { api } from '../api.js';
import { toast } from '../ui.js';

export function renderSettings({ outlet }) {
  outlet.innerHTML = `
    <div class="card panel" style="padding:26px 28px;max-width:560px">
      <div class="field">
        <label for="stTitle">站点名称（不超过 50 字符）</label>
        <input class="input" id="stTitle" maxlength="50" />
      </div>
      <div class="field">
        <label for="stDesc">站点描述（不超过 200 字符）</label>
        <textarea class="textarea" id="stDesc" style="min-height:80px" maxlength="200"></textarea>
      </div>
      <div class="field">
        <label>用户注册</label>
        <label class="switch">
          <input type="checkbox" id="stReg" />
          <span class="switch__track"><span class="switch__thumb"></span></span>
          <span class="switch__text" id="stRegText">开放注册</span>
        </label>
      </div>
      <p class="form-error" id="stError"></p>
      <button class="btn" id="stSave" style="margin-top:8px">保存设置</button>
    </div>
  `;

  const titleEl = document.getElementById('stTitle');
  const descEl = document.getElementById('stDesc');
  const regEl = document.getElementById('stReg');
  const regText = document.getElementById('stRegText');
  const errorEl = document.getElementById('stError');

  regEl.addEventListener('change', () => {
    regText.textContent = regEl.checked ? '开放注册' : '已关闭注册';
  });

  // 回填当前设置
  api.get('/api/v1/admin/settings').then((s) => {
    titleEl.value = s.site_title ?? '';
    descEl.value = s.site_description ?? '';
    regEl.checked = Boolean(s.registration_enabled);
    regText.textContent = regEl.checked ? '开放注册' : '已关闭注册';
  }).catch((err) => toast(err.message, 'error'));

  document.getElementById('stSave').addEventListener('click', async (e) => {
    errorEl.textContent = '';
    const btn = e.currentTarget;
    btn.disabled = true;
    try {
      await api.put('/api/v1/admin/settings', {
        site_title: titleEl.value.trim(),
        site_description: descEl.value.trim(),
        registration_enabled: regEl.checked,
      });
      toast('设置已保存，即时生效', 'success');
    } catch (err) {
      errorEl.textContent = err.message;
    } finally {
      btn.disabled = false;
    }
  });
}
