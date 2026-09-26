/**
 * IPTV Web Player Engine
 * Self-contained HTML5 Player supporting MPEG-TS over HTTP (MSE) and HLS (m3u8).
 */
(() => {
  'use strict';

  // State
  const state = {
    channels: [],
    filteredChannels: [],
    currentChannel: null,
    currentStreamUrl: '',
    currentEngineType: 'auto',
    activeEngine: null, // mpegts or hls instance
    playMode: localStorage.getItem('iptv_play_mode') || 'multicast',
    aspectRatio: localStorage.getItem('iptv_aspect_ratio') || 'contain',
    lowLatency: localStorage.getItem('iptv_low_latency') !== 'false',
    autoCleanup: localStorage.getItem('iptv_auto_cleanup') !== 'false',
    alwaysProxy: localStorage.getItem('iptv_always_proxy') === 'true',
    volume: parseFloat(localStorage.getItem('iptv_volume') || '1'),
    isMuted: localStorage.getItem('iptv_muted') === 'true',
    currentCategory: 'all',
    searchQuery: '',
    osdTimer: null,
    controlsTimer: null,
    dialBuffer: '',
    dialTimer: null,
    epgData: {}, // channel -> programmes
    epgDayOffset: 0,
  };

  // DOM Elements
  const $ = (id) => document.getElementById(id);
  const video = $('video');
  const videoWrapper = $('video-wrapper');
  const osd = $('osd');
  const loadingSpinner = $('loading-spinner');
  const loadingText = $('loading-text');
  const errorDialog = $('error-dialog');
  const errorMessage = $('error-message');
  const dialOverlay = $('dial-overlay');
  const dialNumber = $('dial-number');
  const channelDrawer = $('channel-drawer');
  const epgDrawer = $('epg-drawer');
  const streamModal = $('stream-modal');
  const settingsModal = $('settings-modal');

  // Controls Elements
  const btnPlayPause = $('btn-play-pause');
  const iconPlay = $('icon-play');
  const iconPause = $('icon-pause');
  const btnPrevChannel = $('btn-prev-channel');
  const btnNextChannel = $('btn-next-channel');
  const btnMute = $('btn-mute');
  const iconVolHigh = $('icon-vol-high');
  const iconVolMute = $('icon-vol-mute');
  const volumeSlider = $('volume-slider');
  const ctrlCurrentTitle = $('ctrl-current-title');
  const statsPill = $('stats-pill');
  const btnAspect = $('btn-aspect');
  const btnPip = $('btn-pip');
  const btnFullscreen = $('btn-fullscreen');
  const iconFsEnter = $('icon-fs-enter');
  const iconFsExit = $('icon-fs-exit');

  // Initialize
  window.addEventListener('DOMContentLoaded', init);

  async function init() {
    setupEventListeners();
    applyInitialSettings();
    await loadPlaylist();

    // Check URL parameters
    const params = new URLSearchParams(window.location.search);
    const customUrl = params.get('url');
    const channelId = params.get('id') || params.get('channel');
    const mode = params.get('mode');

    if (mode && ['multicast', 'http', 'unicast'].includes(mode)) {
      state.playMode = mode;
      $('select-play-mode').value = mode;
    }

    if (customUrl) {
      playStream(customUrl, { title: '自定义流媒体', isCustom: true });
    } else if (channelId) {
      const match = state.channels.find(c => String(c.id) === String(channelId) || c.name === channelId || c.key === channelId);
      if (match) {
        selectChannel(match);
      } else if (state.channels.length > 0) {
        selectChannel(state.channels[0]);
      }
    } else {
      const lastId = localStorage.getItem('iptv_last_channel_id');
      const match = state.channels.find(c => String(c.id) === String(lastId));
      if (match) {
        selectChannel(match);
      } else if (state.channels.length > 0) {
        selectChannel(state.channels[0]);
      }
    }
  }

  function applyInitialSettings() {
    video.volume = state.volume;
    video.muted = state.isMuted;
    volumeSlider.value = state.isMuted ? 0 : state.volume;
    updateVolumeIcon();
    setAspectRatio(state.aspectRatio);
    $('select-play-mode').value = state.playMode;
    $('chk-opt-lowlatency').checked = state.lowLatency;
    $('chk-opt-cleanup').checked = state.autoCleanup;
    $('chk-opt-global-proxy').checked = state.alwaysProxy;
  }

  // Load Playlist & Channels
  async function loadPlaylist() {
    try {
      const res = await fetch('/api/playlist?fmt=json');
      if (!res.ok) throw new Error('HTTP ' + res.status);
      const data = await res.json();
      state.channels = Array.isArray(data) ? data : [];
      renderChannelList();
    } catch (err) {
      console.warn('Failed to load playlist json, trying fallback m3u:', err);
      state.channels = [];
    }
  }

  // Filter & Categorize Channels
  function getFilteredChannels() {
    return state.channels.filter(c => {
      // Group filter
      if (state.currentCategory !== 'all') {
        const name = (c.name || '').toUpperCase();
        const group = (c.group || '').toUpperCase();
        if (state.currentCategory === 'cctv') {
          if (!name.includes('CCTV') && !name.includes('CGTN') && !group.includes('央视')) return false;
        } else if (state.currentCategory === 'weishi') {
          if (!name.includes('卫视') && !group.includes('卫视')) return false;
        } else if (state.currentCategory === '4k') {
          if (!name.includes('4K') && !name.includes('8K') && !group.includes('4K')) return false;
        } else if (state.currentCategory === 'sports') {
          if (!name.includes('体育') && !name.includes('五星') && !name.includes('风云') && !group.includes('体育')) return false;
        } else if (state.currentCategory === 'local') {
          if (!name.includes('新闻综合') && !name.includes('都市') && !name.includes('东方') && !name.includes('第一财经') && !group.includes('地方') && !group.includes('上海')) return false;
        } else if (state.currentCategory === 'other') {
          // anything else
        }
      }
      // Search filter
      if (state.searchQuery) {
        const q = state.searchQuery.toLowerCase();
        const matchesName = (c.name || '').toLowerCase().includes(q);
        const matchesId = String(c.id || '').includes(q);
        const matchesNum = String(c.num || '').includes(q);
        if (!matchesName && !matchesId && !matchesNum) return false;
      }
      return true;
    });
  }

  function renderChannelList() {
    const listEl = $('channel-list');
    const channels = getFilteredChannels();
    state.filteredChannels = channels;

    if (channels.length === 0) {
      listEl.innerHTML = '<div class="empty-hint">未找到匹配频道</div>';
      return;
    }

    listEl.innerHTML = channels.map((c, idx) => {
      const isActive = state.currentChannel && String(state.currentChannel.id) === String(c.id);
      const logoSrc = c.logo ? c.logo : '/logos/' + encodeURIComponent(c.name) + '.png';
      const num = c.num || (idx + 1);
      return `
        <div class="channel-card ${isActive ? 'active' : ''}" data-id="${c.id}">
          <span class="card-num">${num}</span>
          <div class="card-logo-wrap">
            <img class="card-logo" src="${logoSrc}" alt="" onerror="this.style.display='none'">
          </div>
          <div class="card-meta">
            <div class="card-name-row">
              <span class="card-name">${escapeHtml(c.name)}</span>
              ${c.group ? `<span class="card-group">${escapeHtml(c.group)}</span>` : ''}
            </div>
            <div class="card-program" id="card-prog-${c.id}">${c.current_program || '精彩直播中'}</div>
          </div>
        </div>
      `;
    }).join('');

    // Bind card click
    listEl.querySelectorAll('.channel-card').forEach(card => {
      card.addEventListener('click', () => {
        const id = card.dataset.id;
        const channel = state.channels.find(c => String(c.id) === String(id));
        if (channel) {
          selectChannel(channel);
          if (window.innerWidth <= 640) closeDrawers();
        }
      });
    });
  }

  // Select Channel
  function selectChannel(channel) {
    if (!channel) return;
    state.currentChannel = channel;
    localStorage.setItem('iptv_last_channel_id', channel.id);

    // Highlight active card
    document.querySelectorAll('.channel-card').forEach(card => {
      card.classList.toggle('active', String(card.dataset.id) === String(channel.id));
    });

    // Resolve URL based on playMode
    let streamUrl = '';
    if (state.playMode === 'multicast') {
      streamUrl = channel.play_url || channel.url || `/api/play?id=${encodeURIComponent(channel.id)}&mode=multicast`;
    } else if (state.playMode === 'http') {
      streamUrl = `/api/play?id=${encodeURIComponent(channel.id)}&mode=http`;
    } else if (state.playMode === 'unicast') {
      streamUrl = channel.unicast_url || `/api/play?id=${encodeURIComponent(channel.id)}&mode=unicast`;
    }

    playStream(streamUrl, {
      title: channel.name,
      channel: channel,
      useProxy: state.alwaysProxy,
    });

    loadChannelEPG(channel);
  }

  // Core Playback Function
  function playStream(url, options = {}) {
    hideError();
    showLoading('正在连接流媒体...');
    destroyCurrentEngine();

    state.currentStreamUrl = url;
    let targetUrl = url;

    // Apply proxy wrapper if requested
    if (options.useProxy || state.alwaysProxy) {
      targetUrl = `/api/stream/proxy?url=${encodeURIComponent(url)}`;
    }

    const title = options.title || (options.channel ? options.channel.name : '实时直播流');
    ctrlCurrentTitle.textContent = title;
    showOSD(options.channel, title);

    // Engine Selection
    const engineType = options.engine || state.currentEngineType;
    let chosenEngine = engineType;

    if (chosenEngine === 'auto') {
      if (targetUrl.includes('.m3u8') || targetUrl.includes('/live.m3u8')) {
        chosenEngine = 'hls';
      } else if (targetUrl.includes('/rtp/') || targetUrl.includes('/udp/') || targetUrl.includes(':5140') || targetUrl.includes('.ts')) {
        chosenEngine = 'mpegts';
      } else if (window.mpegts && mpegts.isSupported()) {
        chosenEngine = 'mpegts';
      } else if (window.Hls && Hls.isSupported()) {
        chosenEngine = 'hls';
      } else {
        chosenEngine = 'native';
      }
    }

    updateStatsBadge(chosenEngine);

    if (chosenEngine === 'mpegts') {
      playWithMpegts(targetUrl, options);
    } else if (chosenEngine === 'hls') {
      playWithHls(targetUrl, options);
    } else {
      playWithNative(targetUrl, options);
    }
  }

  // MPEG-TS Playback Engine
  function playWithMpegts(url, options) {
    if (!window.mpegts || !mpegts.isSupported()) {
      showError('当前浏览器不支持 MediaSource Extensions (MSE) 解码 MPEG-TS 流。');
      return;
    }

    try {
      const player = mpegts.createPlayer({
        type: 'mse',
        isLive: true,
        url: url,
      }, {
        enableWorker: true,
        lazyLoad: false,
        liveBufferLatencyChasing: state.lowLatency,
        liveBufferLatencyMaxLatency: 3.5,
        liveBufferLatencyMinRemain: 1.0,
        autoCleanupSourceBuffer: state.autoCleanup,
        autoCleanupMaxBackwardDuration: 30,
        autoCleanupMinBackwardDuration: 15,
      });

      player.attachMediaElement(video);
      player.load();
      player.play().catch(handleAutoplayBlock);

      player.on(mpegts.Events.ERROR, (type, detail, info) => {
        console.warn('mpegts error:', type, detail, info);
        if (type === mpegts.ErrorTypes.NETWORK_ERROR) {
          showError(`流媒体网络错误 (${detail})。若上游服务未启用 CORS 跨域响应，请尝试点击下方“通过服务代理播放”。`, true);
        } else if (type === mpegts.ErrorTypes.MEDIA_ERROR) {
          showError(`媒体解码异常 (${detail})，正在尝试自动恢复...`);
          player.recoverMediaError();
        } else {
          showError(`播放发生错误: ${detail}`);
        }
      });

      player.on(mpegts.Events.STATISTICS_INFO, (stats) => {
        if (stats && stats.speed) {
          const speedKB = Math.round(stats.speed);
          statsPill.textContent = `TS ${speedKB} KB/s`;
          $('osd-stats').textContent = `MPEG-TS MSE | ${speedKB} KB/s`;
        }
      });

      state.activeEngine = player;
    } catch (e) {
      console.error('Failed to init mpegts player:', e);
      showError('初始化 MPEG-TS 解码器失败: ' + e.message, true);
    }
  }

  // HLS Playback Engine
  function playWithHls(url, options) {
    if (window.Hls && Hls.isSupported()) {
      const hls = new Hls({
        enableWorker: true,
        lowLatencyMode: state.lowLatency,
        backBufferLength: 30,
      });

      hls.loadSource(url);
      hls.attachMedia(video);

      hls.on(Hls.Events.MANIFEST_PARSED, () => {
        video.play().catch(handleAutoplayBlock);
        hideLoading();
      });

      hls.on(Hls.Events.ERROR, (event, data) => {
        console.warn('HLS error:', data);
        if (data.fatal) {
          switch (data.type) {
            case Hls.ErrorTypes.NETWORK_ERROR:
              showError(`HLS 网络加载失败 (${data.details})。请检查地址是否有效或启用服务代理。`, true);
              break;
            case Hls.ErrorTypes.MEDIA_ERROR:
              hls.recoverMediaError();
              break;
            default:
              showError(`HLS 播放发生不可恢复错误: ${data.details}`, true);
              destroyCurrentEngine();
              break;
          }
        }
      });

      state.activeEngine = hls;
    } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
      playWithNative(url, options);
    } else {
      showError('当前浏览器不支持 HLS (.m3u8) 流媒体播放。');
    }
  }

  // Native HTML5 Video Playback
  function playWithNative(url, options) {
    video.src = url;
    video.load();
    video.play().catch(handleAutoplayBlock);
  }

  function destroyCurrentEngine() {
    if (state.activeEngine) {
      if (typeof state.activeEngine.destroy === 'function') {
        state.activeEngine.destroy();
      }
      state.activeEngine = null;
    }
    video.removeAttribute('src');
    video.load();
  }

  function handleAutoplayBlock(err) {
    console.warn('Autoplay prevented:', err);
    hideLoading();
    updatePlayPauseState(false);
  }

  // UI Updates & OSD
  function showOSD(channel, customTitle) {
    clearTimeout(state.osdTimer);
    const osdNumber = $('osd-number');
    const osdName = $('osd-name');
    const osdLogo = $('osd-logo');
    const osdLogoBox = $('osd-logo-box');
    const osdTag = $('osd-tag');
    const osdProgTitle = $('osd-program-title');
    const osdProgTime = $('osd-program-time');

    if (channel) {
      osdNumber.textContent = channel.num ? `${channel.num}` : '';
      osdName.textContent = channel.name;
      const logo = channel.logo || `/logos/${encodeURIComponent(channel.name)}.png`;
      osdLogo.src = logo;
      osdLogoBox.style.display = 'flex';
      osdTag.textContent = channel.group || 'IPTV';
      osdProgTitle.textContent = channel.current_program || '精彩节目直播中';
      osdProgTime.textContent = 'LIVE';
    } else {
      osdNumber.textContent = '';
      osdName.textContent = customTitle || '自定义流媒体';
      osdLogoBox.style.display = 'none';
      osdTag.textContent = 'CUSTOM';
      osdProgTitle.textContent = state.currentStreamUrl;
      osdProgTime.textContent = 'RAW';
    }

    osd.classList.remove('hidden');
    state.osdTimer = setTimeout(() => {
      osd.classList.add('hidden');
    }, 4500);
  }

  function showLoading(text) {
    loadingText.textContent = text || '正在缓冲流媒体...';
    loadingSpinner.classList.remove('hidden');
  }

  function hideLoading() {
    loadingSpinner.classList.add('hidden');
  }

  function showError(msg, showProxyOption = false) {
    hideLoading();
    errorMessage.textContent = msg;
    $('btn-error-proxy').style.display = showProxyOption ? 'inline-block' : 'none';
    errorDialog.classList.remove('hidden');
  }

  function hideError() {
    errorDialog.classList.add('hidden');
  }

  function updateStatsBadge(engineName) {
    statsPill.textContent = (engineName || 'auto').toUpperCase();
    $('osd-stats').textContent = (engineName || 'auto').toUpperCase();
  }

  function updateVolumeIcon() {
    if (video.muted || video.volume === 0) {
      iconVolHigh.classList.add('hidden');
      iconVolMute.classList.remove('hidden');
    } else {
      iconVolHigh.classList.remove('hidden');
      iconVolMute.classList.add('hidden');
    }
  }

  function updatePlayPauseState(isPlaying) {
    if (isPlaying) {
      iconPlay.classList.add('hidden');
      iconPause.classList.remove('hidden');
    } else {
      iconPlay.classList.remove('hidden');
      iconPause.classList.add('hidden');
    }
  }

  function setAspectRatio(ratio) {
    videoWrapper.classList.remove('aspect-contain', 'aspect-fill', 'aspect-cover', 'aspect-16-9', 'aspect-4-3');
    videoWrapper.classList.add('aspect-' + ratio);
    state.aspectRatio = ratio;
    localStorage.setItem('iptv_aspect_ratio', ratio);
  }

  function cycleAspectRatio() {
    const list = ['contain', '16-9', '4-3', 'fill'];
    const idx = list.indexOf(state.aspectRatio);
    const next = list[(idx + 1) % list.length];
    setAspectRatio(next);
    showOSD(state.currentChannel, `画面比例: ${next.toUpperCase()}`);
  }

  // EPG Loading
  async function loadChannelEPG(channel) {
    if (!channel) return;
    const name = channel.name;
    const logoSrc = channel.logo || `/logos/${encodeURIComponent(name)}.png`;
    $('epg-header-name').textContent = name;
    $('epg-header-logo').src = logoSrc;
    $('epg-program-list').innerHTML = '<div class="empty-hint">正在获取节目单...</div>';

    try {
      const res = await fetch(`/api/epg/programmes?channel=${encodeURIComponent(name)}`);
      if (!res.ok) throw new Error('EPG not available');
      const data = await res.json();
      state.epgData[name] = data;
      renderEPGList(name);
    } catch (err) {
      $('epg-program-list').innerHTML = '<div class="empty-hint">暂未收录该频道电子节目单</div>';
    }
  }

  function renderEPGList(channelName) {
    const listEl = $('epg-program-list');
    const programmes = state.epgData[channelName] || [];
    if (programmes.length === 0) {
      listEl.innerHTML = '<div class="empty-hint">今日暂无详细节目单</div>';
      return;
    }

    const now = new Date();
    listEl.innerHTML = programmes.map(p => {
      const start = new Date(p.start);
      const end = new Date(p.end);
      const isCurrent = now >= start && now <= end;
      const isPast = now > end;
      const timeStr = `${formatTime(start)} - ${formatTime(end)}`;

      return `
        <div class="epg-card ${isCurrent ? 'current' : ''}">
          <div class="epg-time-row">
            <span>${timeStr}</span>
            ${isCurrent ? '<span style="color:#38bdf8;font-weight:700;">● 正在播出</span>' : ''}
            ${isPast && state.currentChannel && state.currentChannel.catchup_days > 0 ?
              `<button class="btn-catchup" data-start="${p.start}" data-end="${p.end}">回看</button>` : ''}
          </div>
          <div class="epg-title">${escapeHtml(p.title)}</div>
        </div>
      `;
    }).join('');

    // Bind catchup buttons
    listEl.querySelectorAll('.btn-catchup').forEach(btn => {
      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        const start = btn.dataset.start;
        const end = btn.dataset.end;
        playCatchup(state.currentChannel, start, end);
      });
    });
  }

  function playCatchup(channel, start, end) {
    if (!channel) return;
    const playUrl = `/api/play?id=${encodeURIComponent(channel.id)}&start=${encodeURIComponent(start)}&end=${encodeURIComponent(end)}&mode=${state.playMode}`;
    playStream(playUrl, {
      title: `${channel.name} (回看)`,
      channel: channel,
    });
  }

  // Drawers and Modals
  function toggleChannelDrawer() {
    channelDrawer.classList.toggle('open');
    if (channelDrawer.classList.contains('open')) {
      epgDrawer.classList.remove('open');
      $('channel-search').focus();
    }
  }

  function toggleEPGDrawer() {
    epgDrawer.classList.toggle('open');
    if (epgDrawer.classList.contains('open')) {
      channelDrawer.classList.remove('open');
      if (state.currentChannel) loadChannelEPG(state.currentChannel);
    }
  }

  function closeDrawers() {
    channelDrawer.classList.remove('open');
    epgDrawer.classList.remove('open');
  }

  // Channel Dialing (e.g. typing "1", "2")
  function handleDialKey(digit) {
    clearTimeout(state.dialTimer);
    state.dialBuffer += digit;
    dialNumber.textContent = state.dialBuffer;
    dialOverlay.classList.remove('hidden');

    state.dialTimer = setTimeout(() => {
      const targetNum = parseInt(state.dialBuffer, 10);
      state.dialBuffer = '';
      dialOverlay.classList.add('hidden');

      const match = state.channels.find(c => c.num === targetNum || String(c.id) === String(targetNum));
      if (match) {
        selectChannel(match);
      } else if (targetNum > 0 && targetNum <= state.channels.length) {
        selectChannel(state.channels[targetNum - 1]);
      }
    }, 1200);
  }

  // Event Listeners
  function setupEventListeners() {
    // Video state events
    video.addEventListener('playing', () => {
      hideLoading();
      hideError();
      updatePlayPauseState(true);
    });
    video.addEventListener('pause', () => updatePlayPauseState(false));
    video.addEventListener('waiting', () => showLoading('正在缓冲...'));
    video.addEventListener('loadedmetadata', () => {
      hideLoading();
      const res = `${video.videoWidth}x${video.videoHeight}`;
      $('osd-stats').textContent = `${statsPill.textContent} | ${res}`;
    });

    // Controls bar buttons
    btnPlayPause.addEventListener('click', () => {
      if (video.paused) video.play();
      else video.pause();
    });

    btnPrevChannel.addEventListener('click', switchPrevChannel);
    btnNextChannel.addEventListener('click', switchNextChannel);

    btnMute.addEventListener('click', () => {
      video.muted = !video.muted;
      state.isMuted = video.muted;
      localStorage.setItem('iptv_muted', state.isMuted);
      volumeSlider.value = video.muted ? 0 : video.volume;
      updateVolumeIcon();
    });

    volumeSlider.addEventListener('input', (e) => {
      const val = parseFloat(e.target.value);
      video.volume = val;
      video.muted = (val === 0);
      state.volume = val;
      state.isMuted = video.muted;
      localStorage.setItem('iptv_volume', val);
      localStorage.setItem('iptv_muted', state.isMuted);
      updateVolumeIcon();
    });

    btnAspect.addEventListener('click', cycleAspectRatio);

    btnPip.addEventListener('click', async () => {
      try {
        if (document.pictureInPictureElement) {
          await document.exitPictureInPicture();
        } else if (video.requestPictureInPicture) {
          await video.requestPictureInPicture();
        }
      } catch (err) {
        console.warn('PiP error:', err);
      }
    });

    btnFullscreen.addEventListener('click', toggleFullscreen);

    // Header buttons
    $('btn-toggle-channels').addEventListener('click', toggleChannelDrawer);
    $('btn-close-channels').addEventListener('click', closeDrawers);
    $('btn-toggle-epg').addEventListener('click', toggleEPGDrawer);
    $('btn-close-epg').addEventListener('click', closeDrawers);

    $('btn-open-stream-modal').addEventListener('click', () => {
      $('input-stream-url').value = state.currentStreamUrl || 'http://tvpanel.netioe.com/rtp/233.18.204.215:5140?fcc=124.75.26.151%3A15970';
      $('chk-stream-proxy').checked = state.alwaysProxy;
      streamModal.classList.remove('hidden');
      $('input-stream-url').focus();
    });
    $('btn-close-stream-modal').addEventListener('click', () => streamModal.classList.add('hidden'));
    $('btn-cancel-stream').addEventListener('click', () => streamModal.classList.add('hidden'));

    // Custom stream presets
    $('btn-preset-user').addEventListener('click', () => {
      $('input-stream-url').value = 'http://tvpanel.netioe.com/rtp/233.18.204.215:5140?fcc=124.75.26.151%3A15970';
      $('select-stream-engine').value = 'mpegts';
    });

    $('btn-preset-cctv1').addEventListener('click', () => {
      const cctv1 = state.channels.find(c => (c.name || '').includes('CCTV-1'));
      if (cctv1) {
        $('input-stream-url').value = cctv1.play_url || cctv1.url || '';
        $('select-stream-engine').value = 'mpegts';
      }
    });

    $('btn-play-custom-stream').addEventListener('click', () => {
      const url = $('input-stream-url').value.trim();
      const engine = $('select-stream-engine').value;
      const proxy = $('chk-stream-proxy').checked;
      if (!url) return;
      streamModal.classList.add('hidden');
      playStream(url, {
        title: '自定义流媒体',
        engine: engine,
        useProxy: proxy,
        isCustom: true,
      });
    });

    // Settings modal
    $('btn-open-settings').addEventListener('click', () => settingsModal.classList.remove('hidden'));
    $('btn-close-settings').addEventListener('click', () => settingsModal.classList.add('hidden'));
    $('btn-save-settings').addEventListener('click', () => {
      state.lowLatency = $('chk-opt-lowlatency').checked;
      state.autoCleanup = $('chk-opt-cleanup').checked;
      state.alwaysProxy = $('chk-opt-global-proxy').checked;
      localStorage.setItem('iptv_low_latency', state.lowLatency);
      localStorage.setItem('iptv_auto_cleanup', state.autoCleanup);
      localStorage.setItem('iptv_always_proxy', state.alwaysProxy);
      settingsModal.classList.add('hidden');
      if (state.currentStreamUrl) {
        playStream(state.currentStreamUrl, {
          title: ctrlCurrentTitle.textContent,
          channel: state.currentChannel,
          useProxy: state.alwaysProxy,
        });
      }
    });

    // Error actions
    $('btn-error-retry').addEventListener('click', () => {
      if (state.currentStreamUrl) {
        playStream(state.currentStreamUrl, {
          title: ctrlCurrentTitle.textContent,
          channel: state.currentChannel,
          useProxy: state.alwaysProxy,
        });
      }
    });

    $('btn-error-proxy').addEventListener('click', () => {
      if (state.currentStreamUrl) {
        playStream(state.currentStreamUrl, {
          title: ctrlCurrentTitle.textContent,
          channel: state.currentChannel,
          useProxy: true,
        });
      }
    });

    $('btn-error-engine').addEventListener('click', () => {
      const engines = ['auto', 'mpegts', 'hls', 'native'];
      const next = engines[(engines.indexOf(state.currentEngineType) + 1) % engines.length];
      state.currentEngineType = next;
      if (state.currentStreamUrl) {
        playStream(state.currentStreamUrl, {
          title: ctrlCurrentTitle.textContent,
          channel: state.currentChannel,
          engine: next,
          useProxy: state.alwaysProxy,
        });
      }
    });

    // Channel search
    $('channel-search').addEventListener('input', (e) => {
      state.searchQuery = e.target.value.trim();
      renderChannelList();
    });

    // Category tabs
    $('category-tabs').querySelectorAll('.tab-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        $('category-tabs').querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        state.currentCategory = btn.dataset.group;
        renderChannelList();
      });
    });

    // Mode dropdown
    $('select-play-mode').addEventListener('change', (e) => {
      state.playMode = e.target.value;
      localStorage.setItem('iptv_play_mode', state.playMode);
      if (state.currentChannel) {
        selectChannel(state.currentChannel);
      }
    });

    // Mouse movement to reveal controls
    let moveTimer = null;
    videoWrapper.addEventListener('mousemove', () => {
      videoWrapper.classList.remove('hide-controls');
      clearTimeout(moveTimer);
      moveTimer = setTimeout(() => {
        if (!video.paused && !channelDrawer.classList.contains('open') && !epgDrawer.classList.contains('open')) {
          videoWrapper.classList.add('hide-controls');
        }
      }, 3500);
    });

    ctrlCurrentTitle.addEventListener('click', () => {
      showOSD(state.currentChannel, ctrlCurrentTitle.textContent);
    });

    // Keyboard Shortcuts
    window.addEventListener('keydown', handleKeyboardShortcuts);
  }

  function switchPrevChannel() {
    const list = state.channels;
    if (list.length === 0) return;
    let idx = list.findIndex(c => state.currentChannel && String(c.id) === String(state.currentChannel.id));
    idx = (idx - 1 + list.length) % list.length;
    selectChannel(list[idx]);
  }

  function switchNextChannel() {
    const list = state.channels;
    if (list.length === 0) return;
    let idx = list.findIndex(c => state.currentChannel && String(c.id) === String(state.currentChannel.id));
    idx = (idx + 1) % list.length;
    selectChannel(list[idx]);
  }

  function toggleFullscreen() {
    if (!document.fullscreenElement) {
      (videoWrapper.requestFullscreen || videoWrapper.webkitRequestFullscreen || video.webkitEnterFullscreen).call(videoWrapper);
      iconFsEnter.classList.add('hidden');
      iconFsExit.classList.remove('hidden');
    } else {
      (document.exitFullscreen || document.webkitExitFullscreen).call(document);
      iconFsEnter.classList.remove('hidden');
      iconFsExit.classList.add('hidden');
    }
  }

  function handleKeyboardShortcuts(e) {
    // If inside text input, ignore shortcuts except Escape
    if (['INPUT', 'SELECT', 'TEXTAREA'].includes(e.target.tagName)) {
      if (e.key === 'Escape') {
        e.target.blur();
        closeDrawers();
        streamModal.classList.add('hidden');
        settingsModal.classList.add('hidden');
      }
      return;
    }

    if (e.key >= '0' && e.key <= '9') {
      handleDialKey(e.key);
      return;
    }

    switch (e.key) {
      case 'ArrowUp':
      case 'PageUp':
        e.preventDefault();
        switchPrevChannel();
        break;
      case 'ArrowDown':
      case 'PageDown':
        e.preventDefault();
        switchNextChannel();
        break;
      case 'ArrowLeft':
        e.preventDefault();
        video.volume = Math.max(0, video.volume - 0.05);
        volumeSlider.value = video.volume;
        updateVolumeIcon();
        showOSD(null, `音量: ${Math.round(video.volume * 100)}%`);
        break;
      case 'ArrowRight':
        e.preventDefault();
        video.volume = Math.min(1, video.volume + 0.05);
        volumeSlider.value = video.volume;
        updateVolumeIcon();
        showOSD(null, `音量: ${Math.round(video.volume * 100)}%`);
        break;
      case ' ':
      case 'k':
      case 'Enter':
        e.preventDefault();
        if (video.paused) video.play();
        else video.pause();
        break;
      case 'f':
      case 'F':
        e.preventDefault();
        toggleFullscreen();
        break;
      case 'm':
      case 'M':
        e.preventDefault();
        video.muted = !video.muted;
        volumeSlider.value = video.muted ? 0 : video.volume;
        updateVolumeIcon();
        showOSD(null, video.muted ? '已静音' : `音量: ${Math.round(video.volume * 100)}%`);
        break;
      case 'c':
      case 'C':
        e.preventDefault();
        toggleChannelDrawer();
        break;
      case 'e':
      case 'E':
        e.preventDefault();
        toggleEPGDrawer();
        break;
      case 'u':
      case 'U':
        e.preventDefault();
        $('btn-open-stream-modal').click();
        break;
      case 'Escape':
        closeDrawers();
        streamModal.classList.add('hidden');
        settingsModal.classList.add('hidden');
        break;
    }
  }

  // Utilities
  function escapeHtml(str) {
    if (!str) return '';
    return str.replace(/[&<>"']/g, m => ({
      '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    })[m]);
  }

  function formatTime(date) {
    if (!(date instanceof Date) || isNaN(date.getTime())) return '--:--';
    const h = String(date.getHours()).padStart(2, '0');
    const m = String(date.getMinutes()).padStart(2, '0');
    return `${h}:${m}`;
  }
})();
