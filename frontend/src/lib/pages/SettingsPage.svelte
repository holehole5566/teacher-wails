<script lang="ts">
  import { onMount } from 'svelte';
  import { GetSettings, SaveSettings, SelectCountdownMusics, GetCountdownMusicData, ValidateRandomPool, GetCurrentVersion, CheckForUpdate, DoUpdate } from '../../../wailsjs/go/main/App';

  let semesterStart = '';
  let dutyGroupSize = 2;
  let dutyStartNumber = 1;
  let lunchGroupSize = 5;
  let lunchStartNumber = 1;
  let mealBucketsStr = '';
  let countdownTimes: string[] = [];
  let periodTimes: string[] = ['','','','','','','',''];
  const periodTimeLabels = ['第1節', '第2節', '第3節', '第4節', '午休', '第5節', '第6節', '第7節'];
  let newTime = '';
  let saved = false;

  type MusicTrack = { path: string; in_random: boolean };
  type CountdownTimeMusic = { time: string; mode: string; index: number };

  let countdownMusics: MusicTrack[] = [];
  let countdownTimeMusicMap: CountdownTimeMusic[] = [];
  let countdownVolume = 0.5;
  let discordWebhook = '';
  let audioOutputDevice = '';
  let audioDevices: { deviceId: string; label: string }[] = [];

  let currentVersion = '';
  let updateStatus: 'idle' | 'checking' | 'available' | 'updating' | 'upToDate' | 'error' = 'idle';
  let latestVersion = '';
  let updateError = '';

  async function refreshAudioDevices() {
    try {
      // Try to get permission to see full device labels
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
        stream.getTracks().forEach(t => t.stop());
      } catch {}
      const devices = await navigator.mediaDevices.enumerateDevices();
      audioDevices = devices
        .filter(d => d.kind === 'audiooutput' && !d.label.startsWith('通訊') && !d.label.startsWith('預設'))
        .map(d => ({ deviceId: d.deviceId, label: d.label || `未知裝置 (${d.deviceId.slice(0, 8)})` }));
    } catch {
      audioDevices = [];
    }
  }
  let testAudios: (HTMLAudioElement | null)[] = [];

  async function loadSettings() {
    const s = await GetSettings();
    semesterStart = s.semester_start_date || '';
    dutyGroupSize = s.duty_group_size || 2;
    dutyStartNumber = s.duty_start_number || 1;
    lunchGroupSize = s.lunch_group_size || 5;
    lunchStartNumber = s.lunch_start_number || 1;
    mealBucketsStr = (s.meal_buckets || []).join('，');
    countdownTimes = (s.countdown_times || []).slice().sort();
    countdownMusics = (s.countdown_musics || []).map((t: any) => ({ path: t.path, in_random: t.in_random }));
    countdownTimeMusicMap = (s.countdown_time_music_map || []).map((m: any) => ({ time: m.time, mode: m.mode || 'random', index: m.index || 0 }));
    countdownVolume = s.countdown_volume > 0 ? s.countdown_volume : 0.5;
    discordWebhook = s.discord_webhook || '';
    audioOutputDevice = s.audio_output_device || '';
    await refreshAudioDevices();
    testAudios = countdownMusics.map(() => null);
    rebuildTimeMusicSettings();
    const pt = s.period_times || [];
    for (let i = 0; i < 8; i++) periodTimes[i] = pt[i] || '';
    periodTimes = periodTimes;
  }

  function addTime() {
    const t = newTime.trim();
    if (t && /^\d{2}:\d{2}$/.test(t) && !countdownTimes.includes(t)) {
      countdownTimes = [...countdownTimes, t].sort();
      newTime = '';
      rebuildTimeMusicSettings();
    }
  }

  function removeTime(t: string) {
    countdownTimes = countdownTimes.filter(x => x !== t);
    syncTimeMusicMap();
  }

  async function addMusics() {
    const paths = await SelectCountdownMusics();
    if (!paths || paths.length === 0) return;
    const existing = new Set(countdownMusics.map(t => t.path));
    const newTracks: MusicTrack[] = paths
      .filter(p => !existing.has(p))
      .map(p => ({ path: p, in_random: true }));
    countdownMusics = [...countdownMusics, ...newTracks];
    testAudios = countdownMusics.map(() => null);
  }

  function removeTrack(i: number) {
    stopTrackAudio(i);
    countdownMusics = countdownMusics.filter((_, idx) => idx !== i);
    testAudios = countdownMusics.map(() => null);
    countdownTimeMusicMap = countdownTimeMusicMap.map(m => ({
      ...m,
      index: m.index > i ? m.index - 1 : m.index === i ? 0 : m.index
    }));
  }

  function stopTrackAudio(i: number) {
    if (testAudios[i]) { testAudios[i]!.pause(); testAudios[i] = null; testAudios = [...testAudios]; }
  }

  async function previewTrack(i: number) {
    if (testAudios[i]) { stopTrackAudio(i); return; }
    try {
      const url = await GetCountdownMusicData(i);
      if (!url) return;
      const a = new Audio(url);
      a.volume = countdownVolume;
      const deviceId = audioOutputDevice || 'default';
      try { await (a as any).setSinkId(deviceId); } catch {}
      a.onended = () => { testAudios[i] = null; testAudios = [...testAudios]; };
      testAudios[i] = a;
      testAudios = [...testAudios];
      a.play().catch(() => {});
    } catch (e: any) {
      alert('無法播放：' + (e?.message || '檔案可能已被移動或刪除'));
    }
  }

  // Reactive map: time string → { mode, index } for instant template reactivity
  let timeMusicSettings: Record<string, { mode: string; index: number }> = {};

  function rebuildTimeMusicSettings() {
    const map: Record<string, { mode: string; index: number }> = {};
    for (const t of countdownTimes) {
      const entry = countdownTimeMusicMap.find(m => m.time === t);
      map[t] = entry ? { mode: entry.mode, index: entry.index } : { mode: 'random', index: 0 };
    }
    timeMusicSettings = map;
  }

  function setTimeMusicMode(t: string, mode: string) {
    timeMusicSettings[t] = { ...timeMusicSettings[t], mode };
    timeMusicSettings = { ...timeMusicSettings };
    syncMapFromSettings();
  }

  function setTimeMusicIndex(t: string, index: number) {
    timeMusicSettings[t] = { ...timeMusicSettings[t], index };
    timeMusicSettings = { ...timeMusicSettings };
    syncMapFromSettings();
  }

  function syncMapFromSettings() {
    countdownTimeMusicMap = Object.entries(timeMusicSettings).map(([time, v]) => ({
      time, mode: v.mode, index: v.index
    }));
  }

  function syncTimeMusicMap() {
    countdownTimeMusicMap = countdownTimeMusicMap.filter(m => countdownTimes.includes(m.time));
    rebuildTimeMusicSettings();
  }

  function onTimeMusicModeChange(t: string, e: Event) {
    setTimeMusicMode(t, (e.target as HTMLSelectElement).value);
  }

  function onTimeMusicIndexChange(t: string, e: Event) {
    setTimeMusicIndex(t, Number((e.target as HTMLSelectElement).value));
  }

  async function handleSave() {
    const buckets = mealBucketsStr
      .split(/[,，]/)
      .map((b: string) => b.trim())
      .filter((b: string) => b);

    if (buckets.length > 0 && buckets.length !== lunchGroupSize) {
      if (!confirm(`餐桶數量（${buckets.length}）與抬餐每組人數（${lunchGroupSize}）不一致，確定要儲存嗎？`)) {
        return;
      }
    }

    const hasRandomMode = countdownMusics.length > 0 && (
      countdownTimes.some(t => {
        const entry = countdownTimeMusicMap.find(m => m.time === t);
        return !entry || entry.mode === 'random';
      })
    );
    if (hasRandomMode) {
      const valid = await ValidateRandomPool();
      if (!valid) {
        alert('請至少將一首音樂加入隨機清單');
        return;
      }
    }

    await SaveSettings({
      semester_start_date: semesterStart,
      duty_group_size: dutyGroupSize,
      duty_start_number: dutyStartNumber,
      lunch_group_size: lunchGroupSize,
      lunch_start_number: lunchStartNumber,
      meal_buckets: buckets,
      auto_start: false,
      countdown_times: countdownTimes,
      period_times: periodTimes,
      countdown_volume: countdownVolume,
      countdown_musics: countdownMusics,
      countdown_time_music_map: countdownTimeMusicMap,
      discord_webhook: discordWebhook,
      audio_output_device: audioOutputDevice,
    });
    saved = true;
    setTimeout(() => { saved = false; }, 2000);
  }

  async function checkUpdate() {
    updateStatus = 'checking';
    updateError = '';
    try {
      const result = await CheckForUpdate();
      if (result.has_update) {
        updateStatus = 'available';
        latestVersion = result.latest_version;
      } else {
        updateStatus = 'upToDate';
      }
    } catch (e: any) {
      updateStatus = 'error';
      updateError = e?.message || '檢查更新失敗';
    }
  }

  async function doUpdate() {
    updateStatus = 'updating';
    try {
      await DoUpdate();
    } catch (e: any) {
      updateStatus = 'error';
      updateError = e?.message || '更新失敗';
    }
  }

  onMount(async () => {
    await loadSettings();
    currentVersion = await GetCurrentVersion();
  });
</script>

<div class="page">
  <h2 class="page-title">設定</h2>

  <div class="card settings-form">
    <div class="form-group">
      <label>學期開始日期</label>
      <input type="date" bind:value={semesterStart} />
    </div>

    <div class="form-row">
      <div class="form-group">
        <label>值日每組人數</label>
        <input type="number" min="1" bind:value={dutyGroupSize} />
      </div>
      <div class="form-group">
        <label>值日起始座號</label>
        <input type="number" min="1" bind:value={dutyStartNumber} />
      </div>
    </div>

    <div class="form-row">
      <div class="form-group">
        <label>抬餐每組人數</label>
        <input type="number" min="1" bind:value={lunchGroupSize} />
      </div>
      <div class="form-group">
        <label>抬餐起始座號</label>
        <input type="number" min="1" bind:value={lunchStartNumber} />
      </div>
    </div>

    <div class="form-group">
      <label>餐桶名稱（以逗號分隔）</label>
      <input type="text" bind:value={mealBucketsStr} placeholder="飯，菜1，菜2，湯，餐具" />
    </div>

    <div class="form-group">
      <label>每節上課時間（用於展示模式判斷當前節次）</label>
      <div class="period-grid">
        {#each periodTimes as _, i}
          <div class="period-input">
            <span class="period-label">{periodTimeLabels[i]}</span>
            <input type="time" bind:value={periodTimes[i]} />
          </div>
        {/each}
      </div>
    </div>

    <div class="form-group">
      <label>上課倒數時間點（到達該時間前 1 分鐘開始倒數）</label>
      <div class="inline-form">
        <input type="time" bind:value={newTime} />
        <button class="btn-primary btn-sm" on:click={addTime}>新增</button>
      </div>
      {#if countdownTimes.length > 0}
        <div class="time-list">
          {#each countdownTimes as t}
            <div class="time-row">
              <span class="time-tag">
                {t}
                <button class="tag-remove" on:click={() => removeTime(t)}>×</button>
              </span>
              {#if countdownMusics.length > 0 && timeMusicSettings[t]}
                <select
                  class="mode-select"
                  value={timeMusicSettings[t].mode}
                  on:change={e => onTimeMusicModeChange(t, e)}
                >
                  <option value="random">隨機</option>
                  <option value="index">指定</option>
                  <option value="none">不播放</option>
                </select>
                {#if timeMusicSettings[t].mode === 'index'}
                  <select
                    class="track-select"
                    value={timeMusicSettings[t].index}
                    on:change={e => onTimeMusicIndexChange(t, e)}
                  >
                    {#each countdownMusics as track, i}
                      <option value={i}>{track.path.split(/[/\\]/).pop()}</option>
                    {/each}
                  </select>
                {/if}
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <div class="form-group">
      <label>倒數音樂清單</label>
      <div class="inline-form" style="margin-bottom:8px">
        <button class="btn-primary btn-sm" on:click={addMusics}>＋ 新增音樂</button>
      </div>
      {#if countdownMusics.length > 0}
        <div class="music-list">
          {#each countdownMusics as track, i}
            <div class="music-row">
              <span class="music-name">{track.path.split(/[/\\]/).pop()}</span>
              <label class="random-toggle">
                <input type="checkbox" bind:checked={countdownMusics[i].in_random} />
                加入隨機
              </label>
              <button class="btn-sm btn-test" on:click={() => previewTrack(i)}>
                {testAudios[i] ? '⏹' : '▶'}
              </button>
              <button class="btn-sm btn-danger" on:click={() => removeTrack(i)}>✕</button>
            </div>
          {/each}
        </div>
        <div class="volume-row">
          <span class="volume-label">🔈</span>
          <input type="range" min="0" max="1" step="0.05" bind:value={countdownVolume} />
          <span class="volume-label">🔊</span>
          <span class="volume-value">{Math.round(countdownVolume * 100)}%</span>
        </div>
      {:else}
        <p class="empty-hint">尚未加入任何音樂</p>
      {/if}
    </div>

    <div class="form-group">
      <label>音訊輸出裝置</label>
      <div class="inline-form">
        <select bind:value={audioOutputDevice} style="flex:1">
          <option value="">跟隨系統預設（每次播放時刷新）</option>
          {#each audioDevices as device}
            <option value={device.deviceId}>{device.label}</option>
          {/each}
        </select>
        <button class="btn-primary btn-sm" on:click={refreshAudioDevices}>重新偵測</button>
      </div>
    </div>

    <div class="form-group">
      <label>Discord Webhook（錯誤通知用）</label>
      <input type="text" bind:value={discordWebhook} placeholder="https://discord.com/api/webhooks/..." />
    </div>

    <div class="form-actions">
      <button class="btn-primary save-btn" on:click={handleSave}>儲存設定</button>
      {#if saved}
        <span class="save-ok">已儲存</span>
      {/if}
    </div>
  </div>

  <div class="card update-section">
    <h3 class="section-title">軟體更新</h3>
    <div class="update-row">
      <span class="version-info">目前版本：{currentVersion || '載入中...'}</span>
      {#if updateStatus === 'idle'}
        <button class="btn-primary btn-sm" on:click={checkUpdate}>檢查更新</button>
      {:else if updateStatus === 'checking'}
        <span class="update-msg">檢查中...</span>
      {:else if updateStatus === 'upToDate'}
        <span class="update-msg update-ok">已是最新版本</span>
        <button class="btn-primary btn-sm" on:click={checkUpdate}>重新檢查</button>
      {:else if updateStatus === 'available'}
        <span class="update-msg">有新版本：{latestVersion}</span>
        <button class="btn-primary btn-sm" on:click={doUpdate}>立即更新</button>
      {:else if updateStatus === 'updating'}
        <span class="update-msg">更新中，請稍候...</span>
      {:else if updateStatus === 'error'}
        <span class="update-msg update-err">{updateError}</span>
        <button class="btn-primary btn-sm" on:click={checkUpdate}>重試</button>
      {/if}
    </div>
  </div>
</div>

<style>
  .settings-form {
    max-width: 560px;
  }
  .form-row {
    display: flex;
    gap: 16px;
  }
  .form-row .form-group {
    flex: 1;
  }
  .form-actions {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-top: 16px;
  }
  .save-btn {
    padding: 12px 32px;
    font-size: 16px;
    font-weight: 600;
  }
  .save-ok {
    color: var(--success);
    font-size: 13px;
    font-weight: 500;
  }
  .period-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: 6px;
  }
  .period-input {
    display: flex;
    align-items: center;
    gap: 4px;
    background: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 4px 8px;
  }
  .period-label {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-secondary);
    white-space: nowrap;
  }
  .period-input input {
    width: 110px;
    padding: 4px 6px;
    font-size: 12px;
    border: none;
    background: transparent;
  }
  .time-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-top: 8px;
  }
  .time-tag {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    background: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: 20px;
    padding: 4px 10px;
    font-size: 13px;
    font-weight: 500;
  }
  .tag-remove {
    background: none;
    border: none;
    padding: 0 2px;
    font-size: 16px;
    color: var(--danger);
    cursor: pointer;
    line-height: 1;
  }
  .tag-remove:hover {
    color: var(--danger-hover);
  }
  .music-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-bottom: 10px;
  }
  .music-row {
    display: flex;
    align-items: center;
    gap: 8px;
    background: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 6px 10px;
  }
  .music-name {
    flex: 1;
    font-size: 13px;
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .random-toggle {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
    white-space: nowrap;
    cursor: pointer;
  }
  .empty-hint {
    font-size: 13px;
    color: var(--text-secondary);
    margin: 4px 0;
  }
  .time-row {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
    background: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 6px 10px;
  }
  .mode-select, .track-select {
    font-size: 12px;
    padding: 2px 6px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: var(--bg-primary);
    cursor: pointer;
  }
  .track-select {
    max-width: 160px;
  }
  .btn-danger {
    background: var(--danger);
    color: white;
    border: none;
    border-radius: 6px;
    cursor: pointer;
  }
  .btn-danger:hover {
    background: var(--danger-hover);
  }
  .btn-test {
    background: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: 6px;
    cursor: pointer;
    font-size: 12px;
  }
  .btn-test:hover {
    background: var(--border);
  }
  .volume-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 8px;
  }
  .volume-row input[type="range"] {
    flex: 1;
    max-width: 200px;
  }
  .volume-label {
    font-size: 14px;
  }
  .volume-value {
    font-size: 12px;
    color: var(--text-secondary);
    min-width: 36px;
  }
  .update-section {
    max-width: 560px;
    margin-top: 16px;
  }
  .section-title {
    font-size: 15px;
    font-weight: 600;
    margin: 0 0 10px;
  }
  .update-row {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }
  .version-info {
    font-size: 13px;
    color: var(--text-secondary);
  }
  .update-msg {
    font-size: 13px;
  }
  .update-ok {
    color: var(--success);
  }
  .update-err {
    color: var(--danger);
  }
</style>
