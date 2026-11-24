'use strict';

const $ = (id) => document.getElementById(id);

const DEFAULT_GROUP_ORDER = ['4K', '央视', '卫视', '高清', '本地', '少儿', '标清', '其它', '待识别'];
const DEFAULT_LOGO_SOURCES = ['https://tvlogo.netioe.com'];
const CATCHUP_PRESETS = [
  '&utc=${start}&lutc=${end}',
  '&start={utc}&end={utcend}',
  '&utc={utc}&lutc={utcend}',
  '&start=${timestamp}&duration=${duration}',
];
const PAGE_SIZE = 30;
const JOB_STATES = {
  idle: '待命',
  running: '运行中',
  stopping: '正在停止',
  completed: '已完成',
  cancelled: '已停止',
  failed: '失败',
};

let settings = null;
let channels = [];
let page = 0;
let editing = null;
let playing = null;
let toastTimer = null;
let statusPollTimer = null;
let epgPollTimer = null;
let probePollTimer = null;
let epgLiveTimer = null;
let lastScanState = 'idle';
let lastRefreshState = 'idle';
let lastEPGState = '';
let lastProbeState = 'idle';
let epgRequestVersion = 0;
let logoMatchVersion = 0;
let pendingLogoMatch = null;
let currentEPGProgrammes = [];
let epgCurrentFilter = 'all';
let currentLogoSources = [...DEFAULT_LOGO_SOURCES];
let logoSourcesStatusData = {};
let savedMappings = [];
let currentGroupOrder = [...DEFAULT_GROUP_ORDER];
let currentGroupChannelOrders = {};
let selectedSortGroup = '';
let currentCandidates = [];
let currentMatchQuery = '';
let tableUploadTargetChannel = null;
let snapshotURL = null;

/* ==========================================================================
   Helpers & API
   ========================================================================== */

function toast(message, error = false) {
  const el = $('toast');
  if (!el) return;
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
      input.className = 'visually-hidden-input';
      document.body.appendChild(input);
      input.focus();
      input.select();
      const ok = document.execCommand('copy');
      input.remove();
      if (!ok) throw new Error('execCommand copy failed');
    }
    toast(successMsg);
    return true;
  } catch (_) {
    toast('复制失败，请手动复制：' + text, true);
    return false;
  }
}

async function api(path, method = 'GET', body) {
  const token = sessionStorage.getItem('iptv-token') || '';
  const isFormData = typeof FormData !== 'undefined' && body instanceof FormData;
  const headers = token ? { Authorization: 'Bearer ' + token } : {};
  if (!isFormData) {
    headers['Content-Type'] = 'application/json';
  }
  const response = await fetch('/api/panel/' + path, {
    method,
    headers,
    ...(body !== undefined ? { body: isFormData ? body : JSON.stringify(body) } : {}),
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
  if (!button) return action();
  button.disabled = true;
  try {
    await action();
  } catch (e) {
    toast(e.message, true);
  } finally {
    button.disabled = false;
  }
}

function getResolutionLabel(res) {
  if (!res) return '';
  const m = String(res).match(/^(\d{3,5})[xX*](\d{3,5})$/);
  if (!m) return res;
  const w = parseInt(m[1], 10);
  const h = parseInt(m[2], 10);
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

function isLocalLogoPath(logo) {
  return Boolean(logo && (logo.startsWith('/logos/') || logo.startsWith('/api/panel/logos/local/')));
}

function isInvalidID(id) {
  return !id || id === '0' || id === '-1' || String(id).startsWith('scan-') || id === 'none' || id === 'unknown' || isNaN(Number(id)) || Number(id) <= 0;
}

function isValidID(id) {
  return !isInvalidID(id);
}

function isUnknownChannel(c) {
  return c.group === '待识别' || c.name.startsWith('未知频道') || isInvalidID(c.id);
}

function toIgmpUrl(rawUrl) {
  if (!rawUrl) return '';
  return rawUrl.replace(/^(?:rtp|udp):\/\//i, 'igmp://');
}

function cleanM3UValue(s) {
  return String(s || '').replace(/[\r\n]/g, ' ').replace(/"/g, "'");
}

function getCCTVRank(name) {
  const s = String(name || '').trim();
  if (!s) return null;
  const upper = s.toUpperCase();
  const isCCTVScope = upper.includes('CCTV') || s.includes('中央') || s.includes('央视');
  if (isCCTVScope && upper.includes('4K')) return 18.0;
  if (isCCTVScope && upper.includes('8K')) return 19.0;
  if (/cctv[-_\s]*5\s*(?:\+|plus)/i.test(s) || s.includes('中央五+') || s.includes('中央5+')) return 5.5;
  const mNum = s.match(/cctv[-_\s]*(\d+)(?:\+?|\s*plus)?/i);
  if (mNum) {
    const val = parseInt(mNum[1], 10);
    if (val >= 1 && val <= 17) return (upper.includes('+') || upper.includes('PLUS')) ? val + 0.5 : val;
  }
  const mCentral = s.match(/中央(?:电视台)?[-_\s]*(\d+)/);
  if (mCentral) {
    const val = parseInt(mCentral[1], 10);
    if (val >= 1 && val <= 17) return s.includes('+') ? val + 0.5 : val;
  }
  if (isCCTVScope) return 100.0;
  return null;
}

/* ==========================================================================
   Navigation & Mobile Sidebar
   ========================================================================== */

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

$('mobile-menu-btn')?.addEventListener('click', openMobileSidebar);
$('sidebar-close-btn')?.addEventListener('click', closeMobileSidebar);
$('sidebar-backdrop')?.addEventListener('click', closeMobileSidebar);
if ($('mobile-reload-btn') && $('reload')) {
  $('mobile-reload-btn').addEventListener('click', () => $('reload').click());
}
document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape' && document.body.classList.contains('drawer-opened')) {
    closeMobileSidebar();
  }
});

function tab(name) {
  closeMobileSidebar();
  document.querySelectorAll('.tab').forEach(el => {
    el.hidden = el.id !== 'tab-' + name;
  });
  document.querySelectorAll('[data-tab]').forEach(el => {
    const active = el.dataset.tab === name;
    el.classList.toggle('active', active);
    if (active) el.setAttribute('aria-current', 'page');
    else el.removeAttribute('aria-current');
  });
  $('page-title').textContent = {
    channels: '频道管理',
    account: 'IPTV配置',
    scan: '组播扫描',
    ai: '智能识别',
    epg: 'EPG 节目单',
  }[name] || '控制台';
  if (name === 'channels') {
    const activeSubBtn = document.querySelector('.channel-subnav-btn.active');
    switchChannelView(activeSubBtn ? activeSubBtn.dataset.channelView : 'list');
  }
}

function switchChannelView(view) {
  document.querySelectorAll('.channel-subnav-btn').forEach(btn => {
    const active = btn.dataset.channelView === view;
    btn.classList.toggle('active', active);
    btn.setAttribute('aria-selected', String(active));
  });
  if ($('channel-view-list')) $('channel-view-list').hidden = (view !== 'list');
  if ($('channel-view-sorting')) $('channel-view-sorting').hidden = (view !== 'sorting');
  if ($('channel-view-mapping')) $('channel-view-mapping').hidden = (view !== 'mapping');

  if (view === 'sorting') {
    renderGroupOrderList();
    updateGroupChannelSelect();
  } else if (view === 'mapping') {
    loadMappings();
  }
}

document.querySelectorAll('[data-tab]').forEach(el => {
  el.addEventListener('click', () => tab(el.dataset.tab));
});
document.querySelectorAll('[data-channel-view]').forEach(btn => {
  btn.addEventListener('click', () => switchChannelView(btn.dataset.channelView));
});
document.querySelectorAll('[data-close]').forEach(el => {
  el.addEventListener('click', () => $(el.dataset.close)?.close());
});

// Interactive top stat cards act as quick channel filters
document.querySelectorAll('[data-stat-filter]').forEach(card => {
  card.addEventListener('click', () => {
    const filterVal = card.dataset.statFilter;
    $('channel-filter').value = filterVal;
    page = 0;
    tab('channels');
    switchChannelView('list');
    renderChannels();
  });
});

function syncStatCardActive() {
  const filterVal = $('channel-filter')?.value || 'all';
  const onChannelsList = !$('tab-channels')?.hidden && !$('channel-view-list')?.hidden;
  document.querySelectorAll('[data-stat-filter]').forEach(card => {
    card.classList.toggle('is-active', onChannelsList && card.dataset.statFilter === filterVal);
  });
}

/* ==========================================================================
   Settings, FCC & Catchup Template
   ========================================================================== */

function populateCatchupTemplateSelect(currentTpl) {
  const select = $('catchup-template-select');
  const input = $('forward-catchup-template-input');
  if (!select) return;
  const val = (currentTpl || '').trim() || CATCHUP_PRESETS[0];

  if (CATCHUP_PRESETS.includes(val)) {
    select.value = val;
    if (input) { input.value = val; input.hidden = true; }
    select.classList.remove('is-custom');
  } else {
    select.value = '__custom__';
    if (input) { input.value = val; input.hidden = false; }
    select.classList.add('is-custom');
  }
}

function getActiveCatchupTemplate() {
  const select = $('catchup-template-select');
  const input = $('forward-catchup-template-input');
  if (select && select.value === '__custom__') {
    return input ? input.value.trim() : '';
  }
  return select ? select.value : (input ? input.value.trim() : '');
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

function getActiveFCC() {
  const select = $('fcc-select');
  const input = $('forward-fcc-input');
  if (select && select.value === '__custom__') {
    return input ? input.value.trim() : '';
  }
  return select ? select.value : (input ? input.value.trim() : '');
}

function fillSettings(result) {
  settings = result.settings;
  document.querySelectorAll('[data-setting]').forEach(el => {
    const [a, b] = el.dataset.setting.split('.');
    const v = settings[a]?.[b];
    if (el.type === 'checkbox') el.checked = Boolean(v);
    else el.value = Array.isArray(v) ? v.join('\n') : (v ?? '');
  });
  $('clear-key').checked = false;
  $('key-status').textContent = result.has_api_key ? '已保存 key（不会回显）' : '尚未配置 key；本地无认证接口可留空';
  populateFCCSelect(settings.forward?.fcc_list, settings.forward?.fcc);
  populateCatchupTemplateSelect(settings.forward?.catchup_template);
  if (settings.logos) {
    currentLogoSources = settings.logos.sources || (settings.logos.primary ? [settings.logos.primary] : [...DEFAULT_LOGO_SOURCES]);
  }
  currentGroupOrder = settings.group_order?.length ? [...settings.group_order] : [...DEFAULT_GROUP_ORDER];
  currentGroupChannelOrders = settings.group_channel_order ? structuredClone(settings.group_channel_order) : {};
  renderGroupOrderList();
  renderLogoSourcesList();
  updateGroupChannelSelect();
  updatePreview();
}

function collectSettings() {
  const value = structuredClone(settings);
  document.querySelectorAll('[data-setting]').forEach(el => {
    const [a, b] = el.dataset.setting.split('.');
    value[a][b] = el.type === 'checkbox'
      ? el.checked
      : el.type === 'number'
        ? Number(el.value)
        : el.tagName === 'TEXTAREA'
          ? el.value.split(/\s+/).filter(Boolean)
          : el.value.trim();
  });
  const activeFCC = getActiveFCC();
  value.forward.fcc = activeFCC;
  const activeTpl = getActiveCatchupTemplate();
  value.forward.catchup_template = activeTpl || CATCHUP_PRESETS[0];
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
  const tpl = v.catchup_template || CATCHUP_PRESETS[0];
  const normTpl = tpl.startsWith('&') || tpl.startsWith('?')
    ? (tpl.startsWith('?') ? '&' + tpl.slice(1) : tpl)
    : '&' + tpl;
  $('url-preview').textContent = `直播: http://${v.address || 'IP:端口'}/${v.protocol}/233.18.204.1:5140${fccStr}\n回看: http://${location.host || '当前服务'}/api/play?id=1&mode=unicast${normTpl}`;
}

document.querySelectorAll('[data-setting^="forward."]').forEach(el => {
  el.addEventListener('input', updatePreview);
});

$('fcc-select')?.addEventListener('change', () => {
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
});
$('forward-fcc-input')?.addEventListener('input', updatePreview);

$('catchup-template-select')?.addEventListener('change', () => {
  const select = $('catchup-template-select');
  const input = $('forward-catchup-template-input');
  if (select.value === '__custom__') {
    select.classList.add('is-custom');
    if (input) {
      input.hidden = false;
      if (!input.value.trim() || CATCHUP_PRESETS.includes(input.value.trim())) {
        input.value = CATCHUP_PRESETS[0];
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
});
$('forward-catchup-template-input')?.addEventListener('input', updatePreview);

async function saveSettings() {
  if (!settings) throw Error('请先连接面板');
  await api('settings', 'PUT', { settings: collectSettings(), clear_api_key: $('clear-key').checked });
  fillSettings(await api('settings'));
}

$('save-settings').onclick = () => busy($('save-settings'), async () => {
  await saveSettings();
  toast('设置已保存');
});
$('save-ai').onclick = () => busy($('save-ai'), async () => {
  await saveSettings();
  toast('识别设置已保存');
});

/* ==========================================================================
   Channel List & Inline Operations
   ========================================================================== */

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

async function toggleChannel(c) {
  const channelRef = c.key || c.id;
  const updated = { ...c, enabled: !c.enabled };
  if (!updated.id || !updated.id.trim()) updated.id = '0';
  await api('channels/' + encodeURIComponent(channelRef), 'PUT', updated);
  c.enabled = !c.enabled;
  renderChannels();
  fillEPGChannels();
  toast(c.enabled ? `已启用 ${c.name}` : `已停用 ${c.name}`);
}

function renderChannels() {
  $('count-total').textContent = channels.length;
  $('count-enabled').textContent = channels.filter(c => c.enabled).length;
  $('count-unknown').textContent = channels.filter(isUnknownChannel).length;
  syncStatCardActive();

  const rawQuery = $('search').value;
  if ($('search-clear')) $('search-clear').hidden = rawQuery.length === 0;
  const query = rawQuery.trim().toLowerCase();
  const filter = $('channel-filter').value;
  const selectedGroup = filter.startsWith('group:') ? filter.slice(6) : null;

  const filtered = channels.filter(c => {
    if (query) {
      const displayUrl = toIgmpUrl(c.url);
      const haystack = `${c.name} ${c.id || ''} ${c.operator_id || ''} ${c.group || ''} ${c.url || ''} ${displayUrl} ${c.resolution || ''}`.toLowerCase();
      if (!haystack.includes(query)) return false;
    }
    if (selectedGroup !== null) return (c.group || '未分组') === selectedGroup;
    if (filter === 'unknown') return isUnknownChannel(c);
    if (filter === 'enabled') return c.enabled;
    if (filter === 'disabled') return !c.enabled;
    return true;
  });

  const pages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  page = Math.min(Math.max(0, page), pages - 1);

  const tbody = $('channel-rows');
  const frag = document.createDocumentFragment();
  $('empty').hidden = filtered.length !== 0;
  $('result-count').textContent = `共 ${filtered.length} 个频道`;
  $('page-label').textContent = `${page + 1} / ${pages}`;
  $('prev-page').disabled = page === 0;
  $('next-page').disabled = page >= pages - 1;

  for (const c of filtered.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE)) {
    const row = document.createElement('tr');
    row.className = c.enabled ? 'channel-row-enabled' : 'channel-row-disabled';
    const cell = (cls) => {
      const td = document.createElement('td');
      if (cls) td.className = cls;
      row.append(td);
      return td;
    };

    const isLocal = isLocalLogoPath(c.logo);

    // 1. 名称
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

    const nameTags = document.createElement('div');
    nameTags.className = 'channel-name-tags';
    if (isLocal) {
      const localBadge = document.createElement('span');
      localBadge.className = 'badge-local';
      localBadge.textContent = '本地';
      localBadge.title = '本地台标';
      nameTags.append(localBadge);
    }
    if (c.suggestion) {
      const sugSpan = document.createElement('span');
      sugSpan.className = 'channel-sug-tag';
      sugSpan.textContent = '💡 建议';
      sugSpan.title = `识别建议: ${c.suggestion.name || ''} (${Math.round(c.suggestion.confidence * 100)}%)`;
      nameTags.append(sugSpan);
    }

    nameWrapper.append(nameTop);
    if (nameTags.childNodes.length) nameWrapper.append(nameTags);
    nameTd.append(nameWrapper);

    // 2. 频道号 (点击直接行内编辑)
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
          await api('channels/' + encodeURIComponent(channelRef), 'PUT', { ...c, id: finalId });
          c.id = finalId;
          renderChannels();
          fillEPGChannels();
          toast(`已更新频道号: ${c.name} -> ${finalId}`);
        } catch (err) {
          toast('更新失败: ' + err.message, true);
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

    // 3. 台标 (点击上传本地台标)
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

    // 4. 状态切换
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

    // 5. 分辨率
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

    // 6. 分组
    const groupTd = cell('col-group');
    const groupBadge = document.createElement('span');
    groupBadge.className = 'badge';
    groupBadge.textContent = c.group || '未分组';
    groupTd.append(groupBadge);

    // 7. 内网源地址 (点击复制)
    const urlTd = cell('col-url');
    const displayUrl = toIgmpUrl(c.url);
    const urlText = document.createElement('span');
    urlText.className = 'channel-url-mono';
    urlText.textContent = displayUrl;
    urlTd.title = '点击复制: ' + displayUrl;
    urlTd.onclick = () => copyText(displayUrl, '已复制源地址: ' + displayUrl);
    urlTd.append(urlText);

    // 8. 操作图标按钮
    const actionTd = cell('col-actions');
    const actions = document.createElement('div');
    actions.className = 'table-action-icons';

    const createActionBtn = (title, iconSvg, onClick, extraClass = '') => {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = ('action-icon-btn ' + extraClass).trim();
      btn.title = title;
      btn.setAttribute('aria-label', title);
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
    frag.append(row);
  }
  tbody.replaceChildren(frag);
}

function fillChannelFilterGroups() {
  const select = $('channel-filter');
  if (!select) return;
  const currentVal = select.value;
  select.querySelector('optgroup')?.remove();

  const groupCounts = new Map();
  for (const c of channels) {
    const g = c.group || '未分组';
    groupCounts.set(g, (groupCounts.get(g) || 0) + 1);
  }
  if (groupCounts.size === 0) return;

  const optgroup = document.createElement('optgroup');
  optgroup.label = '按分组筛选';
  for (const [g, count] of groupCounts.entries()) {
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
  renderGroupOrderList();
  updateGroupChannelSelect();
}

for (const id of ['search', 'channel-filter']) {
  $(id).addEventListener('input', () => { page = 0; renderChannels(); });
}
$('search-clear')?.addEventListener('click', () => {
  $('search').value = '';
  page = 0;
  renderChannels();
  $('search').focus();
});
$('prev-page').onclick = () => { page--; renderChannels(); };
$('next-page').onclick = () => { page++; renderChannels(); };
$('copy-m3u-btn')?.addEventListener('click', () => {
  copyText(location.origin + '/api/m3u8', '已复制 M3U 订阅地址（原生支持 TiviMate / Televizo 自动回看）');
});

/* ==========================================================================
   Channel Edit Dialog, Matching & Logos
   ========================================================================== */

function edit(c) {
  editing = c;
  pendingLogoMatch = null;
  logoMatchVersion++;

  const form = $('channel-form');
  for (const key of ['id', 'operator_id', 'name', 'group', 'logo', 'url', 'unicast_url', 'resolution']) {
    if (form.elements[key]) form.elements[key].value = c?.[key] || '';
  }
  if (form.elements.url?.value) {
    form.elements.url.value = toIgmpUrl(form.elements.url.value);
  }
  if (!form.elements.id.value) {
    form.elements.id.value = '0';
  }
  if (c?.logo_automatic) form.elements.logo.value = '';
  form.elements.enabled.checked = c ? c.enabled : true;
  form.elements.catchup_days.value = c?.catchup_days || 0;

  $('dialog-title').textContent = c ? '编辑频道' : '添加频道';
  $('channel-error').textContent = '';
  if ($('match-candidates-box')) $('match-candidates-box').hidden = true;
  if ($('logo-match-result')) $('logo-match-result').hidden = true;
  if ($('logo-match-image')) {
    $('logo-match-image').hidden = true;
    $('logo-match-image').removeAttribute('src');
  }
  $('suggestion').hidden = !c?.suggestion;
  $('accept-suggestion').hidden = !c?.suggestion?.name;

  if (c?.suggestion) {
    const v = c.suggestion;
    const formatSuggestion = (matchDesc) =>
      `建议：${v.name || '无法确定'}${matchDesc} / ${v.group || '未分组'} · 置信度 ${Math.round(v.confidence * 100)}%\n${v.reason}`;
    let matchDesc = '';
    if (v.matched_id) {
      matchDesc = ` · 匹配台号 ${v.matched_id}${v.matched_name ? ' (' + v.matched_name + ')' : ''}`;
    }
    $('suggestion').textContent = formatSuggestion(matchDesc);
    if (!v.matched_id && v.name) {
      api('channels/match?name=' + encodeURIComponent(v.name)).then(res => {
        if (res && res.matched && res.id && editing === c) {
          v.matched_id = res.id;
          v.matched_name = res.name;
          v.matched_operator_id = res.operator_id;
          v.matched_group = res.group;
          const updatedMatch = ` · 匹配台号 ${res.id}${res.name ? ' (' + res.name + ')' : ''}${v.resolution ? ' · 分辨率 ' + v.resolution : ''}`;
          $('suggestion').textContent = formatSuggestion(updatedMatch);
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
  const select = $('existing-group');
  const current = $('channel-group').value;
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
$('choose-group').onclick = () => {
  fillGroupOptions();
  $('existing-group').focus();
};

let logoPreviewTimer = null;
function updateLogoPreview() {
  clearTimeout(logoPreviewTimer);
  const img = $('logo-preview');
  const statusEl = $('logo-status');
  const custom = $('channel-logo').value.trim();
  const value = custom || (editing?.logo_automatic ? editing.logo : '');
  img.onload = img.onerror = null;
  img.hidden = true;
  img.removeAttribute('src');
  if (!value) {
    statusEl.textContent = '未设置台标（可点击此处或拖拽图片上传本地台标）';
    return;
  }
  const isLocal = isLocalLogoPath(value);
  if (!isLocal && !/^https?:\/\//i.test(value)) {
    statusEl.textContent = '请输入 HTTP(S) 图片地址或本地台标路径（也可点击上传）';
    return;
  }
  statusEl.textContent = '正在加载台标…';
  img.onload = () => {
    img.hidden = false;
    statusEl.textContent = isLocal
      ? '已加载本地台标（点击可更换）'
      : (custom ? '自定义台标（点击可更换）' : '自动匹配台标（点击可更换）');
  };
  img.onerror = () => {
    img.hidden = true;
    statusEl.textContent = '台标加载失败，请检查图片地址（点击可重新上传）';
  };
  img.src = value;
}

$('channel-logo').oninput = () => {
  clearTimeout(logoPreviewTimer);
  logoPreviewTimer = setTimeout(updateLogoPreview, 400);
};
$('channel-logo').onchange = updateLogoPreview;
$('new-channel').onclick = () => edit(null);

$('accept-suggestion').onclick = async () => {
  const form = $('channel-form');
  const v = editing?.suggestion;
  if (!v) return;
  const nameToUse = (v.name && v.name.trim().length > 0 && !/^[a-z0-9\-_+ ]+$/i.test(v.name))
    ? v.name
    : (v.matched_name || v.name);
  form.elements.name.value = nameToUse;
  form.elements.group.value = v.matched_group || v.group || '';
  if (v.matched_id) form.elements.id.value = v.matched_id;
  if (v.matched_operator_id) form.elements.operator_id.value = v.matched_operator_id;

  if (!v.matched_id && (v.name || form.elements.name.value)) {
    try {
      const match = await api('channels/match?name=' + encodeURIComponent(v.name || form.elements.name.value));
      if (match && match.matched && match.id) {
        form.elements.id.value = match.id;
        if (match.operator_id) form.elements.operator_id.value = match.operator_id;
        if (match.name && !v.name) form.elements.name.value = match.name;
        if (match.group && !form.elements.group.value) form.elements.group.value = match.group;
      }
    } catch (_) {}
  }
  if (v.resolution) form.elements.resolution.value = v.resolution;
  fillGroupOptions();
  $('match-logo').click();
};

function renderMatchCandidates(candidates) {
  const box = $('match-candidates-box');
  const list = $('match-candidates-list');
  const count = $('match-candidates-count');
  if (!box || !list) return;
  list.replaceChildren();
  count.textContent = `共 ${candidates.length} 个候选`;

  candidates.forEach((cand, idx) => {
    const item = document.createElement('label');
    item.className = 'candidate-item' + (idx === 0 ? ' is-selected' : '');

    const left = document.createElement('div');
    left.className = 'candidate-item-left';

    const radio = document.createElement('input');
    radio.type = 'radio';
    radio.name = 'match_candidate_option';
    radio.value = String(idx);
    radio.checked = idx === 0;
    radio.onchange = () => {
      list.querySelectorAll('.candidate-item').forEach(it => it.classList.remove('is-selected'));
      item.classList.add('is-selected');
    };

    const badge = document.createElement('span');
    badge.className = 'badge mono';
    badge.textContent = `台号 ${cand.id}`;

    const name = document.createElement('strong');
    name.textContent = cand.name;

    left.append(radio, badge, name);

    if (cand.operator_id) {
      const op = document.createElement('small');
      op.className = 'muted';
      op.textContent = `(${cand.operator_id})`;
      left.append(op);
    }
    if (cand.group) {
      const grp = document.createElement('small');
      grp.className = 'muted';
      grp.textContent = `[${cand.group}]`;
      left.append(grp);
    }
    if (cand.is_saved) {
      const savedBadge = document.createElement('span');
      savedBadge.className = 'badge badge-saved-mapping';
      savedBadge.textContent = '已保存映射';
      left.append(savedBadge);
    }

    const right = document.createElement('small');
    right.className = 'muted';
    right.textContent = cand.is_saved ? '关键词优先' : `匹配度 ${cand.score}%`;

    item.append(left, right);
    item.ondblclick = () => {
      radio.checked = true;
      radio.dispatchEvent(new Event('change'));
      $('apply-candidate-btn').click();
    };
    list.append(item);
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
      renderMatchCandidates(res.candidates);
    } else if (res && res.matched && res.id) {
      currentCandidates = [{
        id: res.id,
        name: res.name,
        operator_id: res.operator_id,
        group: res.group,
        score: 100,
        is_saved: res.is_saved,
      }];
      renderMatchCandidates(currentCandidates);
    } else {
      if ($('match-candidates-box')) $('match-candidates-box').hidden = true;
      toast(`未在已知频道中匹配到「${query}」的台号`, true);
    }
  });
};

$('apply-candidate-btn')?.addEventListener('click', async () => {
  const form = $('channel-form');
  const selectedRadio = document.querySelector('input[name="match_candidate_option"]:checked');
  if (!selectedRadio) {
    toast('请先选择一个候选频道', true);
    return;
  }
  const cand = currentCandidates[Number(selectedRadio.value)];
  if (!cand) return;

  form.elements.id.value = cand.id;
  if (cand.operator_id) form.elements.operator_id.value = cand.operator_id;
  if (!form.elements.name.value) form.elements.name.value = cand.name;
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
});

$('close-candidates-btn')?.addEventListener('click', () => {
  if ($('match-candidates-box')) $('match-candidates-box').hidden = true;
});

$('channel-form').onsubmit = async (event) => {
  event.preventDefault();
  const form = event.target;
  const c = Object.fromEntries(new FormData(form));
  c.operator_id = (c.operator_id || '').trim();
  c.id = (c.id || '').trim() || '0';
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
  tab('channels');
  switchChannelView('list');
  renderChannels();
};

/* ==========================================================================
   Scan, Refresh & Initialization
   ========================================================================== */

async function status() {
  clearTimeout(statusPollTimer);
  statusPollTimer = null;
  const value = await api('status');
  const scan = value.scan;
  const refresh = value.refresh;

  $('scan-state').textContent = JOB_STATES[scan.state] || scan.state;
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
    failed: '最近一次同步失败：' + refresh.error,
  }[refresh.state] || '';

  if (['running', 'stopping'].includes(scan.state) || refresh.state === 'running') {
    statusPollTimer = setTimeout(() => { status().catch(() => {}); }, 2000);
  }
  if (lastScanState === 'running' && scan.state !== 'running') await loadChannels();
  if (lastRefreshState === 'running' && refresh.state !== 'running') await loadChannels();
  lastScanState = scan.state;
  lastRefreshState = refresh.state;
}

$('start-scan').onclick = () => busy($('start-scan'), async () => {
  await saveSettings();
  await api('scan/start', 'POST');
  await status();
  toast('组播扫描已启动');
});
$('stop-scan').onclick = () => busy($('stop-scan'), async () => {
  await api('scan/stop', 'POST');
  await status();
});
$('sync').onclick = () => busy($('sync'), async () => {
  await api('refresh', 'POST');
  await status();
  toast('运营商频道同步已启动');
});

async function init() {
  try {
    fillSettings(await api('settings'));
    await loadChannels();
    await loadMappings();
    await status();
    await checkProbeStatus();
    $('login').hidden = true;
    $('workspace').hidden = false;
  } catch (e) {
    $('login').hidden = false;
    $('workspace').hidden = true;
    throw e;
  }
}

$('reload').onclick = () => busy($('reload'), async () => {
  await loadChannels();
  await loadMappings();
  await status();
  await checkProbeStatus();
  if (!$('tab-epg').hidden) await epgStatus();
  toast('数据已刷新');
});

$('login-form').onsubmit = async (e) => {
  e.preventDefault();
  sessionStorage.setItem('iptv-token', $('token').value.trim());
  $('login-error').textContent = '';
  try {
    await init();
  } catch (err) {
    $('login-error').textContent = err.message;
  }
};

/* ==========================================================================
   Playback Dialog
   ========================================================================== */

function play(c) {
  playing = c;
  $('play-title').textContent = c.name;
  $('play-mode').value = settings?.forward?.play_mode || 'multicast';
  $('play-replay').checked = false;
  const toLocalInput = (date) => new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
  $('play-start').value = toLocalInput(new Date(Date.now() - 3600000));
  $('play-end').value = toLocalInput(new Date());
  updatePlayAddress();
  $('play-dialog').showModal();
}

function updatePlayAddress() {
  if (!playing) return;
  const mode = $('play-mode').value;
  const replay = $('play-replay').checked;
  $('play-start-label').hidden = $('play-end-label').hidden = !replay;

  let address = playing.play_url || '';
  let notice = '';

  if (mode === 'http' && !playing.operator_id) {
    notice = '该频道未关联运营商 HTTP 源';
  } else if (mode === 'unicast' && !playing.unicast_url) {
    notice = '该频道未配置单播源';
  } else if (replay && !playing.catchup_days) {
    notice = '该频道未启用回看';
  } else if (mode === 'multicast' && replay) {
    notice = '回看请选择 HTTP / HLS 或单播方式';
  } else if (mode !== 'multicast') {
    const u = new URL('/api/play', location.origin);
    const playRef = playing.operator_id || playing.key || playing.id;
    u.searchParams.set('id', playRef);
    u.searchParams.set('mode', mode);
    if (replay) {
      const start = Math.floor(new Date($('play-start').value).getTime() / 1000);
      const end = Math.floor(new Date($('play-end').value).getTime() / 1000);
      if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) {
        notice = '请选择有效的开始和结束时间';
      } else {
        u.searchParams.set('start', String(start));
        u.searchParams.set('end', String(end));
      }
    }
    address = u.href;
  } else {
    try {
      const u = new URL(playing.url);
      if (['igmp:', 'rtp:', 'udp:'].includes(u.protocol) && settings?.forward?.address) {
        const f = new URL('http://' + settings.forward.address + '/' + settings.forward.protocol + '/' + u.host + u.search);
        if (!f.searchParams.get('fcc') && settings.forward.fcc) {
          f.searchParams.set('fcc', settings.forward.fcc);
        }
        address = f.href;
      } else {
        address = playing.url;
      }
    } catch (_) {
      address = playing.url || '';
    }
  }

  $('play-notice').textContent = notice || (replay ? `可回看 ${playing.catchup_days} 天；HTTP 回看按运营商节目边界结束。` : '');
  $('play-address').textContent = notice ? '' : address;

  const hasAddr = !notice && Boolean(address);
  for (const id of ['play-android', 'play-potplayer', 'play-vlc', 'play-iina', 'play-download-m3u', 'play-copy', 'play-open']) {
    const el = $(id);
    if (el) el.hidden = !hasAddr;
  }

  if (hasAddr) {
    $('play-open').href = address;

    let scheme = 'http';
    let noScheme = address;
    const schemeMatch = address.match(/^([a-z][a-z0-9+.-]*):\/\/(.*)$/i);
    if (schemeMatch) {
      scheme = schemeMatch[1].toLowerCase();
      noScheme = schemeMatch[2];
    }
    const androidIntent = `intent://${noScheme}#Intent;action=android.intent.action.VIEW;type=video/*;scheme=${scheme};end`;
    if ($('play-android')) {
      $('play-android').href = androidIntent;
      $('play-android').setAttribute('data-intent', androidIntent);
    }
    if ($('play-potplayer')) $('play-potplayer').href = `potplayer://${address}`;
    if ($('play-vlc')) $('play-vlc').href = `vlc://${address}`;
    if ($('play-iina')) $('play-iina').href = `iina://weblink?url=${encodeURIComponent(address)}`;
  } else {
    $('play-open').removeAttribute('href');
    if ($('play-android')) {
      $('play-android').removeAttribute('href');
      $('play-android').removeAttribute('data-intent');
    }
    $('play-potplayer')?.removeAttribute('href');
    $('play-vlc')?.removeAttribute('href');
    $('play-iina')?.removeAttribute('href');
  }
}

for (const id of ['play-mode', 'play-replay', 'play-start', 'play-end']) {
  $(id).onchange = updatePlayAddress;
}

$('play-android')?.addEventListener('click', (e) => {
  const intentUrl = $('play-android').getAttribute('data-intent');
  if (intentUrl) {
    window.location.href = intentUrl;
    e.preventDefault();
  }
});

$('play-copy')?.addEventListener('click', async () => {
  const addr = $('play-address').textContent;
  if (!addr) return;
  const ok = await copyText(addr, '流地址已复制到剪贴板');
  if (ok) {
    const btn = $('play-copy');
    const orig = btn.textContent;
    btn.textContent = '已复制！';
    setTimeout(() => { btn.textContent = orig; }, 1800);
  }
});

$('play-download-m3u')?.addEventListener('click', () => {
  const addr = $('play-address').textContent;
  if (!addr || !playing) return;
  const safeName = cleanM3UValue(playing.name || 'channel');
  let logoUrl = playing.logo || '';
  if (logoUrl.startsWith('/')) logoUrl = location.origin + logoUrl;
  const m3u = `#EXTM3U\n#EXTINF:-1 tvg-name="${safeName}" tvg-logo="${cleanM3UValue(logoUrl)}",${safeName}\n${addr}\n`;
  const blob = new Blob([m3u], { type: 'application/x-mpegurl;charset=utf-8' });
  const u = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = u;
  a.download = `${safeName}.m3u`;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(u);
});

/* ==========================================================================
   Logo Matching, Upload & Batch Localization
   ========================================================================== */

$('match-logo').onclick = () => busy($('match-logo'), async () => {
  const form = $('channel-form');
  const name = form.elements.name.value.trim();
  const group = form.elements.group.value.trim();
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
  const form = $('channel-form');
  const result = pendingLogoMatch;
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
  const form = $('channel-form');
  const result = pendingLogoMatch;
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
  if (isLocalLogoPath(url)) {
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
  return api('logos/upload', 'POST', fd);
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
      if (editing) editing.logo_automatic = false;
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

$('upload-logo-btn')?.addEventListener('click', () => {
  const input = $('logo-file-input');
  if (input) {
    input.value = '';
    input.click();
  }
});

$('logo-file-input')?.addEventListener('change', async (e) => {
  const file = e.target.files?.[0];
  if (file) await handleDialogLogoUpload(file);
  e.target.value = '';
});

const logoPreviewBox = $('logo-preview-box');
if (logoPreviewBox) {
  logoPreviewBox.onclick = () => {
    const input = $('logo-file-input');
    if (input) {
      input.value = '';
      input.click();
    }
  };
  for (const name of ['dragenter', 'dragover']) {
    logoPreviewBox.addEventListener(name, (e) => {
      e.preventDefault();
      e.stopPropagation();
      if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy';
      logoPreviewBox.classList.add('is-dragover');
    });
  }
  for (const name of ['dragleave', 'dragend']) {
    logoPreviewBox.addEventListener(name, (e) => {
      e.preventDefault();
      e.stopPropagation();
      logoPreviewBox.classList.remove('is-dragover');
    });
  }
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

$('table-logo-file-input')?.addEventListener('change', async (e) => {
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
      if (!updated.id || !updated.id.trim()) updated.id = '0';
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
});

$('batch-download-logos').onclick = () => busy($('batch-download-logos'), async () => {
  const targets = channels.filter(c => c.logo && /^https?:\/\//i.test(c.logo));
  if (targets.length === 0) {
    toast('当前频道列表没有使用远程 URL 的台标');
    return;
  }
  if (!confirm(`发现 ${targets.length} 个频道的台标为远程 URL，是否开始下载并保存到本地？`)) {
    return;
  }
  let success = 0;
  let failed = 0;
  for (let i = 0; i < targets.length; i++) {
    const c = targets[i];
    toast(`正在下载本地台标 (${i + 1}/${targets.length}): ${c.name}...`);
    try {
      const res = await api('logos/download', 'POST', { url: c.logo, name: c.name });
      const updated = { ...c, logo: res.url };
      const channelRef = c.key || c.original_id || c.id;
      await api('channels/' + encodeURIComponent(channelRef), 'PUT', updated);
      success++;
    } catch (_) {
      failed++;
    }
  }
  await loadChannels();
  toast(`本地化完成：成功 ${success} 个${failed ? '，失败 ' + failed + ' 个' : ''}`);
});

/* ==========================================================================
   Modern EPG Guide
   ========================================================================== */

function getShanghaiTodayStr() {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(new Date());
}

function getWeekdayFromDateStr(dateStr) {
  const [y, m, d] = dateStr.split('-').map(Number);
  const utcDay = new Date(Date.UTC(y, m - 1, d)).getUTCDay();
  return ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][utcDay];
}

function getValidEPGChannels() {
  return channels.filter(c => c.enabled && (c.operator_id || (c.source === 'iptv' && !isInvalidID(c.id)) || isValidID(c.id)));
}

function updateEPGHero() {
  const id = $('epg-channel').value;
  const c = channels.find(ch => (ch.key ? ch.key === id : ch.id === id)) || channels.find(ch => ch.id === id);
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
  if (c.logo && (isLocalLogoPath(c.logo) || /^https?:\/\//i.test(c.logo))) {
    logoImg.onload = () => { logoImg.hidden = false; placeholder.hidden = true; };
    logoImg.onerror = () => { logoImg.hidden = true; placeholder.hidden = false; };
    logoImg.src = c.logo;
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
    const dateStr = new Intl.DateTimeFormat('en-CA', {
      timeZone: 'Asia/Shanghai',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    }).format(d);

    let label = getWeekdayFromDateStr(dateStr);
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
  const select = $('epg-channel');
  const previous = select.value;
  select.replaceChildren();
  const validChannels = getValidEPGChannels();
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
  const validChannels = getValidEPGChannels();
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
  clearTimeout(epgPollTimer);
  epgPollTimer = null;
  const v = await api('epg/status');
  const stateName = JOB_STATES[v.state] || v.state;
  const timeStr = v.updated_at && !v.updated_at.startsWith('0001') ? new Date(v.updated_at).toLocaleString() : '尚未同步';

  $('epg-status').textContent = `${v.backend === 'mysql' ? 'MySQL' : '本地文件'} · ${stateName} · ${v.done}/${v.total} 个频道 · 已存 ${v.programmes} 条节目 · ${timeStr}${v.error ? ' · ' + v.error : ''}`;
  $('epg-progress').max = v.total || 1;
  $('epg-progress').value = v.done;
  $('sync-epg').disabled = v.state === 'running';

  $('val-backend').textContent = v.backend === 'mysql' ? 'MySQL 数据库' : '本地文件 (JSON)';
  const stateEl = $('val-state');
  const dot = document.createElement('span');
  dot.className = v.state === 'running' ? 'pulse-dot is-running' : 'pulse-dot';
  stateEl.replaceChildren(dot, document.createTextNode(v.state === 'running' ? ' 同步中...' : ' 就绪'));

  $('val-channels').textContent = `${v.done} / ${v.total} 频道`;
  $('val-programmes').textContent = `${v.programmes.toLocaleString()} 条`;
  $('val-updated').textContent = timeStr;

  if (v.state === 'running') {
    epgPollTimer = setTimeout(() => { epgStatus().catch(() => {}); }, 2000);
  }
  if (lastEPGState === 'running' && v.state !== 'running') {
    await loadEPG();
  }
  lastEPGState = v.state;
}

function renderEPGList() {
  const container = $('epg-programmes');
  container.replaceChildren();

  const id = $('epg-channel').value;
  const c = channels.find(ch => (ch.key ? ch.key === id : ch.id === id)) || channels.find(ch => ch.id === id);
  const now = Date.now() / 1000;
  const rawQuery = $('epg-search').value || '';
  if ($('epg-search-clear')) $('epg-search-clear').hidden = rawQuery.length === 0;
  const query = rawQuery.trim().toLowerCase();

  const timeFmt = (seconds) => new Date(seconds * 1000).toLocaleTimeString('zh-CN', {
    timeZone: 'Asia/Shanghai',
    hour: '2-digit',
    minute: '2-digit',
  });

  let countLive = 0;
  let countCatchup = 0;
  let countUpcoming = 0;
  for (const p of currentEPGProgrammes) {
    if (p.start <= now && p.end > now) countLive++;
    if (p.end <= now && c && c.catchup_days && p.start >= now - c.catchup_days * 86400) countCatchup++;
    if (p.start > now) countUpcoming++;
  }

  $('count-all').textContent = currentEPGProgrammes.length;
  $('count-live').textContent = countLive;
  $('count-catchup').textContent = countCatchup;
  $('count-upcoming').textContent = countUpcoming;
  if ($('epg-jump-live')) $('epg-jump-live').hidden = countLive === 0;

  const filtered = currentEPGProgrammes.filter(p => {
    const isLive = p.start <= now && p.end > now;
    const isCatchup = p.end <= now && c && c.catchup_days && p.start >= now - c.catchup_days * 86400;
    const isUpcoming = p.start > now;

    if (epgCurrentFilter === 'live' && !isLive) return false;
    if (epgCurrentFilter === 'catchup' && !isCatchup) return false;
    if (epgCurrentFilter === 'upcoming' && !isUpcoming) return false;

    if (query && !p.title.toLowerCase().includes(query) && !(p.desc && p.desc.toLowerCase().includes(query))) {
      return false;
    }
    return true;
  });

  if (!filtered.length) {
    const empty = document.createElement('div');
    empty.className = 'empty empty-epg';
    const icon = document.createElement('div');
    icon.className = 'empty-icon';
    icon.textContent = '📅';
    const h3 = document.createElement('h3');
    h3.textContent = '暂无匹配节目';
    const pEl = document.createElement('p');
    pEl.textContent = '该筛选条件下没有找到节目，请尝试更换日期、清除搜索词或点击立即同步。';
    empty.append(icon, h3, pEl);
    container.append(empty);
    return;
  }

  const frag = document.createDocumentFragment();
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

    // Col 2: Title, Progress, Badges & Description
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

      const totalSec = Math.max(1, p.end - p.start);
      const elapsedSec = Math.max(0, now - p.start);
      const pct = Math.min(100, Math.max(0, Math.round((elapsedSec / totalSec) * 100)));
      const elapsedMin = Math.floor(elapsedSec / 60);
      const remainMin = Math.max(0, Math.ceil((p.end - now) / 60));

      const progressWrap = document.createElement('div');
      progressWrap.className = 'epg-live-progress';
      const progressBar = document.createElement('progress');
      progressBar.className = 'epg-progress-bar';
      progressBar.max = 100;
      progressBar.value = pct;

      const meta = document.createElement('span');
      meta.className = 'epg-progress-meta';
      meta.textContent = `已播 ${elapsedMin} 分钟 · 剩余 ${remainMin} 分钟 (${pct}%)`;

      progressWrap.append(progressBar, meta);
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
        const diffText = startDiff >= 60
          ? `${Math.floor(startDiff / 60)}小时${startDiff % 60 ? (startDiff % 60) + '分' : ''}后开始`
          : `${startDiff}分钟后开始`;
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
      playBtn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg> 观看直播';
      playBtn.onclick = () => {
        play(c);
        $('play-mode').value = c.operator_id ? 'http' : (c.unicast_url ? 'unicast' : 'multicast');
        updatePlayAddress();
      };
      actionCol.append(playBtn);
    } else if (isCatchup) {
      const replayLink = document.createElement('a');
      replayLink.className = 'button secondary btn-catchup';
      replayLink.target = '_blank';
      replayLink.rel = 'noopener';
      const playRef = c.operator_id || c.key || c.id;
      const replayMode = c.operator_id ? 'http' : 'unicast';
      const playUrl = `/api/play?id=${encodeURIComponent(playRef)}&mode=${replayMode}&start=${p.start}&end=${p.end}`;
      replayLink.href = playUrl;
      replayLink.innerHTML = '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><polygon points="11 19 2 12 11 5 11 19"/><polygon points="22 19 13 12 22 5 22 19"/></svg> 回看';

      const copyBtn = document.createElement('button');
      copyBtn.type = 'button';
      copyBtn.className = 'secondary btn-copy-url';
      copyBtn.title = '复制回看地址';
      copyBtn.setAttribute('aria-label', '复制回看地址');
      copyBtn.innerHTML = '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>';
      copyBtn.onclick = () => {
        copyText(new URL(playUrl, location.origin).href, '回看地址已复制到剪贴板');
      };
      actionCol.append(replayLink, copyBtn);
    } else {
      const label = document.createElement('span');
      label.className = 'muted epg-status-label';
      label.textContent = isUpcoming ? '待播出' : '已结束';
      actionCol.append(label);
    }

    card.append(timeCol, infoCol, actionCol);
    frag.append(card);
  }
  container.append(frag);
}

async function loadEPG() {
  const version = ++epgRequestVersion;
  const id = $('epg-channel').value;
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

  if ($('epg-date').value === getShanghaiTodayStr()) {
    setTimeout(() => {
      document.querySelector('.epg-item.is-live')?.scrollIntoView({ behavior: 'smooth', block: 'center' });
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
$('epg-search-clear')?.addEventListener('click', () => {
  $('epg-search').value = '';
  renderEPGList();
  $('epg-search').focus();
});

$('epg-jump-live')?.addEventListener('click', () => {
  if (epgCurrentFilter !== 'all' && epgCurrentFilter !== 'live') {
    document.querySelector('.filter-tab[data-filter="all"]')?.click();
  }
  document.querySelector('.epg-item.is-live')?.scrollIntoView({ behavior: 'smooth', block: 'center' });
});

document.querySelectorAll('.filter-tab[data-filter]').forEach(tabBtn => {
  tabBtn.addEventListener('click', () => {
    document.querySelectorAll('.filter-tab[data-filter]').forEach(b => b.classList.remove('active'));
    tabBtn.classList.add('active');
    epgCurrentFilter = tabBtn.dataset.filter;
    renderEPGList();
  });
});

document.querySelector('[data-tab="epg"]').addEventListener('click', () => {
  epgStatus().then(loadEPG).catch(e => toast(e.message, true));
});

if (epgLiveTimer) clearInterval(epgLiveTimer);
epgLiveTimer = setInterval(() => {
  if (!$('tab-epg').hidden && $('epg-date').value === getShanghaiTodayStr() && currentEPGProgrammes.length) {
    renderEPGList();
  }
}, 45000);

/* ==========================================================================
   Logo Sources Management
   ========================================================================== */

function renderLogoSourcesList() {
  const container = $('logo-sources-list');
  if (!container) return;
  container.replaceChildren();

  if (!currentLogoSources || currentLogoSources.length === 0) {
    currentLogoSources = [...DEFAULT_LOGO_SOURCES];
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
    badge.textContent = info
      ? `已索引 ${info.count || 0} 个${info.dark_count ? ' (深色 ' + info.dark_count + ')' : ''}`
      : '待更新索引';

    const actions = document.createElement('div');
    actions.className = 'source-actions';

    const upBtn = document.createElement('button');
    upBtn.type = 'button';
    upBtn.className = 'secondary btn-group-nav';
    upBtn.textContent = '↑';
    upBtn.title = '提高优先级（向上移动）';
    upBtn.disabled = idx === 0;
    upBtn.onclick = () => {
      [currentLogoSources[idx - 1], currentLogoSources[idx]] = [currentLogoSources[idx], currentLogoSources[idx - 1]];
      renderLogoSourcesList();
    };

    const downBtn = document.createElement('button');
    downBtn.type = 'button';
    downBtn.className = 'secondary btn-group-nav';
    downBtn.textContent = '↓';
    downBtn.title = '降低优先级（向下移动）';
    downBtn.disabled = idx === currentLogoSources.length - 1;
    downBtn.onclick = () => {
      [currentLogoSources[idx + 1], currentLogoSources[idx]] = [currentLogoSources[idx], currentLogoSources[idx + 1]];
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

function addLogoSourceFromInput() {
  const input = $('new-logo-source-input');
  if (!input) return;
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
}

$('add-logo-source-btn')?.addEventListener('click', addLogoSourceFromInput);
$('new-logo-source-input')?.addEventListener('keydown', (e) => {
  if (e.key === 'Enter') {
    e.preventDefault();
    addLogoSourceFromInput();
  }
});

async function logoSourcesStatus() {
  const result = await api('logos/status');
  logoSourcesStatusData = result;
  renderLogoSourcesList();
  $('logos-status').textContent = `已配置 ${result.sources?.length || 0} 个源 · 共计索引 ${result.total_count || 0} 个台标`;
}

$('save-logos')?.addEventListener('click', () => busy($('save-logos'), async () => {
  await saveSettings();
  await loadChannels();
  await logoSourcesStatus();
  toast('台标源已保存；更换来源后请更新台标目录');
}));

$('refresh-logos')?.addEventListener('click', () => busy($('refresh-logos'), async () => {
  await saveSettings();
  $('logos-status').textContent = '正在更新台标目录…';
  const r = await api('logos/refresh', 'POST');
  logoSourcesStatusData = r;
  renderLogoSourcesList();
  await loadChannels();
  $('logos-status').textContent = `共计 ${r.total_count || 0} 个台标${r.errors?.length ? ' · 遇到问题：' + r.errors.join('；') : ' · 已更新完成'}`;
}));

$('reset-logos')?.addEventListener('click', () => {
  currentLogoSources = [...DEFAULT_LOGO_SOURCES];
  renderLogoSourcesList();
  toast('已恢复默认台标源，请点击「保存台标源」确认保存');
});

document.querySelector('[data-tab="ai"]').addEventListener('click', () => {
  logoSourcesStatus().catch(e => toast(e.message, true));
});

/* ==========================================================================
   Keyword Mappings
   ========================================================================== */

async function loadMappings() {
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
    idBadge.className = 'badge mono';
    idBadge.textContent = m.target_id;
    tdID.appendChild(idBadge);

    const tdOp = document.createElement('td');
    tdOp.className = 'mono-cell';
    tdOp.textContent = m.operator_id || '-';

    const tdGrp = document.createElement('td');
    tdGrp.textContent = m.group || '-';

    const tdAct = document.createElement('td');
    const delBtn = document.createElement('button');
    delBtn.type = 'button';
    delBtn.className = 'secondary btn-danger-ghost';
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

    tr.append(tdKw, tdTarget, tdID, tdOp, tdGrp, tdAct);
    tbody.appendChild(tr);
  });
}

$('add-mapping-form')?.addEventListener('submit', async (e) => {
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
    await api('mappings', 'POST', {
      keyword: kw,
      target_name: targetName,
      target_id: targetID,
      operator_id: operatorID,
      group,
    });
    form.reset();
    toast(`已添加关键词「${kw}」映射`);
    await loadMappings();
  });
});

$('reset-mappings-btn')?.addEventListener('click', async () => {
  if (!confirm('确定将频道关键词映射恢复为默认预设吗？')) return;
  try {
    await api('mappings/reset', 'POST');
    toast('已恢复为默认关键词映射');
    await loadMappings();
  } catch (err) {
    toast(`恢复映射失败: ${err.message}`, true);
  }
});

/* ==========================================================================
   Drag-and-Drop Reordering Helper
   ========================================================================== */

function attachDragReorder(container, itemSelector, onReorder) {
  let dragFromIdx = -1;

  container.querySelectorAll(itemSelector).forEach((item, idx) => {
    item.draggable = true;
    item.addEventListener('dragstart', (e) => {
      dragFromIdx = idx;
      item.classList.add('is-dragging');
      if (e.dataTransfer) {
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', String(idx));
      }
    });
    item.addEventListener('dragend', () => {
      dragFromIdx = -1;
      container.querySelectorAll(itemSelector).forEach(el => {
        el.classList.remove('is-dragging', 'drag-over-top', 'drag-over-bottom');
      });
    });
    item.addEventListener('dragover', (e) => {
      if (dragFromIdx === -1 || dragFromIdx === idx) return;
      e.preventDefault();
      if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
      const rect = item.getBoundingClientRect();
      const before = e.clientY < rect.top + rect.height / 2;
      item.classList.toggle('drag-over-top', before);
      item.classList.toggle('drag-over-bottom', !before);
    });
    item.addEventListener('dragleave', () => {
      item.classList.remove('drag-over-top', 'drag-over-bottom');
    });
    item.addEventListener('drop', (e) => {
      if (dragFromIdx === -1 || dragFromIdx === idx) return;
      e.preventDefault();
      const rect = item.getBoundingClientRect();
      const before = e.clientY < rect.top + rect.height / 2;
      let targetIdx = before ? idx : idx + 1;
      if (dragFromIdx < targetIdx) targetIdx--;
      const from = dragFromIdx;
      dragFromIdx = -1;
      if (from !== targetIdx) {
        onReorder(from, targetIdx);
      }
    });
  });
}

/* ==========================================================================
   Group Order & In-Group Channel Sorting
   ========================================================================== */

function renderGroupOrderList() {
  const container = $('group-order-list');
  const select = $('quick-add-group-select');
  if (!container) return;
  container.replaceChildren();

  if (!currentGroupOrder || currentGroupOrder.length === 0) {
    currentGroupOrder = [...DEFAULT_GROUP_ORDER];
  }

  currentGroupOrder.forEach((groupName, idx) => {
    const item = document.createElement('div');
    item.className = 'group-order-item';

    const left = document.createElement('div');
    left.className = 'group-order-left';

    const handle = document.createElement('span');
    handle.className = 'drag-handle';
    handle.textContent = '⋮⋮';
    handle.title = '按住拖动调整顺序';

    const indexSpan = document.createElement('span');
    indexSpan.className = 'group-index';
    indexSpan.textContent = `#${idx + 1}`;

    const nameSpan = document.createElement('span');
    nameSpan.className = 'group-name';
    nameSpan.textContent = groupName;

    const count = channels.filter(c => (c.group || '未分组') === groupName).length;
    const badge = document.createElement('span');
    badge.className = 'badge';
    badge.textContent = `${count} 个频道`;

    left.append(handle, indexSpan, nameSpan, badge);

    const actions = document.createElement('div');
    actions.className = 'group-actions';

    const upBtn = document.createElement('button');
    upBtn.type = 'button';
    upBtn.className = 'secondary btn-group-nav';
    upBtn.textContent = '↑';
    upBtn.title = '向上移动';
    upBtn.disabled = idx === 0;
    upBtn.onclick = () => {
      [currentGroupOrder[idx - 1], currentGroupOrder[idx]] = [currentGroupOrder[idx], currentGroupOrder[idx - 1]];
      renderGroupOrderList();
    };

    const downBtn = document.createElement('button');
    downBtn.type = 'button';
    downBtn.className = 'secondary btn-group-nav';
    downBtn.textContent = '↓';
    downBtn.title = '向下移动';
    downBtn.disabled = idx === currentGroupOrder.length - 1;
    downBtn.onclick = () => {
      [currentGroupOrder[idx + 1], currentGroupOrder[idx]] = [currentGroupOrder[idx], currentGroupOrder[idx + 1]];
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

  attachDragReorder(container, '.group-order-item', (from, to) => {
    const [moved] = currentGroupOrder.splice(from, 1);
    currentGroupOrder.splice(to, 0, moved);
    renderGroupOrderList();
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

function addGroupFromInput() {
  const input = $('new-group-name');
  if (!input) return;
  const val = input.value.trim();
  if (!val) return;
  if (!currentGroupOrder.includes(val)) {
    currentGroupOrder.push(val);
    input.value = '';
    renderGroupOrderList();
  } else {
    toast(`分组「${val}」已在排序列表中`, true);
  }
}

$('add-group-btn')?.addEventListener('click', addGroupFromInput);
$('new-group-name')?.addEventListener('keydown', (e) => {
  if (e.key === 'Enter') {
    e.preventDefault();
    addGroupFromInput();
  }
});

$('quick-add-group-select')?.addEventListener('change', () => {
  const val = $('quick-add-group-select').value;
  if (val && !currentGroupOrder.includes(val)) {
    currentGroupOrder.push(val);
    $('quick-add-group-select').value = '';
    renderGroupOrderList();
  }
});

$('save-group-order-btn')?.addEventListener('click', () => busy($('save-group-order-btn'), async () => {
  if (!settings) settings = (await api('settings')).settings;
  settings.group_order = currentGroupOrder;
  await api('settings', 'PUT', { settings: collectSettings() });
  fillSettings(await api('settings'));
  await loadChannels();
  toast('分组排序设置已保存，频道列表和导出播放列表已按新顺序排序');
}));

$('reset-group-order-btn')?.addEventListener('click', () => {
  currentGroupOrder = [...DEFAULT_GROUP_ORDER];
  renderGroupOrderList();
  toast('已恢复默认分组排序，请点击「保存分组排序」确认保存');
});

$('probe-channel-btn')?.addEventListener('click', () => busy($('probe-channel-btn'), async () => {
  const form = $('channel-form');
  const ref = editing?.key || editing?.id || form.elements.id.value;
  if (!ref || ref === '0') {
    toast('请先保存频道或选择已知频道再探测', true);
    return;
  }
  toast('正在从实时流探测分辨率…');
  const res = await api('channels/' + encodeURIComponent(ref) + '/probe', 'POST');
  if (res && res.resolution) {
    form.elements.resolution.value = res.resolution;
    if (editing) editing.resolution = res.resolution;
    toast(`探测成功: ${res.resolution} (${getResolutionLabel(res.resolution)})`);
  } else {
    toast('未能探测到有效分辨率', true);
  }
}));

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
  container.replaceChildren();
  if (!selectedSortGroup) return;

  const groupChans = getGroupChannels(selectedSortGroup);
  if (groupChans.length === 0) {
    const emptyP = document.createElement('p');
    emptyP.className = 'muted in-group-empty';
    emptyP.textContent = '此分组暂无频道';
    container.append(emptyP);
    return;
  }

  const ordered = [...groupChans];
  const customKeys = currentGroupChannelOrders[selectedSortGroup] || [];
  if (customKeys.length > 0) {
    ordered.sort((a, b) => {
      const iA = customKeys.indexOf(a.key || a.id);
      const iB = customKeys.indexOf(b.key || b.id);
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

    const handle = document.createElement('span');
    handle.className = 'drag-handle';
    handle.textContent = '⋮⋮';
    handle.title = '按住拖动调整顺序';

    const seq = document.createElement('span');
    seq.className = 'in-group-seq';
    seq.textContent = idx + 1;

    const logoBox = document.createElement('div');
    logoBox.className = 'in-group-logo';
    if (c.logo) {
      const img = document.createElement('img');
      img.src = c.logo;
      img.alt = '';
      img.referrerPolicy = 'no-referrer';
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
      idBadge.className = 'badge mono';
      idBadge.textContent = `台号: ${c.id}`;
      meta.append(idBadge);
    }
    if (c.resolution) {
      const resBadge = document.createElement('span');
      resBadge.className = 'badge-res ' + getResolutionClass(c.resolution);
      resBadge.textContent = c.resolution;
      meta.append(resBadge);
    }

    const actions = document.createElement('div');
    actions.className = 'in-group-actions';

    const moveItem = (newIdx) => {
      const list = getCurrentInGroupKeys();
      const [moved] = list.splice(idx, 1);
      list.splice(newIdx, 0, moved);
      currentGroupChannelOrders[selectedSortGroup] = list;
      renderInGroupChannelList();
    };

    const mkBtn = (label, title, disabled, onClick) => {
      const b = document.createElement('button');
      b.type = 'button';
      b.className = 'secondary';
      b.textContent = label;
      b.title = title;
      b.disabled = disabled;
      b.onclick = onClick;
      return b;
    };

    actions.append(
      mkBtn('⤒', '移到置顶', idx === 0, () => moveItem(0)),
      mkBtn('↑', '向上移动', idx === 0, () => moveItem(idx - 1)),
      mkBtn('↓', '向下移动', idx === ordered.length - 1, () => moveItem(idx + 1)),
      mkBtn('⤓', '移到末尾', idx === ordered.length - 1, () => moveItem(ordered.length - 1))
    );

    item.append(handle, seq, logoBox, nameSpan, meta, actions);
    container.append(item);
  });

  attachDragReorder(container, '.in-group-channel-item', (from, to) => {
    const list = getCurrentInGroupKeys();
    const [moved] = list.splice(from, 1);
    list.splice(to, 0, moved);
    currentGroupChannelOrders[selectedSortGroup] = list;
    renderInGroupChannelList();
  });
}

$('group-channel-select')?.addEventListener('change', () => {
  selectedSortGroup = $('group-channel-select').value;
  renderInGroupChannelList();
});

$('sort-in-group-by-id')?.addEventListener('click', () => {
  const groupChans = getGroupChannels(selectedSortGroup);
  groupChans.sort((a, b) => {
    const rankA = getCCTVRank(a.name);
    const rankB = getCCTVRank(b.name);
    if (rankA !== null && rankB !== null && rankA !== rankB) return rankA - rankB;
    if ((rankA !== null) !== (rankB !== null)) return rankA !== null ? -1 : 1;
    const idA = Number(a.id);
    const idB = Number(b.id);
    if (!isNaN(idA) && !isNaN(idB) && idA > 0 && idB > 0 && idA !== idB) return idA - idB;
    if (!isNaN(idA) && idA > 0) return -1;
    if (!isNaN(idB) && idB > 0) return 1;
    return a.name.localeCompare(b.name, 'zh-CN', { numeric: true });
  });
  currentGroupChannelOrders[selectedSortGroup] = groupChans.map(c => c.key || c.id);
  renderInGroupChannelList();
  toast(`已按台号/央视顺序排列「${selectedSortGroup}」频道`);
});

$('sort-in-group-by-name')?.addEventListener('click', () => {
  const groupChans = getGroupChannels(selectedSortGroup);
  groupChans.sort((a, b) => a.name.localeCompare(b.name, 'zh-CN', { numeric: true }));
  currentGroupChannelOrders[selectedSortGroup] = groupChans.map(c => c.key || c.id);
  renderInGroupChannelList();
  toast(`已按名称字典序排列「${selectedSortGroup}」频道`);
});

$('sort-in-group-by-res')?.addEventListener('click', () => {
  const resScore = (res) => {
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
    const diff = resScore(b.resolution) - resScore(a.resolution);
    if (diff !== 0) return diff;
    return a.name.localeCompare(b.name, 'zh-CN', { numeric: true });
  });
  currentGroupChannelOrders[selectedSortGroup] = groupChans.map(c => c.key || c.id);
  renderInGroupChannelList();
  toast(`已按分辨率规格优先排列「${selectedSortGroup}」频道`);
});

$('save-in-group-order-btn')?.addEventListener('click', () => busy($('save-in-group-order-btn'), async () => {
  const keys = getCurrentInGroupKeys();
  if (!currentGroupChannelOrders) currentGroupChannelOrders = {};
  currentGroupChannelOrders[selectedSortGroup] = keys;
  if (!settings) settings = (await api('settings')).settings;
  settings.group_channel_order = currentGroupChannelOrders;
  await api('settings', 'PUT', { settings: collectSettings() });
  fillSettings(await api('settings'));
  await loadChannels();
  toast(`「${selectedSortGroup}」组内频道排序已保存`);
}));

$('reset-in-group-order-btn')?.addEventListener('click', () => {
  if (currentGroupChannelOrders && currentGroupChannelOrders[selectedSortGroup]) {
    delete currentGroupChannelOrders[selectedSortGroup];
  }
  renderInGroupChannelList();
  toast(`已恢复「${selectedSortGroup}」默认组内顺序，请点击「保存组内排序」确认保存`);
});

/* ==========================================================================
   Batch Probe Progress & Polling
   ========================================================================== */

async function checkProbeStatus() {
  try {
    const st = await api('probe/status');
    const card = $('probe-progress-card');
    if (!card) return;

    if (st.state === 'running') {
      card.hidden = false;
      $('probe-progress-title').textContent = `正在一键探测分辨率… (已获取 ${st.found} 个规格)`;
      $('probe-progress-count').textContent = `${st.done} / ${st.total}`;
      const bar = $('probe-progress-bar');
      if (bar) {
        bar.max = st.total || 1;
        bar.value = st.done || 0;
      }
      if (!probePollTimer) {
        probePollTimer = setInterval(checkProbeStatus, 1000);
      }
    } else {
      if (probePollTimer) {
        clearInterval(probePollTimer);
        probePollTimer = null;
      }
      card.hidden = true;
      if (lastProbeState === 'running') {
        if (st.state === 'completed') {
          toast(`一键探测完成：共完成 ${st.done} 个频道探测，获取到 ${st.found} 个视频分辨率规格`);
          await loadChannels();
        } else if (st.state === 'stopped') {
          toast(`一键探测已停止：完成 ${st.done}/${st.total} 个频道`);
          await loadChannels();
        }
      }
    }
    lastProbeState = st.state;
  } catch (_) {
    if (probePollTimer) {
      clearInterval(probePollTimer);
      probePollTimer = null;
    }
  }
}

$('batch-probe-btn')?.addEventListener('click', async () => {
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
});

$('stop-probe-btn')?.addEventListener('click', async () => {
  try {
    await api('probe/stop', 'POST');
    toast('正在停止探测…');
    await checkProbeStatus();
  } catch (e) {
    toast('停止探测失败: ' + e.message, true);
  }
});

// Initialize workspace
init().catch(() => {});
