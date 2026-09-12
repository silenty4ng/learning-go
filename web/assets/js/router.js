/* ==========================================================================
   iGoBlog · router.js — 极简 hash 路由器
   路由形如 '#/article/:id'，:xxx 为参数；query 形如 '#/?page=2&keyword=go'。
   ========================================================================== */

/** 解析当前 hash：返回 { path, params(query) }。 */
export function parseHash() {
  const raw = location.hash.slice(1) || '/';
  const [path, query = ''] = raw.split('?');
  const params = Object.fromEntries(new URLSearchParams(query));
  return { path: path || '/', params };
}

/**
 * 创建路由器。
 * @param {HTMLElement} outlet 视图渲染容器
 * @param {Array<{path:string, view:(ctx)=>any}>} routes
 *   view 接收 ctx = { params(路径参数), query(查询参数) }，
 *   可返回清理函数（视图销毁时调用，如 clearInterval）。
 */
export function createRouter(outlet, routes) {
  let cleanup = null;

  function match() {
    const { path, params: query } = parseHash();
    for (const route of routes) {
      const keys = [];
      // 将 '/article/:id' 转为正则
      const pattern = route.path
        .split('/')
        .map((seg) => {
          if (seg.startsWith(':')) {
            keys.push(seg.slice(1));
            return '([^/]+)';
          }
          return seg.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        })
        .join('/');
      const m = new RegExp(`^${pattern}/?$`).exec(path);
      if (m) {
        const routeParams = {};
        keys.forEach((k, i) => (routeParams[k] = decodeURIComponent(m[i + 1])));
        return { route, params: routeParams, query };
      }
    }
    return null;
  }

  function render() {
    if (typeof cleanup === 'function') cleanup();
    cleanup = null;

    const matched = match();
    if (!matched) {
      location.hash = '#/';
      return;
    }
    outlet.replaceChildren();
    cleanup = matched.route.view({
      params: matched.params,
      query: matched.query,
      outlet,
    }) ?? null;
  }

  window.addEventListener('hashchange', render);
  return { start: render };
}

/** 编写带 query 的 hash 地址。 */
export function buildHash(path, query = {}) {
  const qs = new URLSearchParams(
    Object.fromEntries(Object.entries(query).filter(([, v]) => v !== '' && v != null))
  ).toString();
  return `#${path}${qs ? `?${qs}` : ''}`;
}
