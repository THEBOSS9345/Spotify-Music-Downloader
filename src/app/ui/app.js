let state = {
  playlists: [],
  currentPlaylistId: null,
  currentSongs: [],
  selectedIds: new Set(),
  downloads: [],
  polling: false,
};

function $(id) { return document.getElementById(id); }

function showView(id) {
  document.querySelectorAll('.view').forEach(v => v.classList.add('hidden'));
  $(id).classList.remove('hidden');
}

function showPage(id) {
  document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
  $(id).classList.add('active');
}

function formatDuration(seconds) {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${s.toString().padStart(2, '0')}`;
}

function toast(message, type = 'info') {
  const container = $('toast-container');
  const el = document.createElement('div');
  el.className = `toast ${type}`;
  el.textContent = message;
  container.appendChild(el);
  setTimeout(() => el.remove(), 3000);
}

// ========== LOGIN ==========
async function handleLogin() {
  const btn = $('btn-login');
  btn.disabled = true;
  btn.textContent = 'Opening browser...';

  try {
    const url = await loginURL();
    window.open(url, '_blank');
    btn.textContent = 'Waiting for login...';

    const pollInterval = setInterval(async () => {
      const authed = await isAuthenticated();
      if (authed) {
        clearInterval(pollInterval);
        onAuthDone();
      }
    }, 1500);

    setTimeout(async () => {
      clearInterval(pollInterval);
      if (!(await isAuthenticated())) {
        btn.disabled = false;
        btn.innerHTML = `<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M12 0C5.4 0 0 5.4 0 12s5.4 12 12 12 12-5.4 12-12S18.66 0 12 0zm5.521 17.34c-.24.359-.66.48-1.021.24-2.82-1.74-6.36-2.101-10.561-1.141-.418.122-.779-.179-.899-.539-.12-.421.18-.78.54-.9 4.56-1.021 8.52-.6 11.64 1.32.42.18.479.659.301 1.02zm1.44-3.3c-.301.42-.841.6-1.262.3-3.239-1.98-8.159-2.58-11.939-1.38-.479.12-1.02-.12-1.14-.6-.12-.48.12-1.021.6-1.141C9.6 9.9 15 10.561 18.72 12.84c.361.181.54.78.241 1.2zm.12-3.36C15.24 8.4 8.82 8.16 5.16 9.301c-.6.179-1.2-.181-1.38-.721-.18-.601.18-1.2.72-1.381 4.26-1.26 11.28-1.02 15.721 1.621.539.3.719 1.02.419 1.56-.299.421-1.02.599-1.559.3z"/></svg> Login with Spotify`;
      }
    }, 30000);
  } catch (e) {
    btn.disabled = false;
    btn.innerHTML = `<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M12 0C5.4 0 0 5.4 0 12s5.4 12 12 12 12-5.4 12-12S18.66 0 12 0zm5.521 17.34c-.24.359-.66.48-1.021.24-2.82-1.74-6.36-2.101-10.561-1.141-.418.122-.779-.179-.899-.539-.12-.421.18-.78.54-.9 4.56-1.021 8.52-.6 11.64 1.32.42.18.479.659.301 1.02zm1.44-3.3c-.301.42-.841.6-1.262.3-3.239-1.98-8.159-2.58-11.939-1.38-.479.12-1.02-.12-1.14-.6-.12-.48.12-1.021.6-1.141C9.6 9.9 15 10.561 18.72 12.84c.361.181.54.78.241 1.2zm.12-3.36C15.24 8.4 8.82 8.16 5.16 9.301c-.6.179-1.2-.181-1.38-.721-.18-.601.18-1.2.72-1.381 4.26-1.26 11.28-1.02 15.721 1.621.539.3.719 1.02.419 1.56-.299.421-1.02.599-1.559.3z"/></svg> Login with Spotify`;
    toast('Failed to open browser: ' + e.message, 'error');
  }
}

// ========== AUTH DONE ==========
async function onAuthDone() {
  showView('view-app');
  await initApp();
}

async function handleLogout() {
  await logout();
  state.playlists = [];
  state.currentSongs = [];
  state.selectedIds = new Set();
  state.currentPlaylistId = null;
  $('playlist-list').innerHTML = '';
  $('songs-list').innerHTML = '';
  $('downloads-list').innerHTML = '';
  showView('view-login');
  toast('Logged out', 'info');
}

// ========== INIT ==========
async function initApp() {
  await loadUserProfile();
  await loadPlaylists();
  navigate('downloads');
}

async function loadUserProfile() {
  const user = await getCurrentUser();
  if (!user) return;
  $('user-name').textContent = user.displayName || user.id;
  if (user.avatarUrl) {
    $('user-avatar').src = user.avatarUrl;
  } else {
    $('user-avatar').style.display = 'none';
  }
}

async function loadPlaylists() {
  const playlists = await getPlaylists();
  state.playlists = playlists || [];
  renderPlaylists();
}

function renderPlaylists() {
  const container = $('playlist-list');
  container.innerHTML = state.playlists.map(p => `
    <div class="playlist-item ${p.id === state.currentPlaylistId ? 'active' : ''}"
         onclick="selectPlaylist('${p.id}')">
      ${p.imageUrl
        ? `<img class="playlist-thumb" src="${p.imageUrl}" alt="" loading="lazy">`
        : `<div class="playlist-thumb-placeholder">🎵</div>`
      }
      <div class="playlist-item-info">
        <div class="playlist-item-name">${escapeHtml(p.name)}</div>
        <div class="playlist-item-meta">${p.trackCount} tracks</div>
      </div>
    </div>
  `).join('');
}

// ========== PLAYLIST SELECTION ==========
async function selectPlaylist(playlistId) {
  state.currentPlaylistId = playlistId;
  state.selectedIds = new Set();
  renderPlaylists();

  const playlist = state.playlists.find(p => p.id === playlistId);
  if (!playlist) return;

  showPage('view-playlist');

  $('playlist-name').textContent = playlist.name;
  $('playlist-image').src = playlist.imageUrl || '';
  $('playlist-desc').textContent = playlist.description || '';
  $('playlist-meta').textContent = `${playlist.owner} · ${playlist.trackCount} songs`;
  $('playlist-loading').classList.remove('hidden');
  $('playlist-songs').classList.add('hidden');
  $('btn-download-all').disabled = true;
  $('btn-download-selected').disabled = true;
  $('select-all').checked = false;

  const songs = await getPlaylistTracks(playlistId);
  state.currentSongs = songs || [];

  $('playlist-loading').classList.add('hidden');
  $('playlist-songs').classList.remove('hidden');
  $('btn-download-all').disabled = false;

  renderSongs();
}

function renderSongs() {
  const container = $('songs-list');
  container.innerHTML = state.currentSongs.map((s, i) => {
    const dl = state.downloads.find(d => d.song && d.song.title === s.title && d.song.artist === s.artist);
    const isDownloading = dl && (dl.status === 'searching' || dl.status === 'downloading' || dl.status === 'converting' || dl.status === 'pending');
    const isComplete = dl && dl.status === 'complete';
    const rowClass = isDownloading ? 'song-row downloading' : isComplete ? 'song-row complete' : 'song-row';
    return `
      <div class="${rowClass}" data-id="${s.id}">
        <div class="col-check">
          <input type="checkbox" ${state.selectedIds.has(s.id) ? 'checked' : ''}
                 onchange="toggleSong('${s.id}')" ${isDownloading ? 'disabled' : ''}>
        </div>
        <div class="col-num">${i + 1}</div>
        <div class="col-title">
          ${s.albumArt ? `<img class="song-album-art" src="${s.albumArt}" alt="" loading="lazy">` : ''}
          <span class="song-title-text">${escapeHtml(s.title)}</span>
        </div>
        <div class="col-artist">${escapeHtml(s.artist)}</div>
        <div class="col-album">${escapeHtml(s.album)}</div>
        <div class="col-duration">${formatDuration(s.duration)}</div>
        <div class="col-action">
          <button class="btn-download-single ${isComplete ? 'downloaded' : ''}"
                  onclick="downloadSong('${s.id}')"
                  title="${isComplete ? 'Downloaded' : 'Download'}"
                  ${isDownloading ? 'disabled' : ''}>
            ${isComplete
              ? '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>'
              : '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>'
            }
          </button>
        </div>
      </div>
    `;
  }).join('');
}

function toggleSong(songId) {
  if (state.selectedIds.has(songId)) {
    state.selectedIds.delete(songId);
  } else {
    state.selectedIds.add(songId);
  }
  updateSelectedButton();
}

function toggleSelectAll() {
  const checked = $('select-all').checked;
  if (checked) {
    state.currentSongs.forEach(s => state.selectedIds.add(s.id));
  } else {
    state.selectedIds.clear();
  }
  renderSongs();
  updateSelectedButton();
}

function updateSelectedButton() {
  $('btn-download-selected').disabled = state.selectedIds.size === 0;
}

// ========== DOWNLOADS ==========
async function downloadSong(songId) {
  const song = state.currentSongs.find(s => s.id === songId);
  if (!song) return;
  await startDownloads([song]);
}

async function downloadAll() {
  await startDownloads(state.currentSongs);
}

async function downloadSelected() {
  const songs = state.currentSongs.filter(s => state.selectedIds.has(s.id));
  if (songs.length === 0) return;
  await startDownloads(songs);
}

async function startDownloads(songs) {
  const json = JSON.stringify(songs.map(s => ({
    id: s.id,
    title: s.title,
    artist: s.artist,
    album: s.album,
    duration: s.duration,
    albumArt: s.albumArt,
    trackNum: s.trackNum,
    playlistId: s.playlistId,
  })));

  await downloadSongs(json);
  toast(`Downloading ${songs.length} song${songs.length > 1 ? 's' : ''}...`, 'success');
  renderSongs();
  navigate('downloads');

  if (!state.polling) {
    state.polling = true;
    pollDownloads();
  }
}

async function pollDownloads() {
  const poll = async () => {
    try {
      const downloads = await getDownloads();
      state.downloads = downloads || [];

      updateDownloadBadge();

      const activeDl = $('downloads-list');
      if (activeDl) renderDownloads();

      renderSongs();

      const hasActive = state.downloads.some(d =>
        d.status === 'pending' || d.status === 'searching' ||
        d.status === 'downloading' || d.status === 'converting'
      );

      if (!hasActive) {
        state.polling = false;
        const completed = state.downloads.filter(d => d.status === 'complete').length;
        const failed = state.downloads.filter(d => d.status === 'failed').length;
        if (completed > 0 && failed === 0) {
          toast(`Downloaded ${completed} song${completed > 1 ? 's' : ''}!`, 'success');
        } else if (failed > 0) {
          toast(`${completed} downloaded, ${failed} failed`, 'error');
        }
        return;
      }

      setTimeout(poll, 1000);
    } catch (e) {
      setTimeout(poll, 2000);
    }
  };

  poll();
}

function updateDownloadBadge() {
  const active = state.downloads.filter(d =>
    d.status === 'pending' || d.status === 'searching' ||
    d.status === 'downloading' || d.status === 'converting'
  ).length;
  const badge = $('download-badge');
  if (active > 0) {
    badge.textContent = active;
    badge.classList.remove('hidden');
  } else {
    badge.classList.add('hidden');
  }
}

function renderDownloads() {
  const container = $('downloads-list');
  if (state.downloads.length === 0) {
    container.innerHTML = '<div class="download-empty"><p>No downloads yet</p></div>';
    return;
  }

  container.innerHTML = state.downloads.map(d => {
    const statusLabel = {
      pending: 'Queued',
      searching: 'Searching YouTube...',
      downloading: 'Downloading...',
      converting: 'Converting...',
      complete: 'Complete',
      failed: 'Failed',
    }[d.status] || d.status;

    return `
      <div class="download-item">
        <div class="download-item-info">
          <div class="download-item-title">${escapeHtml(d.song?.title || '')}</div>
          <div class="download-item-artist">${escapeHtml(d.song?.artist || '')}</div>
        </div>
        <div class="download-progress">
          <div class="progress-bar">
            <div class="progress-bar-fill ${d.status === 'failed' ? 'failed' : d.status === 'complete' ? 'complete' : ''}"
                 style="width: ${d.progress}%"></div>
          </div>
        </div>
        <div class="download-item-status ${d.status}">${statusLabel}</div>
      </div>
    `;
  }).join('');
}

async function clearDownloads() {
  state.downloads = [];
  renderDownloads();
  $('download-badge').classList.add('hidden');
}

// ========== NAVIGATION ==========
function navigate(view) {
  if (view === 'downloads') {
    showPage('view-downloads');
    renderDownloads();
  } else if (view === 'empty') {
    showPage('view-empty');
  }
}

// ========== UTILITY ==========
function escapeHtml(str) {
  if (!str) return '';
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

// ========== INIT ==========
(async function() {
  try {
    const authed = await isAuthenticated();
    if (authed) {
      showView('view-app');
      await initApp();
    }
  } catch (e) {
    console.error('Init error:', e);
  }
})();
