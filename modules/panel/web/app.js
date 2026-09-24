function getResolutionLabel(res) {
  if (!res) return '';
  const m = String(res).match(/^(\d{3,5})[xX*](\d{3,5})$/);
  if (!m) return res;
  const w = parseInt(m[1], 10), h = parseInt(m[2], 10);
  if (h >= 2160 || w >= 3840) return '4K';
  if (h >= 1080 || w >= 1920) return '1080P';
  if (h >= 720 || w >= 1280) return '720P';
  if (h >= 576) return '576P';
  if (h >= 480) return '480P';
  return res;
}

function getResolutionClass(res) {
  const lbl = getResolutionLabel(res);
  if (lbl === '4K') return 'badge-res-4k';
  if (lbl === '1080P') return 'badge-res-1080';
  if (lbl === '720P') return 'badge-res-720';
  return 'badge-res-sd';
}

async function probeChannel(c) {
  const ref = c.key || c.id;
  try {
    toast(`正在探测「${c.name}」的视频流分辨率…`);
    const res = await api('channels/' + encodeURIComponent(ref) + '/probe', 'POST');
    if (res && res.resolution) {
      c.resolution = res.resolution;
      renderChannels();
      toast(`探测成功：「${c.name}」分辨率为 ${res.resolution} (${getResolutionLabel(res.resolution)})`);
    } else {
      toast(`未能探测到「${c.name}」的分辨率`, true);
    }
  } catch (err) {
    toast(`探测「${c.name}」失败: ${err.message}`, true);
  }
}

'use strict';

const $ = (id) => document.getElementById(id);
let settings, channels = [], page = 0, editing = null, toastTimer, lastScanState = 'idle', lastRefreshState = 'idle';
const pageSize = 30;
let epgRequestVersion = 0, lastEPGState = '';
let logoMatchVersion = 0, pendingLogoMatch = null;
let currentEPGProgrammes = [], epgCurrentFilter = 'all', epgLiveTimer = null;

function toast(message, error = false) {
  const el = $('toast');
  el.textContent = message;
  el.classList.toggle('error', error);
  el.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { el.hidden = true; }, error ? 10000 : 4000);
}

async function copyText(text, successMsg = '已复制到剪贴板') {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
    } else {
      const input = document.createElement('textarea');
      input.value = text;
      input.style.position = 'fixed';
      input.style.opacity = '0';
      document.body.appendChild(input);
      input.focus();
      input.select();
      document.execCommand('copy');
      input.remove();
    }
    toast(successMsg);
  } catch (e) {
    toast('复制失败，请手动复制：' + text, true);
  }
}

async function api(path, method = 'GET', body) {
  const token = sessionStorage.getItem('iptv-token') || '';
  const isFormData = typeof FormData !== 'undefined' && body instanceof FormData;
  const headers = {
    ...(token ? { Authorization: 'Bearer ' + token } : {})
  };
  if (!isFormData) {
    headers['Content-Type'] = 'application/json';
  }
  const response = await fetch('/api/panel/' + path, {
    method,
    headers,
    ...(body !== undefined ? { body: isFormData ? body : JSON.stringify(body) } : {})
  });
  if (response.status === 401 || response.status === 403) {
    $('login').hidden = false;
    $('workspace').hidden = true;
  }
  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw Error(error.error || '请求失败：' + response.status);
  }
  return response.headers.get('Content-Type')?.startsWith('image/') ? response.blob() : response.json();
}

async function busy(button, action) {
  button.disabled = true;
  try {
    await action();
  } catch (e) {
    toast(e.message, true);
  } finally {
    button.disabled = false;
  }
}

function openMobileSidebar() {
  const aside = document.querySelector('aside');
  const backdrop = $('sidebar-backdrop');
  if (aside) aside.classList.add('drawer-open');
  if (backdrop) backdrop.classList.add('active');
  document.body.classList.add('drawer-opened');
}

function closeMobileSidebar() {
  const aside = document.querySelector('aside');
  const backdrop = $('sidebar-backdrop');
  if (aside) aside.classList.remove('drawer-open');
  if (backdrop) backdrop.classList.remove('active');
  document.body.classList.remove('drawer-opened');
}

if ($('mobile-menu-btn')) $('mobile-menu-btn').addEventListener('click', openMobileSidebar);
if ($('sidebar-close-btn')) $('sidebar-close-btn').addEventListener('click', closeMobileSidebar);
if ($('sidebar-backdrop')) $('sidebar-backdrop').addEventListener('click', closeMobileSidebar);
if ($('mobile-reload-btn') && $('reload')) {
  $('mobile-reload-btn').addEventListener('click', () => $('reload').click());
}

function tab(name) {
  closeMobileSidebar();
  document.querySelectorAll('.tab').forEach(el => el.hidden = el.id !== 'tab-' + name);
  document.querySelectorAll('[data-tab]').forEach(el => el.classList.toggle('active', el.dataset.tab === name));
  $('page-title').textContent = {
    channels: '频道管理',
    account: 'IPTV配置',
    scan: '组播扫描',
    ai: '智能识别',
    epg: 'EPG 节目单'
  }[name];
  if (name === 'channels') {
    const activeSubBtn = document.querySelector('.channel-subnav-btn.active');
    const view = activeSubBtn ? activeSubBtn.dataset.channelView : 'list';
    switchChannelView(view);
  }
}

function switchChannelView(view) {
  document.querySelectorAll('.channel-subnav-btn').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.channelView === view);
  });
  const listEl = $('channel-view-list');
  const sortingEl = $('channel-view-sorting');
  const mappingEl = $('channel-view-mapping');
  if (listEl) listEl.hidden = (view !== 'list');
  if (sortingEl) sortingEl.hidden = (view !== 'sorting');
  if (mappingEl) mappingEl.hidden = (view !== 'mapping');

  if (view === 'sorting') {
    if (typeof renderGroupOrderList === 'function') renderGroupOrderList();
    if (typeof updateGroupChannelSelect === 'function') updateGroupChannelSelect();
  } else if (view === 'mapping') {
    if (typeof loadMappings === 'function') loadMappings();
  }
}

document.querySelectorAll('[data-tab]').forEach(el => el.addEventListener('click', () => tab(el.dataset.tab)));
document.querySelectorAll('[data-channel-view]').forEach(btn => {
  btn.addEventListener('click', () => switchChannelView(btn.dataset.channelView));
});
document.querySelectorAll('[data-close]').forEach(el => el.addEventListener('click', () => $(el.dataset.close).close()));

const CATCHUP_PRESETS = [
  "&utc=${start}&lutc=${end}",
  "&start={utc}&end={utcend}",
  "&utc={utc}&lutc={utcend}",
  "&start=${timestamp}&duration=${duration}",
];

function populateCatchupTemplateSelect(currentTpl) {
  const select = $("catchup-template-select");
  const input = $("forward-catchup-template-input");
  if (!select) return;
  const val = (currentTpl || "").trim() || "&utc=${start}&lutc=${end}";

  if (CATCHUP_PRESETS.includes(val)) {
    select.value = val;
    if (input) { input.value = val; input.hidden = true; }
    select.classList.remove("is-custom");
  } else {
    select.value = "__custom__";
    if (input) { input.value = val; input.hidden = false; }
    select.classList.add("is-custom");
  }
}

function getActiveCatchupTemplate() {
  const select = $("catchup-template-select");
  const input = $("forward-catchup-template-input");
  if (select && select.value === "__custom__") {
    return input ? input.value.trim() : "";
  }
  return select ? select.value : (input ? input.value.trim() : "");
}

function populateFCCSelect(fccList, currentFCC) {
  const select = $('fcc-select');
  const input = $('forward-fcc-input');
  if (!select) return;
  select.replaceChildren();
  select.append(new Option('不使用快速换台', ''));

  const presets = [
    '124.75.26.151:15970',
    '124.75.26.152:15970',
    '124.75.26.153:15970',
    '124.75.29.151:15970',
    '124.75.29.152:15970',
    '124.75.34.151:15970',
  ];

  const allFCCs = Array.from(new Set([...presets, ...(fccList || [])])).filter(Boolean);
  for (const item of allFCCs) {
    let label = item;
    if (item === '124.75.26.151:15970') label += ' (推荐)';
    else if (item.startsWith('124.75.')) label += ' (电信常用)';
    select.append(new Option(label, item));
  }
  select.append(new Option('自定义输入…', '__custom__'));

  if (!currentFCC) {
    select.value = '';
    if (input) { input.value = ''; input.hidden = true; }
    select.classList.remove('is-custom');
  } else if (allFCCs.includes(currentFCC)) {
    select.value = currentFCC;
    if (input) { input.value = currentFCC; input.hidden = true; }
    select.classList.remove('is-custom');
  } else {
    select.value = '__custom__';
    if (input) { input.value = currentFCC; input.hidden = false; }
    select.classList.add('is-custom');
  }
}

function fillSettings(result) {
  settings = result.settings;
  document.querySelectorAll('[data-setting]').forEach(el => {
    const [a, b] = el.dataset.setting.split('.');
    const v = settings[a][b];
    if (el.type === 'checkbox') el.checked = v;
    else el.value = Array.isArray(v) ? v.join('\n') : (v ?? '');
  });
  $('clear-key').checked = false;
  $('key-status').textContent = result.has_api_key ? '已保存 key（不会回显）' : '尚未配置 key；本地无认证接口可留空';
  populateFCCSelect(settings.forward?.fcc_list, settings.forward?.fcc);
  populateCatchupTemplateSelect(settings.forward?.catchup_template);
  if (settings.logos) {
    currentLogoSources = settings.logos.sources || (settings.logos.primary ? [settings.logos.primary] : ["https://github.com/sggc/SDU-IPTV-PRO/tree/main/logo"]);
  }
  currentGroupOrder = settings.group_order || ["央视频道", "卫视频道", "上海频道", "数字频道", "其它", "待识别"];
  currentGroupChannelOrders = settings.group_channel_order || {};
  if (typeof renderGroupOrderList === "function") renderGroupOrderList();
  if (typeof renderLogoSourcesList === "function") renderLogoSourcesList();
  if (typeof updateGroupChannelSelect === "function") updateGroupChannelSelect();
  updatePreview();
}

function getActiveFCC() {
  const select = $('fcc-select');
  const input = $('forward-fcc-input');
  if (select && select.value === '__custom__') {
    return input ? input.value.trim() : '';
  }
  return select ? select.value : (input ? input.value.trim() : '');
}

function collectSettings() {
  const value = structuredClone(settings);
  document.querySelectorAll('[data-setting]').forEach(el => {
    const [a, b] = el.dataset.setting.split('.');
    value[a][b] = el.type === 'checkbox' ? el.checked : el.type === 'number' ? Number(el.value) : el.tagName === 'TEXTAREA' ? el.value.split(/\s+/).filter(Boolean) : el.value.trim();
  });
  const activeFCC = getActiveFCC();
  value.forward.fcc = activeFCC;
  const activeTpl = getActiveCatchupTemplate();
  value.forward.catchup_template = activeTpl || "&utc=${start}&lutc=${end}";
  if (activeFCC) {
    if (!value.forward.fcc_list) value.forward.fcc_list = [];
    if (!value.forward.fcc_list.includes(activeFCC)) {
      value.forward.fcc_list.push(activeFCC);
    }
  }
  value.logos = { sources: currentLogoSources };
  value.group_order = currentGroupOrder;
  value.group_channel_order = currentGroupChannelOrders;
  return value;
}

function updatePreview() {
  if (!settings) return;
  const v = collectSettings().forward;
  const fccStr = v.fcc ? '?fcc=' + v.fcc : '';
  const tpl = v.catchup_template || '&utc=${start}&lutc=${end}';
  const normTpl = tpl.startsWith('&') || tpl.startsWith('?') ? (tpl.startsWith('?') ? '&' + tpl.slice(1) : tpl) : '&' + tpl;
  $('url-preview').textContent = `直播: http://${v.address || 'IP:端口'}/${v.protocol}/239.45.0.1:5140${fccStr}\n回看: http://${location.host || '当前服务'}/api/play?id=1&mode=unicast${normTpl}`;
}

document.querySelectorAll('[data-setting^="forward."]').forEach(el => el.addEventListener('input', updatePreview));

if ($('fcc-select')) {
  $('fcc-select').onchange = () => {
    const select = $('fcc-select');
    const input = $('forward-fcc-input');
    if (select.value === '__custom__') {
      select.classList.add('is-custom');
      if (input) {
        input.hidden = false;
        input.focus();
        input.select();
      }
    } else {
      select.classList.remove('is-custom');
      if (input) {
        input.hidden = true;
        input.value = select.value;
      }
    }
    updatePreview();
  };
}

if ($('forward-fcc-input')) {
  $('forward-fcc-input').addEventListener('input', updatePreview);
}

if ($('catchup-template-select')) {
  $('catchup-template-select').onchange = () => {
    const select = $('catchup-template-select');
    const input = $('forward-catchup-template-input');
    if (select.value === '__custom__') {
      select.classList.add('is-custom');
      if (input) {
        input.hidden = false;
        if (!input.value.trim() || CATCHUP_PRESETS.includes(input.value.trim())) {
          input.value = '&utc=${start}&lutc=${end}';
        }
        input.focus();
        input.select();
      }
    } else {
      select.classList.remove('is-custom');
      if (input) {
        input.hidden = true;
        input.value = select.value;
      }
    }
    updatePreview();
  };
}

if ($('forward-catchup-template-input')) {
  $('forward-catchup-template-input').addEventListener('input', updatePreview);
}

async function saveSettings() {
  if (!settings) throw Error('请先连接面板');
  await api('settings', 'PUT', { settings: collectSettings(), clear_api_key: $('clear-key').checked });
  fillSettings(await api('settings'));
}

$('save-settings').onclick = () => busy($('save-settings'), async () => { await saveSettings(); toast('设置已保存'); });
$('save-ai').onclick = () => busy($('save-ai'), async () => { await saveSettings(); toast('识别设置已保存'); });

const isInvalidID = id => !id || id === "0" || id === "-1" || String(id).startsWith("scan-") || id === "none" || id === "unknown" || isNaN(Number(id)) || Number(id) <= 0;
const isValidID = id => !isInvalidID(id);
const unknown = c => c.group === "待识别" || c.name.startsWith("未知频道") || isInvalidID(c.id);

async function toggleChannel(c) {
  const channelRef = c.key || c.id;
  const updated = { ...c, enabled: !c.enabled };
  if (!updated.id || !updated.id.trim()) updated.id = "0";
  await api("channels/" + encodeURIComponent(channelRef), "PUT", updated);
  c.enabled = !c.enabled;
  renderChannels();
  fillEPGChannels();
  toast(c.enabled ? `已启用 ${c.name}` : `已停用 ${c.name}`);
}

function toIgmpUrl(rawUrl) {
  if (!rawUrl) return '';
  return rawUrl.replace(/^(?:rtp|udp):\/\//i, 'igmp://');
}

function renderChannels() {
  $('count-total').textContent = channels.length;
  $('count-enabled').textContent = channels.filter(c => c.enabled).length;
  $('count-unknown').textContent = channels.filter(unknown).length;

  const query = $('search').value.toLowerCase(), filter = $('channel-filter').value;
  const selectedGroup = filter.startsWith('group:') ? filter.slice(6) : null;
  const filtered = channels.filter(c => {
    const displayUrl = toIgmpUrl(c.url);
    const textMatch = `${c.name} ${c.group} ${c.url} ${displayUrl} ${c.resolution || ''}`.toLowerCase().includes(query);
    if (!textMatch) return false;
    if (selectedGroup !== null) return (c.group || '未分组') === selectedGroup;
    if (filter === 'unknown') return unknown(c);
    if (filter === 'enabled') return c.enabled;
    if (filter === 'disabled') return !c.enabled;
    return true;
  });
  const pages = Math.max(1, Math.ceil(filtered.length / pageSize));
  page = Math.min(page, pages - 1);

  $('channel-rows').replaceChildren();
  $('empty').hidden = filtered.length !== 0;
  $('result-count').textContent = `共 ${filtered.length} 个频道`;
  $('page-label').textContent = `${page + 1} / ${pages}`;
  $('prev-page').disabled = page === 0;
  $('next-page').disabled = page >= pages - 1;

  for (const c of filtered.slice(page * pageSize, (page + 1) * pageSize)) {
    const row = document.createElement('tr');
    row.className = c.enabled ? 'channel-row-enabled' : 'channel-row-disabled';
    const cell = (cls) => {
      const td = document.createElement('td');
      if (cls) td.className = cls;
      row.append(td);
      return td;
    };

    const isLocal = c.logo && (c.logo.startsWith('/logos/') || c.logo.startsWith('/api/panel/logos/local/'));

    // 1. 名称 (Name) - 第一列，左对齐
    const nameTd = cell('col-name');
    const nameWrapper = document.createElement('div');
    nameWrapper.className = 'channel-name-wrapper';

    const nameTop = document.createElement('div');
    nameTop.className = 'channel-name-top';
    const nameTitle = document.createElement('span');
    nameTitle.className = 'channel-name-text';
    nameTitle.textContent = c.name;
    nameTitle.title = c.name;
    nameTop.append(nameTitle);

    if (isLocal) {
      const localBadge = document.createElement('span');
      localBadge.className = 'badge-local';
      localBadge.textContent = '本地';
      localBadge.title = '本地台标';
      nameTop.append(localBadge);
    }

    if (c.suggestion) {
      const sugSpan = document.createElement('span');
      sugSpan.className = 'channel-sug-tag';
      sugSpan.textContent = '💡 建议';
      sugSpan.title = `识别建议: ${c.suggestion.name || ''} (${Math.round(c.suggestion.confidence * 100)}%)`;
      nameTop.append(sugSpan);
    }

    nameWrapper.append(nameTop);
    nameTd.append(nameWrapper);

    // 2. 频道号 (Digital ID) - 第二列，单独一列在名称后面，居中，点击直接输入
    const idTd = cell('col-id');
    const isInvalid = isInvalidID(c.id);

    const idBadge = document.createElement('span');
    idBadge.className = 'badge-channel-id clickable' + (isInvalid ? ' is-invalid' : '');
    idBadge.textContent = isInvalid ? '--' : String(c.id);
    idBadge.title = '点击直接修改频道号';

    idBadge.onclick = (e) => {
      e.stopPropagation();
      const input = document.createElement('input');
      input.type = 'text';
      input.className = 'inline-id-input';
      input.value = isInvalid ? '' : String(c.id);
      input.placeholder = '--';
      input.maxLength = 10;

      let saved = false;
      const saveId = async () => {
        if (saved) return;
        saved = true;
        const newVal = input.value.trim();
        const finalId = newVal || '0';
        if (finalId === (c.id || '0')) {
          renderChannels();
          return;
        }
        try {
          const channelRef = c.key || c.id;
          const updated = { ...c, id: finalId };
          await api('channels/' + encodeURIComponent(channelRef), 'PUT', updated);
          c.id = finalId;
          renderChannels();
          fillEPGChannels();
          toast(`已更新频道号: ${c.name} -> ${finalId}`);
        } catch (err) {
          toast('更新失败: ' + err.message);
          renderChannels();
        }
      };

      input.onkeydown = (ev) => {
        if (ev.key === 'Enter') {
          ev.preventDefault();
          input.blur();
        } else if (ev.key === 'Escape') {
          saved = true;
          renderChannels();
        }
      };
      input.onblur = saveId;

      idTd.replaceChildren(input);
      input.focus();
      input.select();
    };

    idTd.append(idBadge);

    // 3. 台标 (Logo) - 第三列，居中
    const logoTd = cell('col-logo');
    const logoBox = document.createElement('div');
    logoBox.className = 'channel-logo-box has-logo-upload';
    if (c.logo && (isLocal || /^https?:\/\//i.test(c.logo))) {
      const img = document.createElement('img');
      img.src = c.logo;
      img.alt = c.name;
      img.loading = 'lazy';
      img.referrerPolicy = 'no-referrer';
      img.onerror = () => {
        img.hidden = true;
        logoBox.textContent = '📺';
      };
      logoBox.append(img);
    } else {
      logoBox.textContent = '📺';
    }
    const tipPrefix = isLocal ? '本地台标: ' : (c.logo ? '当前台标: ' : '未设置台标: ');
    logoBox.title = `${tipPrefix}${c.name}\n(点击直接上传本地台标)`;
    logoBox.onclick = (e) => {
      e.stopPropagation();
      tableUploadTargetChannel = c;
      const fileInput = $('table-logo-file-input');
      if (fileInput) {
        fileInput.value = '';
        fileInput.click();
      }
    };
    logoTd.append(logoBox);

    // 4. 状态 (Status: interactive toggle button) - 第四列，居中
    const statusTd = cell('col-status');
    const statusBtn = document.createElement('button');
    statusBtn.type = 'button';
    statusBtn.className = 'status-toggle-btn ' + (c.enabled ? 'is-enabled' : 'is-disabled');
    statusBtn.title = c.enabled ? '当前已启用，点击停用' : '当前已停用，点击启用';
    statusBtn.onclick = (e) => {
      e.stopPropagation();
      busy(statusBtn, () => toggleChannel(c));
    };
    const statusDot = document.createElement('span');
    statusDot.className = 'status-indicator-dot';
    const statusText = document.createElement('span');
    statusText.textContent = c.enabled ? '已启用' : '已停用';
    statusBtn.append(statusDot, statusText);
    statusTd.append(statusBtn);

    // 5. 分辨率 (Resolution) - 第五列，居中
    const resTd = cell('col-res');
    if (c.resolution) {
      const resSpan = document.createElement('span');
      resSpan.className = 'badge-res ' + getResolutionClass(c.resolution);
      resSpan.textContent = c.resolution;
      resSpan.title = `清晰度规格: ${getResolutionLabel(c.resolution)}`;
      resTd.append(resSpan);
    } else {
      const noRes = document.createElement('span');
      noRes.className = 'text-placeholder';
      noRes.textContent = '--';
      resTd.append(noRes);
    }

    // 6. 分组 (Group) - 第六列，居中
    const groupTd = cell('col-group');
    const groupBadge = document.createElement('span');
    groupBadge.className = 'badge';
    groupBadge.textContent = c.group || '未分组';
    groupTd.append(groupBadge);

    // 7. 内网源地址 (URL, 统一为 igmp://) - 第七列，居中
    const urlTd = cell('col-url');
    const displayUrl = toIgmpUrl(c.url);
    const urlText = document.createElement('span');
    urlText.className = 'channel-url-mono';
    urlText.textContent = displayUrl;
    urlTd.title = '点击复制: ' + displayUrl;
    urlTd.onclick = () => copyText(displayUrl, '已复制源地址: ' + displayUrl);
    urlTd.append(urlText);

    // 8. 操作 (Actions) - 第八列，居中对齐，紧凑排列
    const actionTd = cell('col-actions');
    const actions = document.createElement('div');
    actions.className = 'table-action-icons';

    const createActionBtn = (title, iconSvg, onClick, extraClass = '') => {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'action-icon-btn ' + extraClass;
      btn.title = title;
      btn.innerHTML = iconSvg;
      btn.onclick = (e) => {
        e.stopPropagation();
        busy(btn, onClick);
      };
      return btn;
    };

    actions.append(
      createActionBtn('编辑频道', '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>', () => edit(c)),
      createActionBtn('播放测试', '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>', () => play(c), 'btn-play'),
      createActionBtn('查看节目单', '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>', () => showEPG(c), 'btn-epg'),
      createActionBtn('探测流规格', '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polygon points="16.24 7.76 14.12 14.12 7.76 16.24 9.88 9.88 16.24 7.76"/></svg>', () => probeChannel(c), 'btn-probe'),
      createActionBtn('截取视频画面', '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/><circle cx="12" cy="13" r="4"/></svg>', () => snapshot(c), 'btn-snap'),
      createActionBtn('智能识别建议', '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/><line x1="11" y1="8" x2="11" y2="14"/><line x1="8" y1="11" x2="14" y2="11"/></svg>', () => identify(c), 'btn-ai')
    );
    actionTd.append(actions);

    $('channel-rows').append(row);
  }
}

function fillChannelFilterGroups() {
  const select = $('channel-filter');
  if (!select) return;
  const currentVal = select.value;
  const oldOpt = select.querySelector('optgroup');
  if (oldOpt) oldOpt.remove();

  const groups = [...new Set(channels.map(c => c.group || '未分组'))].filter(Boolean);
  if (groups.length === 0) return;

  const optgroup = document.createElement('optgroup');
  optgroup.label = '按分组筛选';
  for (const g of groups) {
    const count = channels.filter(c => (c.group || '未分组') === g).length;
    const opt = document.createElement('option');
    opt.value = 'group:' + g;
    opt.textContent = `${g} (${count})`;
    optgroup.appendChild(opt);
  }
  select.appendChild(optgroup);
  if (currentVal && [...select.options].some(o => o.value === currentVal)) {
    select.value = currentVal;
  }
}

async function loadChannels() {
  channels = await api('channels');
  fillChannelFilterGroups();
  renderChannels();
  fillEPGChannels();
  if (typeof renderGroupOrderList === 'function') renderGroupOrderList();
  if (typeof updateGroupChannelSelect === 'function') updateGroupChannelSelect();
}

for (const id of ['search', 'channel-filter']) {
  $(id).addEventListener('input', () => { page = 0; renderChannels(); });
}
$('prev-page').onclick = () => { page--; renderChannels(); };
$('next-page').onclick = () => { page++; renderChannels(); };
if ($('copy-m3u-btn')) {
  $('copy-m3u-btn').onclick = () => copyText(location.origin + '/api/m3u8', '已复制 M3U 订阅地址（原生支持 TiviMate / Televizo 自动回看）');
}

function edit(c) {
  editing = c;
  const form = $('channel-form');
  for (const key of ['id', 'operator_id', 'name', 'group', 'logo', 'url', 'unicast_url', 'resolution']) {
    if (form.elements[key]) form.elements[key].value = c?.[key] || '';
  }
  if (form.elements.url && form.elements.url.value) {
    form.elements.url.value = toIgmpUrl(form.elements.url.value);
  }
  if (!form.elements.id.value) {
    form.elements.id.value = "0";
  }
  if (c?.logo_automatic) form.elements.logo.value = '';
  form.elements.enabled.checked = c ? c.enabled : true;
  form.elements.catchup_days.value = c?.catchup_days || 0;
  $('dialog-title').textContent = c ? '编辑频道' : '添加频道';
  $('channel-error').textContent = '';
  if ($('match-candidates-box')) $('match-candidates-box').hidden = true;
  $('suggestion').hidden = !c?.suggestion;
  $('accept-suggestion').hidden = !c?.suggestion?.name;
  if (c?.suggestion) {
    const v = c.suggestion;
    let matchDesc = '';
    if (v.matched_id) {
      matchDesc = ` · 匹配台号 ${v.matched_id}${v.matched_name ? ' (' + v.matched_name + ')' : ''}`;
    }
    $('suggestion').textContent = `建议：${v.name || '无法确定'}${matchDesc} / ${v.group || '未分组'} · 置信度 ${Math.round(v.confidence * 100)}%\n${v.reason}`;
    if (!v.matched_id && v.name) {
      api('channels/match?name=' + encodeURIComponent(v.name)).then(res => {
        if (res && res.matched && res.id && editing === c) {
          v.matched_id = res.id;
          v.matched_name = res.name;
          v.matched_operator_id = res.operator_id;
          v.matched_group = res.group;
          const updatedMatch = ` · 匹配台号 ${res.id}${res.name ? ' (' + res.name + ')' : ''}${v.resolution ? ' · 分辨率 ' + v.resolution : ''}`;
          $('suggestion').textContent = `建议：${v.name || '无法确定'}${updatedMatch} / ${v.group || '未分组'} · 置信度 ${Math.round(v.confidence * 100)}%\n${v.reason}`;
        }
      }).catch(() => {});
    }
  }
  if ($('logo-file-input')) $('logo-file-input').value = '';
  $('logo-preview-box')?.classList.remove('is-dragover');
  fillGroupOptions();
  updateLogoPreview();
  $('channel-dialog').showModal();
}

function fillGroupOptions() {
  const select = $('existing-group'), current = $('channel-group').value;
  select.replaceChildren(new Option('未分组', ''));
  const groups = [...new Set([...channels.map(c => c.group), current].filter(g => g && g.trim()))].sort((a, b) => a.localeCompare(b, 'zh-CN'));
  for (const group of groups) select.append(new Option(group, group));
  const custom = new Option('自定义…', '');
  custom.dataset.custom = 'true';
  select.append(custom);
  select.value = current;
  select.hidden = false;
  $('custom-group-control').hidden = true;
}

$('existing-group').onchange = () => {
  const select = $('existing-group');
  if (select.selectedOptions[0]?.dataset.custom) {
    select.hidden = true;
    $('custom-group-control').hidden = false;
    $('channel-group').focus();
    $('channel-group').select();
  } else {
    $('channel-group').value = select.value;
  }
};
$('choose-group').onclick = () => { fillGroupOptions(); $('existing-group').focus(); };

let logoPreviewTimer;
function updateLogoPreview() {
  clearTimeout(logoPreviewTimer);
  const img = $('logo-preview'), status = $('logo-status'), custom = $('channel-logo').value.trim(), value = custom || (editing?.logo_automatic ? editing.logo : '');
  img.onload = img.onerror = null;
  img.hidden = true;
  img.removeAttribute('src');
  if (!value) { status.textContent = '未设置台标（可点击此处或拖拽图片上传本地台标）'; return; }
  const isLocal = value.startsWith('/logos/') || value.startsWith('/api/panel/logos/local/');
  if (!isLocal && !/^https?:\/\//i.test(value)) { status.textContent = '请输入 HTTP(S) 图片地址或本地台标路径（也可点击上传）'; return; }
  status.textContent = '正在加载台标…';
  img.onload = () => { img.hidden = false; status.textContent = isLocal ? '已加载本地台标（点击可更换）' : (custom ? '自定义台标（点击可更换）' : '自动匹配台标（点击可更换）'); };
  img.onerror = () => { img.hidden = true; status.textContent = '台标加载失败，请检查图片地址（点击可重新上传）'; };
  img.src = value;
}

$('channel-logo').oninput = () => { clearTimeout(logoPreviewTimer); logoPreviewTimer = setTimeout(updateLogoPreview, 400); };
$('channel-logo').onchange = updateLogoPreview;
$('new-channel').onclick = () => edit(null);
$('accept-suggestion').onclick = async () => {
  const form = $('channel-form');
  const v = editing?.suggestion;
  if (!v) return;
  const nameToUse = (v.name && v.name.trim().length > 0 && !/^[a-z0-9\-_+ ]+$/i.test(v.name)) ? v.name : (v.matched_name || v.name);
  form.elements.name.value = nameToUse;
  form.elements.group.value = v.matched_group || v.group || '';
  if (v.matched_id) {
    form.elements.id.value = v.matched_id;
  }
  if (v.matched_operator_id) {
    form.elements.operator_id.value = v.matched_operator_id;
  }
  if (!v.matched_id && (v.name || form.elements.name.value)) {
    try {
      const match = await api('channels/match?name=' + encodeURIComponent(v.name || form.elements.name.value));
      if (match && match.matched && match.id) {
        form.elements.id.value = match.id;
        if (match.operator_id) form.elements.operator_id.value = match.operator_id;
        if (match.name && !v.name) form.elements.name.value = match.name;
        if (match.group && !form.elements.group.value) form.elements.group.value = match.group;
      }
    } catch (e) {}
  }
  if (editing?.suggestion?.resolution) { $('channel-form').elements.resolution.value = editing.suggestion.resolution; }
  fillGroupOptions();
  $('match-logo').click();
};

let currentCandidates = [];
let currentMatchQuery = '';

function renderMatchCandidates(candidates, query) {
  const box = $('match-candidates-box');
  const list = $('match-candidates-list');
  const count = $('match-candidates-count');
  if (!box || !list) return;
  list.innerHTML = '';
  count.textContent = `共 ${candidates.length} 个候选`;

  candidates.forEach((cand, idx) => {
    const item = document.createElement('label');
    item.className = 'candidate-item';
    item.style.cssText = 'display:flex; align-items:center; justify-content:space-between; padding:6px 10px; border-radius:6px; cursor:pointer; background:var(--bg, #f8fafc); border:1px solid var(--line, #e2e8f0); font-size:12px; transition:all 0.15s;';
    item.onmouseenter = () => { item.style.borderColor = 'var(--primary, #2563eb)'; };
    item.onmouseleave = () => {
      const r = item.querySelector('input[type="radio"]');
      if (!r || !r.checked) item.style.borderColor = 'var(--line, #e2e8f0)';
    };

    const left = document.createElement('div');
    left.style.cssText = 'display:flex; align-items:center; gap:8px; flex-wrap:wrap;';

    const radio = document.createElement('input');
    radio.type = 'radio';
    radio.name = 'match_candidate_option';
    radio.value = idx;
    radio.checked = idx === 0;
    radio.style.margin = '0';
    radio.onchange = () => {
      list.querySelectorAll('.candidate-item').forEach(it => it.style.borderColor = 'var(--line, #e2e8f0)');
      item.style.borderColor = 'var(--primary, #2563eb)';
    };

    const badge = document.createElement('span');
    badge.className = 'badge';
    badge.textContent = `台号 ${cand.id}`;
    badge.style.cssText = 'font-family:monospace; font-weight:600; padding:1px 6px;';

    const name = document.createElement('strong');
    name.textContent = cand.name;

    left.appendChild(radio);
    left.appendChild(badge);
    left.appendChild(name);

    if (cand.operator_id) {
      const op = document.createElement('span');
      op.className = 'muted';
      op.style.fontSize = '11px';
      op.textContent = `(${cand.operator_id})`;
      left.appendChild(op);
    }
    if (cand.group) {
      const grp = document.createElement('span');
      grp.className = 'muted';
      grp.style.fontSize = '11px';
      grp.textContent = `[${cand.group}]`;
      left.appendChild(grp);
    }
    if (cand.is_saved) {
      const savedBadge = document.createElement('span');
      savedBadge.className = 'badge';
      savedBadge.style.cssText = 'background:#e6f4ea; color:#137333; font-size:10px; padding:1px 5px;';
      savedBadge.textContent = '已保存映射';
      left.appendChild(savedBadge);
    }

    const right = document.createElement('small');
    right.className = 'muted';
    right.textContent = cand.is_saved ? '关键词优先' : `匹配度 ${cand.score}%`;

    item.appendChild(left);
    item.appendChild(right);

    item.ondblclick = () => {
      radio.checked = true;
      $('apply-candidate-btn').click();
    };

    list.appendChild(item);
  });

  box.hidden = false;
}

$('match-channel-id').onclick = async () => {
  const form = $('channel-form');
  const query = (form.elements.name.value || editing?.suggestion?.name || '').trim();
  if (!query) {
    toast('请先输入频道名称再匹配台号', true);
    return;
  }
  currentMatchQuery = query;
  await busy($('match-channel-id'), async () => {
    const res = await api('channels/match?name=' + encodeURIComponent(query));
    if (res && res.candidates && res.candidates.length > 0) {
      currentCandidates = res.candidates;
      renderMatchCandidates(res.candidates, query);
    } else if (res && res.matched && res.id) {
      currentCandidates = [{ id: res.id, name: res.name, operator_id: res.operator_id, group: res.group, score: 100, is_saved: res.is_saved }];
      renderMatchCandidates(currentCandidates, query);
    } else {
      if ($('match-candidates-box')) $('match-candidates-box').hidden = true;
      toast(`未在已知频道中匹配到「${query}」的台号`, true);
    }
  });
};

if ($('apply-candidate-btn')) {
  $('apply-candidate-btn').onclick = async () => {
    const form = $('channel-form');
    const selectedRadio = document.querySelector('input[name="match_candidate_option"]:checked');
    if (!selectedRadio) {
      toast('请先选择一个候选频道', true);
      return;
    }
    const cand = currentCandidates[Number(selectedRadio.value)];
    if (!cand) return;

    form.elements.id.value = cand.id;
    if (cand.operator_id) {
      form.elements.operator_id.value = cand.operator_id;
    }
    if (!form.elements.name.value) {
      form.elements.name.value = cand.name;
    }
    if (cand.group && !form.elements.group.value) {
      form.elements.group.value = cand.group;
      fillGroupOptions();
    }

    const query = currentMatchQuery || form.elements.name.value.trim();
    const shouldSave = $('save-mapping-check')?.checked;

    if (shouldSave && query) {
      try {
        await api('mappings', 'POST', {
          keyword: query,
          target_id: cand.id,
          target_name: cand.name,
          operator_id: cand.operator_id || '',
          group: cand.group || '',
        });
        toast(`已应用台号 ${cand.id} (${cand.name}) 并保存「${query}」映射`);
        loadMappings();
      } catch (e) {
        toast(`已应用台号 ${cand.id} (${cand.name})，保存映射失败: ${e.message}`, true);
      }
    } else {
      toast(`已应用台号 ${cand.id} (${cand.name})`);
    }

    $('match-candidates-box').hidden = true;
  };
}

if ($('close-candidates-btn')) {
  $('close-candidates-btn').onclick = () => {
    if ($('match-candidates-box')) $('match-candidates-box').hidden = true;
  };
}

$('channel-form').onsubmit = async event => {
  event.preventDefault();
  const form = event.target;
  const c = Object.fromEntries(new FormData(form));
  c.operator_id = (c.operator_id || '').trim();
  c.id = (c.id || '').trim() || "0";
  c.enabled = form.elements.enabled.checked;
  c.catchup_days = Number(form.elements.catchup_days.value);
  if (editing?.key) c.key = editing.key;
  if (editing?.original_id) c.original_id = editing.original_id;
  const channelRef = editing?.key || editing?.id || c.key || c.id;
  await busy(form.querySelector('button[type="submit"]'), async () => {
    try {
      await api('channels/' + encodeURIComponent(channelRef), 'PUT', c);
      $('channel-dialog').close();
      await loadChannels();
      toast('频道已保存，导出列表已更新');
    } catch (e) {
      $('channel-error').textContent = e.message;
      throw e;
    }
  });
};

let snapshotURL;
async function snapshot(c) {
  $('snapshot-image').hidden = true;
  $('snapshot-status').textContent = '正在截取 ' + c.name + '，最多等待 20 秒…';
  $('snapshot-dialog').showModal();
  try {
    const channelRef = c.key || c.original_id || c.id;
    const blob = await api('channels/' + encodeURIComponent(channelRef) + '/snapshot');
    if (snapshotURL) URL.revokeObjectURL(snapshotURL);
    snapshotURL = URL.createObjectURL(blob);
    $('snapshot-image').src = snapshotURL;
    $('snapshot-image').hidden = false;
    $('snapshot-status').textContent = c.name;
  } catch (e) {
    $('snapshot-status').textContent = e.message;
    throw e;
  }
}

async function identify(c) {
  toast('正在截图并识别 ' + c.name + '…');
  const channelRef = c.key || c.original_id || c.id;
  await api('channels/' + encodeURIComponent(channelRef) + '/identify', 'POST');
  await loadChannels();
  edit(channels.find(v => (v.key ? v.key === c.key : v.id === c.id)) || channels.find(v => v.id === c.id));
}

$('show-unknown').onclick = () => {
  $('channel-filter').value = 'unknown';
  page = 0;
  renderChannels();
  switchChannelView('list');
  tab('channels');
};

const states = { idle: '待命', running: '运行中', stopping: '正在停止', completed: '已完成', cancelled: '已停止', failed: '失败' };

async function status() {
  const value = await api('status');
  const scan = value.scan, refresh = value.refresh;
  $('scan-state').textContent = states[scan.state] || scan.state;
  $('scan-progress').max = scan.total || 1;
  $('scan-progress').value = scan.done;
  $('scan-count').textContent = `已探测 ${scan.done} / ${scan.total}，发现可用流 ${scan.found}`;
  $('scan-error').textContent = scan.error ? '最近一次探测信息：' + scan.error : '';
  $('start-scan').disabled = ['running', 'stopping'].includes(scan.state);
  $('stop-scan').disabled = scan.state !== 'running';
  $('sync').disabled = refresh.state === 'running';
  $('refresh-status').textContent = {
    running: '正在从运营商门户同步频道…',
    completed: '最近一次同步完成',
    failed: '最近一次同步失败：' + refresh.error
  }[refresh.state] || '';
  if (['running', 'stopping'].includes(scan.state) || refresh.state === 'running') setTimeout(status, 2000);
  if (lastScanState === 'running' && scan.state !== 'running') await loadChannels();
  if (lastRefreshState === 'running' && refresh.state !== 'running') await loadChannels();
  lastScanState = scan.state;
  lastRefreshState = refresh.state;
}

$('start-scan').onclick = () => busy($('start-scan'), async () => { await saveSettings(); await api('scan/start', 'POST'); await status(); toast('组播扫描已启动'); });
$('stop-scan').onclick = () => busy($('stop-scan'), async () => { await api('scan/stop', 'POST'); await status(); });
$('sync').onclick = () => busy($('sync'), async () => { await api('refresh', 'POST'); await status(); toast('运营商频道同步已启动'); });

async function init() {
  try {
    fillSettings(await api('settings'));
    await loadChannels();
    await loadMappings();
    await status();
    $('login').hidden = true;
    $('workspace').hidden = false;
  } catch (e) {
    $('login').hidden = false;
    $('workspace').hidden = true;
  }
}

$('reload').onclick = () => busy($('reload'), async () => {
  await loadChannels();
  await loadMappings();
  await status();
  if (!$('tab-epg').hidden) await epgStatus();
  toast('数据已刷新');
});

$('login-form').onsubmit = async e => {
  e.preventDefault();
  sessionStorage.setItem('iptv-token', $('token').value.trim());
  $('login-error').textContent = '';
  try {
    await init();
  } catch (err) {
    $('login-error').textContent = err.message;
  }
};

let playing;
function play(c) {
  playing = c;
  $('play-title').textContent = c.name;
  $('play-mode').value = settings.forward.play_mode || 'multicast';
  $('play-replay').checked = false;
  const local = date => new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
  $('play-start').value = local(new Date(Date.now() - 3600000));
  $('play-end').value = local(new Date());
  updatePlayAddress();
  $('play-dialog').showModal();
}

function updatePlayAddress() {
  const mode = $('play-mode').value, replay = $('play-replay').checked;
  $('play-start-label').hidden = $('play-end-label').hidden = !replay;
  let address = playing.play_url, notice = '';
  if (mode === 'http' && !playing.operator_id) notice = '该频道未关联运营商 HTTP 源';
  else if (mode === 'unicast' && !playing.unicast_url) notice = '该频道未配置单播源';
  else if (replay && !playing.catchup_days) notice = '该频道未启用回看';
  else if (mode === 'multicast' && replay) notice = '回看请选择 HTTP / HLS 或单播方式';
  else if (mode !== 'multicast') {
    const u = new URL('/api/play', location.origin);
    u.searchParams.set('id', playing.id);
    u.searchParams.set('mode', mode);
    if (replay) {
      const start = new Date($('play-start').value).getTime() / 1000, end = new Date($('play-end').value).getTime() / 1000;
      if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) notice = '请选择有效的开始和结束时间';
      u.searchParams.set('start', String(start));
      u.searchParams.set('end', String(end));
    }
    address = u.href;
  } else {
    const u = new URL(playing.url);
    if (['igmp:', 'rtp:', 'udp:'].includes(u.protocol) && settings.forward.address) {
      const f = new URL('http://' + settings.forward.address + '/' + settings.forward.protocol + '/' + u.host + u.search);
      if (!f.searchParams.get('fcc') && settings.forward.fcc) f.searchParams.set('fcc', settings.forward.fcc);
      address = f.href;
    } else address = playing.url;
  }
  $('play-notice').textContent = notice || (replay ? '可回看 ' + playing.catchup_days + ' 天；HTTP 回看按运营商节目边界结束。' : '');
  $('play-address').textContent = notice ? '' : address;
  $('play-open').hidden = Boolean(notice);
  if (!notice) $('play-open').href = address;
  else $('play-open').removeAttribute('href');
}

for (const id of ['play-mode', 'play-replay', 'play-start', 'play-end']) {
  $(id).onchange = updatePlayAddress;
}

$('match-logo').onclick = () => busy($('match-logo'), async () => {
  const form = $('channel-form'), name = form.elements.name.value.trim(), group = form.elements.group.value.trim();
  if (!name) throw Error('请先填写频道名称');
  const version = ++logoMatchVersion;
  pendingLogoMatch = null;
  $('logo-match-result').hidden = false;
  $('accept-logo-match').hidden = true;
  $('download-logo-match').hidden = true;
  $('logo-match-image').hidden = true;
  $('logo-match-image').removeAttribute('src');
  $('logo-match-reason').textContent = '正在使用已配置模型匹配台标…';
  try {
    const result = await api('logos/match', 'POST', { name, group });
    if (version !== logoMatchVersion || !$('channel-dialog').open) return;
    if (name !== form.elements.name.value.trim() || group !== form.elements.group.value.trim()) {
      $('logo-match-reason').textContent = '频道名称或分组已改变，请重新匹配';
      return;
    }
    $('logo-match-reason').textContent = `${result.logo ? '匹配：' + result.name : '未找到可靠匹配'} · 置信度 ${Math.round(result.confidence * 100)}% · ${result.reason || ''}`;
    if (result.logo) {
      pendingLogoMatch = { ...result, channelName: name, group };
      const img = $('logo-match-image');
      img.onload = () => { img.hidden = false; };
      img.onerror = () => { img.hidden = true; $('logo-match-reason').textContent += '（图片加载失败）'; };
      img.src = result.logo;
      $('accept-logo-match').hidden = false;
      $('download-logo-match').hidden = false;
    }
  } catch (e) {
    if (version === logoMatchVersion) $('logo-match-reason').textContent = e.message;
    throw e;
  }
});

$('accept-logo-match').onclick = () => {
  const form = $('channel-form'), result = pendingLogoMatch;
  if (!result) return;
  if (result.channelName !== form.elements.name.value.trim() || result.group !== form.elements.group.value.trim()) {
    toast('频道名称或分组已改变，请重新匹配', true);
    return;
  }
  form.elements.logo.value = result.logo;
  updateLogoPreview();
  toast('已填入远程台标，保存频道后生效');
};

$('download-logo-match').onclick = () => busy($('download-logo-match'), async () => {
  const form = $('channel-form'), result = pendingLogoMatch;
  if (!result) return;
  if (result.channelName !== form.elements.name.value.trim() || result.group !== form.elements.group.value.trim()) {
    toast('频道名称或分组已改变，请重新匹配', true);
    return;
  }
  const dlRes = await api('logos/download', 'POST', { url: result.logo, name: result.channelName });
  form.elements.logo.value = dlRes.url;
  updateLogoPreview();
  toast('台标已下载到本地并填入，保存频道后生效');
});

$('download-current-logo').onclick = () => busy($('download-current-logo'), async () => {
  const form = $('channel-form');
  const url = form.elements.logo.value.trim();
  const name = form.elements.name.value.trim();
  if (!url) throw Error('请先输入或匹配台标 URL');
  if (url.startsWith('/logos/') || url.startsWith('/api/panel/logos/local/')) {
    toast('当前已经是本地台标：' + url);
    return;
  }
  const dlRes = await api('logos/download', 'POST', { url, name: name || 'logo' });
  form.elements.logo.value = dlRes.url;
  updateLogoPreview();
  toast('台标已下载并保存到本地：' + dlRes.url);
});

async function uploadLogoFile(file, channelName = '') {
  if (!file) throw Error('请选择要上传的台标图片文件');
  const validTypes = ['image/png', 'image/jpeg', 'image/webp', 'image/svg+xml', 'image/gif', 'image/x-icon'];
  const validExts = /\.(png|jpe?g|webp|svg|gif|ico)$/i;
  if ((file.type && !validTypes.includes(file.type)) && !validExts.test(file.name)) {
    throw Error('不支持的文件格式，仅支持 PNG、JPG、WEBP、SVG、GIF、ICO 图片');
  }
  if (file.size > 10 * 1024 * 1024) {
    throw Error('台标图片大小不能超过 10MB');
  }
  const fd = new FormData();
  fd.append('file', file);
  if (channelName && channelName.trim()) {
    fd.append('name', channelName.trim());
  }
  return await api('logos/upload', 'POST', fd);
}

async function handleDialogLogoUpload(file) {
  const form = $('channel-form');
  const uploadBtn = $('upload-logo-btn');
  const previewStatus = $('logo-status');
  const channelName = form.elements.name.value.trim() || editing?.name || '';
  if (uploadBtn) uploadBtn.disabled = true;
  if (previewStatus) previewStatus.textContent = '正在上传本地台标…';
  try {
    toast('正在上传本地台标…');
    const res = await uploadLogoFile(file, channelName);
    if (res && res.url) {
      form.elements.logo.value = res.url;
      if (editing) {
        editing.logo_automatic = false;
      }
      updateLogoPreview();
      toast('台标已成功上传并保存为本地文件：' + res.url);
    }
  } catch (err) {
    updateLogoPreview();
    toast('上传台标失败: ' + err.message, true);
  } finally {
    if (uploadBtn) uploadBtn.disabled = false;
  }
}

if ($('upload-logo-btn')) {
  $('upload-logo-btn').onclick = () => {
    const input = $('logo-file-input');
    if (input) {
      input.value = '';
      input.click();
    }
  };
}

if ($('logo-file-input')) {
  $('logo-file-input').onchange = async (e) => {
    const file = e.target.files?.[0];
    if (file) {
      await handleDialogLogoUpload(file);
    }
    e.target.value = '';
  };
}

const logoPreviewBox = $('logo-preview-box');
if (logoPreviewBox) {
  logoPreviewBox.onclick = () => {
    const input = $('logo-file-input');
    if (input) {
      input.value = '';
      input.click();
    }
  };
  ['dragenter', 'dragover'].forEach(name => {
    logoPreviewBox.addEventListener(name, (e) => {
      e.preventDefault();
      e.stopPropagation();
      if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy';
      logoPreviewBox.classList.add('is-dragover');
    });
  });
  ['dragleave', 'dragend'].forEach(name => {
    logoPreviewBox.addEventListener(name, (e) => {
      e.preventDefault();
      e.stopPropagation();
      logoPreviewBox.classList.remove('is-dragover');
    });
  });
  logoPreviewBox.addEventListener('drop', async (e) => {
    e.preventDefault();
    e.stopPropagation();
    logoPreviewBox.classList.remove('is-dragover');
    const files = e.dataTransfer?.files;
    if (files && files.length > 0) {
      await handleDialogLogoUpload(files[0]);
    }
  });
}

let tableUploadTargetChannel = null;
const tableLogoInput = $('table-logo-file-input');
if (tableLogoInput) {
  tableLogoInput.onchange = async (e) => {
    const file = e.target.files?.[0];
    const target = tableUploadTargetChannel;
    e.target.value = '';
    tableUploadTargetChannel = null;
    if (!file || !target) return;
    try {
      toast(`正在为「${target.name}」上传并保存本地台标…`);
      const res = await uploadLogoFile(file, target.name);
      if (res && res.url) {
        const channelRef = target.key || target.original_id || target.id;
        const updated = { ...target, logo: res.url, logo_automatic: false };
        if (!updated.id || !updated.id.trim()) updated.id = "0";
        await api('channels/' + encodeURIComponent(channelRef), 'PUT', updated);
        target.logo = res.url;
        target.logo_automatic = false;
        renderChannels();
        fillEPGChannels();
        toast(`已成功为「${target.name}」更新本地台标！`);
      }
    } catch (err) {
      toast(`上传台标失败: ${err.message}`, true);
    }
  };
}

$('batch-download-logos').onclick = () => busy($('batch-download-logos'), async () => {
  const targets = channels.filter(c => c.logo && (c.logo.startsWith('http://') || c.logo.startsWith('https://')));
  if (targets.length === 0) {
    toast('当前频道列表没有使用远程 URL 的台标');
    return;
  }
  if (!confirm(`发现 ${targets.length} 个频道的台标为远程 URL，是否开始下载并保存到本地？`)) {
    return;
  }
  let success = 0, failed = 0;
  for (let i = 0; i < targets.length; i++) {
    const c = targets[i];
    toast(`正在下载本地台标 (${i + 1}/${targets.length}): ${c.name}...`);
    try {
      const res = await api('logos/download', 'POST', { url: c.logo, name: c.name });
      const updated = { ...c, logo: res.url };
      const channelRef = c.key || c.original_id || c.id;
      await api('channels/' + encodeURIComponent(channelRef), 'PUT', updated);
      success++;
    } catch (e) {
      failed++;
    }
  }
  await loadChannels();
  toast(`本地化完成：成功 ${success} 个${failed ? '，失败 ' + failed + ' 个' : ''}`);
});

/* ==========================================================================
   Modern EPG Logic
   ========================================================================== */

function getShanghaiTodayStr() {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  }).format(new Date());
}

function updateEPGHero() {
  const id = $('epg-channel').value;
  const c = channels.find(c => (c.key ? c.key === id : c.id === id)) || channels.find(c => c.id === id);
  if (!c) {
    $('epg-hero-name').textContent = '未选择频道';
    $('epg-hero-group').textContent = '无';
    $('epg-hero-meta').textContent = '请在下方选择运营商频道查看节目单';
    $('epg-hero-catchup').textContent = '无时移';
    $('epg-hero-catchup').className = 'badge off';
    $('epg-hero-logo').hidden = true;
    $('epg-hero-placeholder').hidden = false;
    $('epg-hero-play').disabled = true;
    return;
  }

  $('epg-hero-name').textContent = c.name;
  $('epg-hero-group').textContent = c.group || '未分组';
  const opDesc = c.operator_id ? ` · 字母ID: ${c.operator_id}` : '';
  $('epg-hero-meta').textContent = `数字频道: ${c.id}${opDesc} · ${c.operator_id ? '运营商频道' : '自定义频道'}${c.unicast_url ? ' · 单播' : ''}`;
  
  if (c.catchup_days > 0) {
    $('epg-hero-catchup').textContent = `支持 ${c.catchup_days} 天回看`;
    $('epg-hero-catchup').className = 'badge badge-green';
  } else {
    $('epg-hero-catchup').textContent = '不支持回看';
    $('epg-hero-catchup').className = 'badge off';
  }

  const logoImg = $('epg-hero-logo');
  const placeholder = $('epg-hero-placeholder');
  const isLocalEPGLogo = c.logo && (c.logo.startsWith('/logos/') || c.logo.startsWith('/api/panel/logos/local/'));
  if (c.logo && (isLocalEPGLogo || /^https?:\/\//i.test(c.logo))) {
    logoImg.src = c.logo;
    logoImg.onload = () => { logoImg.hidden = false; placeholder.hidden = true; };
    logoImg.onerror = () => { logoImg.hidden = true; placeholder.hidden = false; };
  } else {
    logoImg.hidden = true;
    placeholder.hidden = false;
  }

  $('epg-hero-play').disabled = false;
  $('epg-hero-play').onclick = () => play(c);
}

function renderDateRibbon() {
  const container = $('epg-date-ribbon');
  if (!container) return;
  container.replaceChildren();

  const currentDate = $('epg-date').value || getShanghaiTodayStr();
  const todayStr = getShanghaiTodayStr();
  const pastDays = settings?.epg?.past_days ?? 7;
  const futureDays = settings?.epg?.future_days ?? 3;

  const now = new Date(todayStr + 'T00:00:00+08:00');

  for (let i = -pastDays; i <= futureDays; i++) {
    const d = new Date(now.getTime() + i * 86400000);
    const dateStr = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit' }).format(d);
    const weekday = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][d.getDay()];

    let label = weekday;
    if (i === 0) label = '今天';
    else if (i === -1) label = '昨天';
    else if (i === -2) label = '前天';
    else if (i === 1) label = '明天';
    else if (i === 2) label = '后天';

    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'date-pill' + (dateStr === currentDate ? ' active' : '') + (i === 0 ? ' is-today' : '');

    const labelSpan = document.createElement('span');
    labelSpan.className = 'date-pill-label';
    labelSpan.textContent = label;
    if (i === 0) {
      const marker = document.createElement('span');
      marker.className = 'live-marker';
      marker.textContent = 'LIVE';
      labelSpan.append(marker);
    }

    const subSpan = document.createElement('span');
    subSpan.className = 'date-pill-sub';
    subSpan.textContent = dateStr.slice(5);

    btn.append(labelSpan, subSpan);
    btn.onclick = () => {
      $('epg-date').value = dateStr;
      renderDateRibbon();
      loadEPG().catch(e => { $('epg-message').textContent = e.message; });
    };

    container.append(btn);
  }
}

function fillEPGChannels() {
  const select = $('epg-channel'), previous = select.value;
  select.replaceChildren();
  const validChannels = channels.filter(c => c.enabled && (c.operator_id || (c.source === 'iptv' && !isInvalidID(c.id)) || isValidID(c.id)));
  for (const c of validChannels) {
    const label = `${c.name} · 台号:${c.id}` + (c.operator_id ? ` [${c.operator_id}]` : '') + ` (${c.group || '未分组'})`;
    select.append(new Option(label, c.key || c.id));
  }
  if ([...select.options].some(o => o.value === previous)) {
    select.value = previous;
  }
  if (!$('epg-date').value) {
    $('epg-date').value = getShanghaiTodayStr();
  }
  $('epg-url').textContent = new URL('/api/epg', location.origin).href;
  renderDateRibbon();
  updateEPGHero();
}

function switchEPGChannel(direction) {
  const select = $('epg-channel');
  const validChannels = channels.filter(c => c.enabled && (c.operator_id || (c.source === 'iptv' && !isInvalidID(c.id)) || isValidID(c.id)));
  if (!validChannels.length) return;
  const currentIdx = validChannels.findIndex(c => (c.key ? c.key === select.value : c.id === select.value));
  let nextIdx = currentIdx + direction;
  if (nextIdx < 0) nextIdx = validChannels.length - 1;
  if (nextIdx >= validChannels.length) nextIdx = 0;
  select.value = validChannels[nextIdx].key || validChannels[nextIdx].id;
  updateEPGHero();
  loadEPG().catch(e => { $('epg-message').textContent = e.message; });
}

$('epg-prev-channel').onclick = () => switchEPGChannel(-1);
$('epg-next-channel').onclick = () => switchEPGChannel(1);
$('toggle-epg-settings').onclick = () => {
  const p = $('epg-settings-panel');
  p.hidden = !p.hidden;
};
$('copy-epg-url').onclick = () => copyText($('epg-url').textContent, 'XMLTV 地址已复制到剪贴板');

async function epgStatus() {
  const v = await api('epg/status');
  const stateName = states[v.state] || v.state;
  const timeStr = v.updated_at && !v.updated_at.startsWith('0001') ? new Date(v.updated_at).toLocaleString() : '尚未同步';

  $('epg-status').textContent = `${v.backend === 'mysql' ? 'MySQL' : '本地文件'} · ${stateName} · ${v.done}/${v.total} 个频道 · 已存 ${v.programmes} 条节目 · ${timeStr}${v.error ? ' · ' + v.error : ''}`;
  $('epg-progress').max = v.total || 1;
  $('epg-progress').value = v.done;
  $('sync-epg').disabled = v.state === 'running';

  // Update status chips
  $('val-backend').textContent = v.backend === 'mysql' ? 'MySQL 数据库' : '本地文件 (JSON)';
  if (v.state === 'running') {
    $('val-state').innerHTML = '<span class="pulse-dot" style="background:#f59e0b;"></span> 同步中...';
  } else {
    $('val-state').innerHTML = '<span class="pulse-dot"></span> 就绪';
  }
  $('val-channels').textContent = `${v.done} / ${v.total} 频道`;
  $('val-programmes').textContent = `${v.programmes.toLocaleString()} 条`;
  $('val-updated').textContent = timeStr;

  if (lastEPGState === 'running' && v.state !== 'running') await loadEPG();
  lastEPGState = v.state;
}

function renderEPGList() {
  const container = $('epg-programmes');
  container.replaceChildren();

  const id = $('epg-channel').value;
  const c = channels.find(c => (c.key ? c.key === id : c.id === id)) || channels.find(c => c.id === id);
  const now = Date.now() / 1000;
  const query = ($('epg-search').value || '').trim().toLowerCase();

  const timeFmt = seconds => new Date(seconds * 1000).toLocaleTimeString('zh-CN', {
    timeZone: 'Asia/Shanghai',
    hour: '2-digit',
    minute: '2-digit'
  });

  // Calculate stats
  let countLive = 0, countCatchup = 0, countUpcoming = 0;
  for (const p of currentEPGProgrammes) {
    const isLive = p.start <= now && p.end > now;
    const isCatchup = p.end <= now && c && c.catchup_days && p.start >= now - c.catchup_days * 86400;
    const isUpcoming = p.start > now;
    if (isLive) countLive++;
    if (isCatchup) countCatchup++;
    if (isUpcoming) countUpcoming++;
  }

  $('count-all').textContent = currentEPGProgrammes.length;
  $('count-live').textContent = countLive;
  $('count-catchup').textContent = countCatchup;
  $('count-upcoming').textContent = countUpcoming;

  // Filter programmes
  const filtered = currentEPGProgrammes.filter(p => {
    const isLive = p.start <= now && p.end > now;
    const isCatchup = p.end <= now && c && c.catchup_days && p.start >= now - c.catchup_days * 86400;
    const isUpcoming = p.start > now;

    if (epgCurrentFilter === 'live' && !isLive) return false;
    if (epgCurrentFilter === 'catchup' && !isCatchup) return false;
    if (epgCurrentFilter === 'upcoming' && !isUpcoming) return false;

    if (query && !p.title.toLowerCase().includes(query) && !(p.desc && p.desc.toLowerCase().includes(query))) return false;
    return true;
  });

  if (!filtered.length) {
    const empty = document.createElement('div');
    empty.className = 'empty';
    empty.style.padding = '40px 10px';
    empty.innerHTML = `
      <div class="empty-icon" style="font-size:32px;">📅</div>
      <h3 style="font-size:15px;">暂无匹配节目</h3>
      <p style="font-size:12.5px;">该筛选条件下没有找到节目，请尝试更换日期、清除搜索词或点击立即同步。</p>
    `;
    container.append(empty);
    return;
  }

  for (const p of filtered) {
    const isLive = p.start <= now && p.end > now;
    const isCatchup = p.end <= now && c && c.catchup_days && p.start >= now - c.catchup_days * 86400;
    const isUpcoming = p.start > now;

    const card = document.createElement('div');
    card.className = 'epg-item' + (isLive ? ' is-live' : '') + (isCatchup ? ' is-catchup' : '') + (isUpcoming ? ' is-upcoming' : '');

    // Col 1: Time & duration
    const timeCol = document.createElement('div');
    timeCol.className = 'epg-time-col';

    const timeRange = document.createElement('div');
    timeRange.className = 'epg-time-range';
    timeRange.textContent = `${timeFmt(p.start)} – ${timeFmt(p.end)}`;

    const durationMin = Math.max(1, Math.round((p.end - p.start) / 60));
    const duration = document.createElement('div');
    duration.className = 'epg-duration';
    duration.textContent = `${durationMin} 分钟`;

    timeCol.append(timeRange, duration);

    // Col 2: Program title, Live progress, badges
    const infoCol = document.createElement('div');
    infoCol.className = 'epg-info-col';

    const titleRow = document.createElement('div');
    titleRow.className = 'epg-title-row';

    const titleEl = document.createElement('span');
    titleEl.className = 'epg-program-title';
    titleEl.textContent = p.title;
    titleRow.append(titleEl);

    if (isLive) {
      const liveBadge = document.createElement('span');
      liveBadge.className = 'epg-live-badge';
      const liveDot = document.createElement('span');
      liveDot.className = 'live-dot';
      liveBadge.append(liveDot, document.createTextNode('正在直播'));
      titleRow.append(liveBadge);

      // Live Progress Bar
      const totalSec = Math.max(1, p.end - p.start);
      const elapsedSec = Math.max(0, now - p.start);
      const pct = Math.min(100, Math.max(0, Math.round((elapsedSec / totalSec) * 100)));
      const elapsedMin = Math.floor(elapsedSec / 60);
      const remainMin = Math.max(0, Math.ceil((p.end - now) / 60));

      const progressWrap = document.createElement('div');
      progressWrap.className = 'epg-live-progress';
      const track = document.createElement('div');
      track.className = 'epg-progress-track';
      const fill = document.createElement('div');
      fill.className = 'epg-progress-fill';
      fill.style.width = pct + '%';
      track.append(fill);

      const meta = document.createElement('span');
      meta.className = 'epg-progress-meta';
      meta.textContent = `已播 ${elapsedMin} 分钟 · 剩余 ${remainMin} 分钟 (${pct}%)`;

      progressWrap.append(track, meta);
      infoCol.append(titleRow, progressWrap);
    } else {
      if (isCatchup) {
        const catchupBadge = document.createElement('span');
        catchupBadge.className = 'badge badge-green';
        catchupBadge.textContent = '可回看';
        titleRow.append(catchupBadge);
      } else if (isUpcoming) {
        const upcomingBadge = document.createElement('span');
        upcomingBadge.className = 'badge badge-purple';
        const startDiff = Math.ceil((p.start - now) / 60);
        const diffText = startDiff >= 60 ? `${Math.floor(startDiff / 60)}小时${startDiff % 60 ? (startDiff % 60) + '分' : ''}后开始` : `${startDiff}分钟后开始`;
        upcomingBadge.textContent = `即将播出 · ${diffText}`;
        titleRow.append(upcomingBadge);
      } else {
        const endedBadge = document.createElement('span');
        endedBadge.className = 'badge off';
        endedBadge.textContent = '已结束';
        titleRow.append(endedBadge);
      }
      infoCol.append(titleRow);
    }

    if (p.desc) {
      const descEl = document.createElement('div');
      descEl.className = 'epg-program-desc';
      descEl.textContent = p.desc;
      descEl.title = p.desc;
      descEl.onclick = (e) => {
        e.stopPropagation();
        descEl.classList.toggle('expanded');
      };
      infoCol.append(descEl);
    }

    // Col 3: Actions
    const actionCol = document.createElement('div');
    actionCol.className = 'epg-action-col';

    if (isLive) {
      const playBtn = document.createElement('button');
      playBtn.type = 'button';
      playBtn.className = 'btn-live-play';
      playBtn.innerHTML = `
        <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>
        观看直播
      `;
      playBtn.onclick = () => {
        play(c);
        $('play-mode').value = 'http';
        updatePlayAddress();
      };
      actionCol.append(playBtn);
    } else if (isCatchup) {
      const replayLink = document.createElement('a');
      replayLink.className = 'button secondary btn-catchup';
      replayLink.target = '_blank';
      replayLink.rel = 'noopener';
      const playRef = c.operator_id || c.key || c.id;
      const playUrl = `/api/play?id=${encodeURIComponent(playRef)}&mode=http&start=${p.start}&end=${p.end}`;
      replayLink.href = playUrl;
      replayLink.innerHTML = `
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><polygon points="11 19 2 12 11 5 11 19"/><polygon points="22 19 13 12 22 5 22 19"/></svg>
        回看
      `;

      const copyBtn = document.createElement('button');
      copyBtn.type = 'button';
      copyBtn.className = 'secondary btn-copy-url';
      copyBtn.title = '复制回看地址';
      copyBtn.innerHTML = `
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
      `;
      copyBtn.onclick = () => {
        const full = new URL(playUrl, location.origin).href;
        copyText(full, '回看地址已复制到剪贴板');
      };

      actionCol.append(replayLink, copyBtn);
    } else {
      const label = document.createElement('span');
      label.className = 'muted';
      label.style.fontSize = '12px';
      label.textContent = isUpcoming ? '待播出' : '已结束';
      actionCol.append(label);
    }

    card.append(timeCol, infoCol, actionCol);
    container.append(card);
  }
}

async function loadEPG() {
  const version = ++epgRequestVersion, id = $('epg-channel').value;
  updateEPGHero();
  renderDateRibbon();

  if (!id) {
    $('epg-message').textContent = '请先同步运营商频道；扫描频道暂无节目单关联';
    $('epg-programmes').replaceChildren();
    return;
  }

  $('epg-message').textContent = '正在加载节目单…';
  const value = await api('epg/programmes?id=' + encodeURIComponent(id) + '&date=' + encodeURIComponent($('epg-date').value));
  if (version !== epgRequestVersion) return;

  currentEPGProgrammes = value.programmes || [];
  const count = currentEPGProgrammes.length;
  $('epg-message').textContent = count ? `当日共 ${count} 个节目` : '该日期暂无节目单，请点击“立即同步”或选择其他日期';

  renderEPGList();

  // If today is viewed, auto-scroll to the currently live program
  if ($('epg-date').value === getShanghaiTodayStr()) {
    setTimeout(() => {
      const liveItem = document.querySelector('.epg-item.is-live');
      if (liveItem) {
        liveItem.scrollIntoView({ behavior: 'smooth', block: 'center' });
      }
    }, 120);
  }
}

async function showEPG(c) {
  tab('epg');
  $('epg-channel').value = c.key || c.id;
  updateEPGHero();
  await epgStatus();
  await loadEPG();
}

// EPG Event Listeners
$('save-epg').onclick = () => busy($('save-epg'), async () => {
  await saveSettings();
  toast('EPG 设置已保存');
});

$('sync-epg').onclick = () => busy($('sync-epg'), async () => {
  await saveSettings();
  await api('epg/refresh', 'POST');
  await epgStatus();
  toast('节目单同步已开始');
});

$('epg-load').onclick = () => busy($('epg-load'), loadEPG);

for (const id of ['epg-channel', 'epg-date']) {
  $(id).onchange = () => loadEPG().catch(e => { $('epg-message').textContent = e.message; });
}

$('epg-search').oninput = () => renderEPGList();

document.querySelectorAll('.filter-tab').forEach(tabBtn => {
  tabBtn.addEventListener('click', () => {
    document.querySelectorAll('.filter-tab').forEach(b => b.classList.remove('active'));
    tabBtn.classList.add('active');
    epgCurrentFilter = tabBtn.dataset.filter;
    renderEPGList();
  });
});

document.querySelector('[data-tab="epg"]').addEventListener('click', () => {
  epgStatus().then(loadEPG).catch(e => toast(e.message, true));
});

// Periodic live update timer for active progress bar
if (epgLiveTimer) clearInterval(epgLiveTimer);
epgLiveTimer = setInterval(() => {
  if (!$('tab-epg').hidden && $('epg-date').value === getShanghaiTodayStr() && currentEPGProgrammes.length) {
    renderEPGList();
  }
}, 45000);

let currentLogoSources = ['https://github.com/sggc/SDU-IPTV-PRO/tree/main/logo'];
let logoSourcesStatusData = {};

function renderLogoSourcesList() {
  const container = $('logo-sources-list');
  if (!container) return;
  container.innerHTML = '';

  if (!currentLogoSources || currentLogoSources.length === 0) {
    currentLogoSources = ['https://github.com/sggc/SDU-IPTV-PRO/tree/main/logo'];
  }

  currentLogoSources.forEach((url, idx) => {
    const item = document.createElement('div');
    item.className = 'logo-source-item';

    const indexSpan = document.createElement('span');
    indexSpan.className = 'source-index';
    indexSpan.textContent = `#${idx + 1}`;

    const urlSpan = document.createElement('span');
    urlSpan.className = 'source-url';
    urlSpan.textContent = url;

    const badge = document.createElement('span');
    badge.className = 'source-badge';
    const info = logoSourcesStatusData?.sources?.find(s => s.url === url);
    if (info) {
      badge.textContent = `已索引 ${info.count || 0} 个${info.dark_count ? ' (深色 ' + info.dark_count + ')' : ''}`;
    } else {
      badge.textContent = '待更新索引';
    }

    const actions = document.createElement('div');
    actions.className = 'source-actions';

    const upBtn = document.createElement('button');
    upBtn.type = 'button';
    upBtn.className = 'secondary btn-group-nav';
    upBtn.textContent = '↑';
    upBtn.title = '提高优先级（向上移动）';
    upBtn.disabled = idx === 0;
    upBtn.onclick = () => {
      const temp = currentLogoSources[idx - 1];
      currentLogoSources[idx - 1] = currentLogoSources[idx];
      currentLogoSources[idx] = temp;
      renderLogoSourcesList();
    };

    const downBtn = document.createElement('button');
    downBtn.type = 'button';
    downBtn.className = 'secondary btn-group-nav';
    downBtn.textContent = '↓';
    downBtn.title = '降低优先级（向下移动）';
    downBtn.disabled = idx === currentLogoSources.length - 1;
    downBtn.onclick = () => {
      const temp = currentLogoSources[idx + 1];
      currentLogoSources[idx + 1] = currentLogoSources[idx];
      currentLogoSources[idx] = temp;
      renderLogoSourcesList();
    };

    const delBtn = document.createElement('button');
    delBtn.type = 'button';
    delBtn.className = 'secondary btn-group-nav';
    delBtn.textContent = '✕';
    delBtn.title = '移除此台标源';
    delBtn.onclick = () => {
      if (currentLogoSources.length <= 1) {
        toast('至少保留一个台标源', true);
        return;
      }
      currentLogoSources.splice(idx, 1);
      renderLogoSourcesList();
    };

    actions.append(upBtn, downBtn, delBtn);
    item.append(indexSpan, urlSpan, badge, actions);
    container.append(item);
  });
}

if ($('add-logo-source-btn')) {
  $('add-logo-source-btn').onclick = () => {
    const input = $('new-logo-source-input');
    const val = input.value.trim();
    if (!val) return;
    if (!/^https?:\/\//i.test(val)) {
      toast('请输入有效的 HTTP(S) 地址', true);
      return;
    }
    if (!currentLogoSources.includes(val)) {
      currentLogoSources.push(val);
      input.value = '';
      renderLogoSourcesList();
    } else {
      toast('该台标源已存在', true);
    }
  };
}

async function logoSourcesStatus() {
  const result = await api('logos/status');
  logoSourcesStatusData = result;
  renderLogoSourcesList();
  $('logos-status').textContent = `已配置 ${result.sources?.length || 0} 个源 · 共计索引 ${result.total_count || 0} 个台标`;
}

if ($('save-logos')) {
  $('save-logos').onclick = () => busy($('save-logos'), async () => {
    await saveSettings();
    await loadChannels();
    await logoSourcesStatus();
    toast('台标源已保存；更换来源后请更新台标目录');
  });
}

if ($('refresh-logos')) {
  $('refresh-logos').onclick = () => busy($('refresh-logos'), async () => {
    await saveSettings();
    $('logos-status').textContent = '正在更新台标目录…';
    const r = await api('logos/refresh', 'POST');
    logoSourcesStatusData = r;
    renderLogoSourcesList();
    await loadChannels();
    $('logos-status').textContent = `共计 ${r.total_count || 0} 个台标${r.errors?.length ? ' · 遇到问题：' + r.errors.join('；') : ' · 已更新完成'}`;
  });
}

if ($('reset-logos')) {
  $('reset-logos').onclick = () => {
    currentLogoSources = ['https://github.com/sggc/SDU-IPTV-PRO/tree/main/logo'];
    renderLogoSourcesList();
    toast('已恢复默认台标源，请点击「保存台标源」确认保存');
  };
}

document.querySelector('[data-tab="ai"]').addEventListener('click', () => logoSourcesStatus().catch(e => toast(e.message, true)));
if (document.querySelector('[data-tab="account"]')) document.querySelector('[data-tab="account"]').addEventListener('click', () => loadMappings());

// Initialize
init().catch(() => {});

let savedMappings = [];

async function loadMappings() {
  const tbody = $('mapping-rows');
  const empty = $('mapping-empty');
  const badge = $('mapping-badge');
  try {
    savedMappings = (await api('mappings')) || [];
    if (badge) {
      badge.textContent = savedMappings.length;
      badge.hidden = savedMappings.length === 0;
    }
    renderMappings();
  } catch (e) {
    console.error('Failed to load mappings:', e);
    if (badge) badge.hidden = true;
  }
}

function renderMappings() {
  const tbody = $('mapping-rows');
  const empty = $('mapping-empty');
  if (!tbody) return;
  tbody.replaceChildren();
  if (savedMappings.length === 0) {
    if (empty) empty.hidden = false;
    return;
  }
  if (empty) empty.hidden = true;

  savedMappings.forEach(m => {
    const tr = document.createElement('tr');

    const tdKw = document.createElement('td');
    const kwStrong = document.createElement('strong');
    kwStrong.textContent = m.keyword;
    tdKw.appendChild(kwStrong);

    const tdTarget = document.createElement('td');
    tdTarget.textContent = m.target_name;

    const tdID = document.createElement('td');
    const idBadge = document.createElement('span');
    idBadge.className = 'badge';
    idBadge.style.fontFamily = 'monospace';
    idBadge.textContent = m.target_id;
    tdID.appendChild(idBadge);

    const tdOp = document.createElement('td');
    tdOp.style.fontFamily = 'monospace';
    tdOp.style.fontSize = '12px';
    tdOp.textContent = m.operator_id || '-';

    const tdGrp = document.createElement('td');
    tdGrp.textContent = m.group || '-';

    const tdAct = document.createElement('td');
    const delBtn = document.createElement('button');
    delBtn.type = 'button';
    delBtn.className = 'secondary';
    delBtn.style.cssText = 'font-size: 11px; padding: 2px 8px; line-height: 1.2; height: auto; color: var(--live-red, #dc2626);';
    delBtn.textContent = '删除';
    delBtn.onclick = async () => {
      if (!confirm(`确定删除关键词「${m.keyword}」的映射吗？`)) return;
      try {
        await api('mappings/' + encodeURIComponent(m.keyword), 'DELETE');
        toast(`已删除「${m.keyword}」映射`);
        await loadMappings();
      } catch (err) {
        toast(`删除失败: ${err.message}`, true);
      }
    };
    tdAct.appendChild(delBtn);

    tr.appendChild(tdKw);
    tr.appendChild(tdTarget);
    tr.appendChild(tdID);
    tr.appendChild(tdOp);
    tr.appendChild(tdGrp);
    tr.appendChild(tdAct);

    tbody.appendChild(tr);
  });
}

if ($('add-mapping-form')) {
  $('add-mapping-form').onsubmit = async (e) => {
    e.preventDefault();
    const form = e.target;
    const kw = form.elements.keyword.value.trim();
    const targetName = form.elements.target_name.value.trim();
    const targetID = form.elements.target_id.value.trim();
    const operatorID = form.elements.operator_id.value.trim();
    const group = form.elements.group.value.trim();
    if (!kw || !targetName || !targetID) {
      toast('请填写关键词、目标频道名称和数字台号', true);
      return;
    }
    await busy(form.querySelector('button[type="submit"]'), async () => {
      try {
        await api('mappings', 'POST', {
          keyword: kw,
          target_name: targetName,
          target_id: targetID,
          operator_id: operatorID,
          group: group,
        });
        form.reset();
        toast(`已添加关键词「${kw}」映射`);
        await loadMappings();
      } catch (err) {
        toast(`添加映射失败: ${err.message}`, true);
      }
    });
  };
}


let currentGroupOrder = ['央视频道', '卫视频道', '上海频道', '数字频道', '其它', '待识别'];

function renderGroupOrderList() {
  const container = $('group-order-list');
  const select = $('quick-add-group-select');
  if (!container) return;
  container.innerHTML = '';

  if (!currentGroupOrder || currentGroupOrder.length === 0) {
    currentGroupOrder = ['央视频道', '卫视频道', '上海频道', '数字频道', '其它', '待识别'];
  }

  currentGroupOrder.forEach((groupName, idx) => {
    const item = document.createElement('div');
    item.className = 'group-order-item';

    const left = document.createElement('div');
    left.style.display = 'flex';
    left.style.alignItems = 'center';
    left.style.gap = '8px';

    const indexSpan = document.createElement('span');
    indexSpan.className = 'group-index';
    indexSpan.textContent = `#${idx + 1}`;

    const nameSpan = document.createElement('span');
    nameSpan.className = 'group-name';
    nameSpan.textContent = groupName;

    const count = channels.filter(c => (c.group || '未分组') === groupName).length;
    const badge = document.createElement('span');
    badge.className = 'badge';
    badge.style.fontSize = '11px';
    badge.style.padding = '1px 6px';
    badge.textContent = `${count} 个频道`;

    left.append(indexSpan, nameSpan, badge);

    const actions = document.createElement('div');
    actions.className = 'group-actions';

    const upBtn = document.createElement('button');
    upBtn.type = 'button';
    upBtn.className = 'secondary btn-group-nav';
    upBtn.textContent = '↑';
    upBtn.title = '向上移动';
    upBtn.disabled = idx === 0;
    upBtn.onclick = () => {
      const temp = currentGroupOrder[idx - 1];
      currentGroupOrder[idx - 1] = currentGroupOrder[idx];
      currentGroupOrder[idx] = temp;
      renderGroupOrderList();
    };

    const downBtn = document.createElement('button');
    downBtn.type = 'button';
    downBtn.className = 'secondary btn-group-nav';
    downBtn.textContent = '↓';
    downBtn.title = '向下移动';
    downBtn.disabled = idx === currentGroupOrder.length - 1;
    downBtn.onclick = () => {
      const temp = currentGroupOrder[idx + 1];
      currentGroupOrder[idx + 1] = currentGroupOrder[idx];
      currentGroupOrder[idx] = temp;
      renderGroupOrderList();
    };

    const delBtn = document.createElement('button');
    delBtn.type = 'button';
    delBtn.className = 'secondary btn-group-nav';
    delBtn.textContent = '✕';
    delBtn.title = '从排序列表中移除';
    delBtn.onclick = () => {
      currentGroupOrder.splice(idx, 1);
      renderGroupOrderList();
    };

    actions.append(upBtn, downBtn, delBtn);
    item.append(left, actions);
    container.append(item);
  });

  if (select) {
    select.replaceChildren(new Option('快速添加现有分组…', ''));
    const existing = [...new Set(channels.map(c => c.group || '未分组'))].filter(Boolean);
    for (const g of existing) {
      if (!currentGroupOrder.includes(g)) {
        select.append(new Option(g, g));
      }
    }
  }
}

if ($('add-group-btn')) {
  $('add-group-btn').onclick = () => {
    const input = $('new-group-name');
    const val = input.value.trim();
    if (!val) return;
    if (!currentGroupOrder.includes(val)) {
      currentGroupOrder.push(val);
      input.value = '';
      renderGroupOrderList();
    } else {
      toast(`分组「${val}」已在排序列表中`, true);
    }
  };
}

if ($('quick-add-group-select')) {
  $('quick-add-group-select').onchange = () => {
    const val = $('quick-add-group-select').value;
    if (val && !currentGroupOrder.includes(val)) {
      currentGroupOrder.push(val);
      $('quick-add-group-select').value = '';
      renderGroupOrderList();
    }
  };
}

if ($('save-group-order-btn')) {
  $('save-group-order-btn').onclick = () => busy($('save-group-order-btn'), async () => {
    try {
      if (!settings) settings = (await api('settings')).settings;
      settings.group_order = currentGroupOrder;
      await api('settings', 'PUT', { settings: collectSettings() });
      fillSettings(await api('settings'));
      await loadChannels();
      toast('分组排序设置已保存，频道列表和导出播放列表已按新顺序排序');
    } catch (e) {
      toast('保存分组排序失败: ' + e.message, true);
    }
  });
}

if ($('reset-group-order-btn')) {
  $('reset-group-order-btn').onclick = () => {
    currentGroupOrder = ['央视频道', '卫视频道', '上海频道', '数字频道', '其它', '待识别'];
    renderGroupOrderList();
    toast('已恢复默认分组排序，请点击「保存分组排序」确认保存');
  };
}

if ($('probe-channel-btn')) {
  $('probe-channel-btn').onclick = () => busy($('probe-channel-btn'), async () => {
    const form = $('channel-form');
    const ref = editing?.key || editing?.id || form.elements.id.value;
    if (!ref) {
      toast('请先保存频道或选择已知频道再探测', true);
      return;
    }
    try {
      toast('正在从实时流探测分辨率…');
      const res = await api('channels/' + encodeURIComponent(ref) + '/probe', 'POST');
      if (res && res.resolution) {
        form.elements.resolution.value = res.resolution;
        if (editing) editing.resolution = res.resolution;
        toast(`探测成功: ${res.resolution} (${getResolutionLabel(res.resolution)})`);
      } else {
        toast('未能探测到有效分辨率', true);
      }
    } catch (e) {
      toast('探测流分辨率失败: ' + e.message, true);
    }
  });
}


/* ==========================================================================
   In-Group Channel Sorting & Batch Probing
   ========================================================================== */

let currentGroupChannelOrders = {};
let selectedSortGroup = '';

function getGroupChannels(groupName) {
  return channels.filter(c => (c.group || '未分组') === groupName);
}

function updateGroupChannelSelect() {
  const select = $('group-channel-select');
  if (!select) return;
  const allGroups = [...new Set(channels.map(c => c.group || '未分组'))].filter(Boolean);
  allGroups.sort((a, b) => {
    const iA = currentGroupOrder.indexOf(a);
    const iB = currentGroupOrder.indexOf(b);
    if (iA !== -1 && iB !== -1) return iA - iB;
    if (iA !== -1) return -1;
    if (iB !== -1) return 1;
    return a.localeCompare(b, 'zh-CN');
  });

  const currentVal = select.value;
  select.replaceChildren();
  allGroups.forEach(g => {
    const count = getGroupChannels(g).length;
    select.append(new Option(`${g} (${count}个频道)`, g));
  });

  if (allGroups.includes(currentVal)) {
    select.value = currentVal;
  } else if (allGroups.length > 0) {
    select.value = allGroups[0];
  }
  selectedSortGroup = select.value;
  renderInGroupChannelList();
}

function getCurrentInGroupKeys() {
  const container = $('in-group-channel-list');
  if (!container) return [];
  return Array.from(container.querySelectorAll('.in-group-channel-item')).map(el => el.dataset.key);
}

function renderInGroupChannelList() {
  const container = $('in-group-channel-list');
  if (!container) return;
  container.innerHTML = '';
  if (!selectedSortGroup) return;

  const groupChans = getGroupChannels(selectedSortGroup);
  if (groupChans.length === 0) {
    container.innerHTML = '<p class="muted" style="text-align:center; padding:16px;">此分组暂无频道</p>';
    return;
  }

  let ordered = [...groupChans];
  const customKeys = currentGroupChannelOrders[selectedSortGroup] || [];
  if (customKeys.length > 0) {
    ordered.sort((a, b) => {
      const keyA = a.key || a.id;
      const keyB = b.key || b.id;
      const iA = customKeys.indexOf(keyA);
      const iB = customKeys.indexOf(keyB);
      if (iA !== -1 && iB !== -1) return iA - iB;
      if (iA !== -1) return -1;
      if (iB !== -1) return 1;
      return 0;
    });
  }

  ordered.forEach((c, idx) => {
    const item = document.createElement('div');
    item.className = 'in-group-channel-item';
    item.dataset.key = c.key || c.id;

    const seq = document.createElement('span');
    seq.className = 'in-group-seq';
    seq.textContent = idx + 1;

    const logoBox = document.createElement('div');
    logoBox.className = 'in-group-logo';
    if (c.logo) {
      const img = document.createElement('img');
      img.src = c.logo;
      img.alt = '';
      img.onerror = () => { logoBox.textContent = '📺'; };
      logoBox.append(img);
    } else {
      logoBox.textContent = '📺';
    }

    const nameSpan = document.createElement('span');
    nameSpan.className = 'in-group-name';
    nameSpan.textContent = c.name;

    const meta = document.createElement('div');
    meta.className = 'in-group-meta';

    if (c.id && !isInvalidID(c.id)) {
      const idBadge = document.createElement('span');
      idBadge.className = 'badge';
      idBadge.style.fontSize = '11px';
      idBadge.textContent = `台号: ${c.id}`;
      meta.append(idBadge);
    }

    if (c.resolution) {
      const resBadge = document.createElement('span');
      resBadge.className = 'badge badge-res';
      resBadge.style.fontSize = '11px';
      resBadge.textContent = c.resolution;
      meta.append(resBadge);
    }

    const actions = document.createElement('div');
    actions.className = 'in-group-actions';

    const topBtn = document.createElement('button');
    topBtn.type = 'button';
    topBtn.className = 'secondary';
    topBtn.textContent = '⤒';
    topBtn.title = '移到置顶';
    topBtn.disabled = idx === 0;
    topBtn.onclick = () => {
      const currentList = getCurrentInGroupKeys();
      const itemKey = currentList.splice(idx, 1)[0];
      currentList.unshift(itemKey);
      currentGroupChannelOrders[selectedSortGroup] = currentList;
      renderInGroupChannelList();
    };

    const upBtn = document.createElement('button');
    upBtn.type = 'button';
    upBtn.className = 'secondary';
    upBtn.textContent = '↑';
    upBtn.title = '向上移动';
    upBtn.disabled = idx === 0;
    upBtn.onclick = () => {
      const currentList = getCurrentInGroupKeys();
      const temp = currentList[idx - 1];
      currentList[idx - 1] = currentList[idx];
      currentList[idx] = temp;
      currentGroupChannelOrders[selectedSortGroup] = currentList;
      renderInGroupChannelList();
    };

    const downBtn = document.createElement('button');
    downBtn.type = 'button';
    downBtn.className = 'secondary';
    downBtn.textContent = '↓';
    downBtn.title = '向下移动';
    downBtn.disabled = idx === ordered.length - 1;
    downBtn.onclick = () => {
      const currentList = getCurrentInGroupKeys();
      const temp = currentList[idx + 1];
      currentList[idx + 1] = currentList[idx];
      currentList[idx] = temp;
      currentGroupChannelOrders[selectedSortGroup] = currentList;
      renderInGroupChannelList();
    };

    const btmBtn = document.createElement('button');
    btmBtn.type = 'button';
    btmBtn.className = 'secondary';
    btmBtn.textContent = '⤓';
    btmBtn.title = '移到末尾';
    btmBtn.disabled = idx === ordered.length - 1;
    btmBtn.onclick = () => {
      const currentList = getCurrentInGroupKeys();
      const itemKey = currentList.splice(idx, 1)[0];
      currentList.push(itemKey);
      currentGroupChannelOrders[selectedSortGroup] = currentList;
      renderInGroupChannelList();
    };

    actions.append(topBtn, upBtn, downBtn, btmBtn);
    item.append(seq, logoBox, nameSpan, meta, actions);
    container.append(item);
  });
}

if ($('group-channel-select')) {
  $('group-channel-select').onchange = () => {
    selectedSortGroup = $('group-channel-select').value;
    renderInGroupChannelList();
  };
}

if ($('sort-in-group-by-id')) {
  $('sort-in-group-by-id').onclick = () => {
    const groupChans = getGroupChannels(selectedSortGroup);
    groupChans.sort((a, b) => {
      const idA = Number(a.id), idB = Number(b.id);
      if (!isNaN(idA) && !isNaN(idB) && idA > 0 && idB > 0) return idA - idB;
      if (!isNaN(idA) && idA > 0) return -1;
      if (!isNaN(idB) && idB > 0) return 1;
      return a.name.localeCompare(b.name, 'zh-CN', { numeric: true });
    });
    currentGroupChannelOrders[selectedSortGroup] = groupChans.map(c => c.key || c.id);
    renderInGroupChannelList();
    toast(`已按台号顺序排列「${selectedSortGroup}」频道`);
  };
}

if ($('sort-in-group-by-name')) {
  $('sort-in-group-by-name').onclick = () => {
    const groupChans = getGroupChannels(selectedSortGroup);
    groupChans.sort((a, b) => a.name.localeCompare(b.name, 'zh-CN', { numeric: true }));
    currentGroupChannelOrders[selectedSortGroup] = groupChans.map(c => c.key || c.id);
    renderInGroupChannelList();
    toast(`已按名称字典序排列「${selectedSortGroup}」频道`);
  };
}

if ($('sort-in-group-by-res')) {
  $('sort-in-group-by-res').onclick = () => {
    const resScore = res => {
      if (!res) return 0;
      const m = res.match(/(\d+)x(\d+)/i);
      if (m) return Number(m[1]) * Number(m[2]);
      if (/4k/i.test(res)) return 3840 * 2160;
      if (/1080/i.test(res)) return 1920 * 1080;
      if (/720/i.test(res)) return 1280 * 720;
      return 1;
    };
    const groupChans = getGroupChannels(selectedSortGroup);
    groupChans.sort((a, b) => {
      const sA = resScore(a.resolution);
      const sB = resScore(b.resolution);
      if (sA !== sB) return sB - sA;
      return a.name.localeCompare(b.name, 'zh-CN', { numeric: true });
    });
    currentGroupChannelOrders[selectedSortGroup] = groupChans.map(c => c.key || c.id);
    renderInGroupChannelList();
    toast(`已按分辨率规格优先排列「${selectedSortGroup}」频道`);
  };
}

if ($('save-in-group-order-btn')) {
  $('save-in-group-order-btn').onclick = () => busy($('save-in-group-order-btn'), async () => {
    try {
      const keys = getCurrentInGroupKeys();
      if (!currentGroupChannelOrders) currentGroupChannelOrders = {};
      currentGroupChannelOrders[selectedSortGroup] = keys;
      if (!settings) settings = (await api('settings')).settings;
      settings.group_channel_order = currentGroupChannelOrders;
      await api('settings', 'PUT', { settings: collectSettings() });
      fillSettings(await api('settings'));
      await loadChannels();
      toast(`「${selectedSortGroup}」组内频道排序已保存`);
    } catch (e) {
      toast('保存组内排序失败: ' + e.message, true);
    }
  });
}

if ($('reset-in-group-order-btn')) {
  $('reset-in-group-order-btn').onclick = () => {
    if (currentGroupChannelOrders && currentGroupChannelOrders[selectedSortGroup]) {
      delete currentGroupChannelOrders[selectedSortGroup];
    }
    renderInGroupChannelList();
    toast(`已恢复「${selectedSortGroup}」默认组内顺序，请点击「保存组内排序」确认保存`);
  };
}

/* ==========================================================================
   Batch Probe Progress & Polling
   ========================================================================== */

let probePollTimer = null;

async function checkProbeStatus() {
  try {
    const status = await api('probe/status');
    const card = $('probe-progress-card');
    if (!card) return;

    if (status.state === 'running') {
      card.hidden = false;
      $('probe-progress-title').textContent = `正在一键探测分辨率… (已获取 ${status.found} 个规格)`;
      $('probe-progress-count').textContent = `${status.done} / ${status.total}`;
      const pct = status.total > 0 ? Math.round((status.done / status.total) * 100) : 0;
      $('probe-progress-fill').style.width = pct + '%';
      if (!probePollTimer) {
        probePollTimer = setInterval(checkProbeStatus, 1000);
      }
    } else {
      if (probePollTimer) {
        clearInterval(probePollTimer);
        probePollTimer = null;
      }
      if (status.state === 'completed' || status.state === 'stopped') {
        card.hidden = true;
        if (status.state === 'completed') {
          toast(`一键探测完成：共完成 ${status.done} 个频道探测，获取到 ${status.found} 个视频分辨率规格`);
          await loadChannels();
        } else if (status.state === 'stopped') {
          toast(`一键探测已停止：完成 ${status.done}/${status.total} 个频道`);
          await loadChannels();
        }
      } else {
        card.hidden = true;
      }
    }
  } catch (e) {
    if (probePollTimer) {
      clearInterval(probePollTimer);
      probePollTimer = null;
    }
  }
}

if ($('batch-probe-btn')) {
  $('batch-probe-btn').onclick = async () => {
    const missingCount = channels.filter(c => c.enabled && (c.url || c.unicast_url) && !c.resolution).length;
    const totalEnabled = channels.filter(c => c.enabled && (c.url || c.unicast_url)).length;
    if (totalEnabled === 0) {
      toast('没有已启用且可播放的频道', true);
      return;
    }

    let onlyMissing = false;
    if (missingCount > 0 && missingCount < totalEnabled) {
      if (confirm(`发现 ${missingCount} 个频道尚未探测分辨率（共 ${totalEnabled} 个启用频道）。\n\n点击【确定】仅探测这 ${missingCount} 个未获取分辨率的频道；\n点击【取消】重新探测全部 ${totalEnabled} 个频道。`)) {
        onlyMissing = true;
      }
    } else if (!confirm(`将对 ${totalEnabled} 个启用频道进行一键流分辨率探测，是否开始？`)) {
      return;
    }

    try {
      await api(`probe/start?only_missing=${onlyMissing}`, 'POST');
      toast('一键探测任务已在后台启动');
      await checkProbeStatus();
    } catch (e) {
      toast('启动探测失败: ' + e.message, true);
    }
  };
}

if ($('stop-probe-btn')) {
  $('stop-probe-btn').onclick = async () => {
    try {
      await api('probe/stop', 'POST');
      toast('正在停止探测…');
      await checkProbeStatus();
    } catch (e) {
      toast('停止探测失败: ' + e.message, true);
    }
  };
}
