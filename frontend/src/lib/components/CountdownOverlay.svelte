<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { GetActiveCountdownMusicData, GetSettings, ReportError, DebugLog } from '../../../wailsjs/go/main/App';

  export let seconds: number = 60;
  export let triggerTime: string = '';
  export let onFinished: () => void = () => {};

  let remaining = seconds;
  let audio: HTMLAudioElement | null = null;

  const interval = setInterval(() => {
    remaining--;
    if (remaining <= 0) {
      clearInterval(interval);
      stopMusic();
      onFinished();
    }
  }, 1000);

  function stopMusic() {
    if (audio) {
      audio.pause();
      audio = null;
    }
  }

  onMount(async () => {
    DebugLog(`[Countdown] onMount triggered, triggerTime=${triggerTime}, seconds=${seconds}`);
    try {
      DebugLog(`[Countdown] Calling GetActiveCountdownMusicData...`);
      const [dataUrl, settings] = await Promise.all([
        GetActiveCountdownMusicData(triggerTime),
        GetSettings()
      ]);
      DebugLog(`[Countdown] GetActiveCountdownMusicData returned, dataUrl length=${dataUrl?.length || 0}`);
      if (dataUrl) {
        audio = new Audio(dataUrl);
        const vol = settings.countdown_volume > 0 ? settings.countdown_volume : 0.5;
        audio.volume = vol;

        // Set audio output device
        const deviceId = settings.audio_output_device || 'default';
        try {
          await (audio as any).setSinkId(deviceId);
          DebugLog(`[Countdown] setSinkId("${deviceId}") succeeded`);
        } catch (e: any) {
          DebugLog(`[Countdown] setSinkId("${deviceId}") failed: ${e?.message || e}, using default`);
        }

        // Log audio device info
        const sinkId = (audio as any).sinkId ?? 'unknown';
        DebugLog(`[Countdown] Audio created, volume=${vol}, sinkId="${sinkId}"`);
        try {
          const devices = await navigator.mediaDevices.enumerateDevices();
          const outputs = devices.filter(d => d.kind === 'audiooutput');
          DebugLog(`[Countdown] Available outputs: ${outputs.map(d => `${d.label}(${d.deviceId.slice(0,8)})`).join(', ')}`);
        } catch {}

        audio.addEventListener('loadedmetadata', () => {
          if (audio && audio.duration < 60) {
            audio.loop = true;
            DebugLog(`[Countdown] Audio duration=${audio.duration.toFixed(1)}s < 60s, looping enabled`);
          }
        });

        DebugLog(`[Countdown] Attempting play...`);
        audio.play().then(() => {
          DebugLog(`[Countdown] Audio play() succeeded, sinkId="${(audio as any)?.sinkId ?? 'unknown'}"`);
        }).catch((e: any) => {
          DebugLog(`[Countdown] Audio play() FAILED: ${e?.message || e}`);
          ReportError(`音樂播放失敗（triggerTime=${triggerTime}）：${e?.message || e}`);
        });
      } else {
        DebugLog(`[Countdown] dataUrl is empty, no music to play`);
      }
    } catch (e: any) {
      DebugLog(`[Countdown] CATCH error: ${e?.message || e}`);
      ReportError(`倒數音樂載入失敗（triggerTime=${triggerTime}）：${e?.message || e}`);
    }
  });

  onDestroy(() => {
    clearInterval(interval);
    stopMusic();
  });

  $: minutes = Math.floor(remaining / 60);
  $: secs = remaining % 60;
  $: display = minutes > 0
    ? `${minutes}:${String(secs).padStart(2, '0')}`
    : `${secs}`;
  $: progress = remaining / seconds;
</script>

<div class="overlay">
  <div class="countdown-container">
    <div class="ring-wrapper">
      <svg viewBox="0 0 200 200" class="ring">
        <circle cx="100" cy="100" r="90" class="ring-bg" />
        <circle
          cx="100" cy="100" r="90"
          class="ring-progress"
          style="stroke-dashoffset: {565.48 * (1 - progress)}"
        />
      </svg>
      <span class="time">{display}</span>
    </div>
    <p class="label">上課倒數</p>
    <p class="sublabel">請回到座位準備上課</p>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    z-index: 99999;
    background: rgba(15, 23, 42, 0.97);
    display: flex;
    align-items: center;
    justify-content: center;
    user-select: none;
    cursor: default;
  }
  .countdown-container {
    text-align: center;
    color: white;
  }
  .ring-wrapper {
    position: relative;
    width: 280px;
    height: 280px;
    margin: 0 auto 32px;
  }
  .ring {
    width: 100%;
    height: 100%;
    transform: rotate(-90deg);
  }
  .ring-bg {
    fill: none;
    stroke: rgba(255, 255, 255, 0.1);
    stroke-width: 8;
  }
  .ring-progress {
    fill: none;
    stroke: #3b82f6;
    stroke-width: 8;
    stroke-linecap: round;
    stroke-dasharray: 565.48;
    transition: stroke-dashoffset 1s linear;
  }
  .time {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 80px;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    letter-spacing: -2px;
  }
  .label {
    font-size: 28px;
    font-weight: 600;
    margin-bottom: 8px;
  }
  .sublabel {
    font-size: 16px;
    opacity: 0.6;
  }
</style>
